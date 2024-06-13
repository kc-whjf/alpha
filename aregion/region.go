package aregion

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/kc-whjf/alpha/aconfig"
	"github.com/kc-whjf/alpha/alog"
	"github.com/kc-whjf/alpha/httpclient"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"strings"
	"sync"
	"time"
)

type RegionResult struct {
	Status   int       `json:"status,omitempty"`
	Message  string    `json:"message,omitempty"`
	ErrorMsg string    `json:"errorMsg,omitempty"`
	Regions  []*Region `json:"result,omitempty"`
}

type Region struct {
	RegionCode string `json:"regionCode,omitempty"`
	RegionName string `json:"regionName,omitempty"`
	Status     int    `json:"status,omitempty"`
	OpsEnv     string `json:"opsEnv,omitempty"`
	OpsArch    string `json:"opsArch,omitempty"`
	Default    bool   `json:"default,omitempty"`
	PropMap    Prop   `json:"propMap,omitempty"`
	AzList     []*Az  `json:"azList,omitempty"`
}

type Az struct {
	AzCode  string `json:"azCode,omitempty"`
	AzName  string `json:"azName,omitempty"`
	Status  int    `json:"status,omitempty"`
	Default bool   `json:"default,omitempty"`
	PropMap Prop   `json:"propMap,omitempty"`
}

type Prop map[string]interface{}

func (p Prop) GetString(key string) string {
	return cast.ToString(p[key])
}

func (p Prop) GetStringIgnoreCase(key string) string {
	for k, v := range p {
		if strings.EqualFold(k, key) {
			return cast.ToString(v)
		}
	}
	return ""
}

var (
	crossRegionUrl string
	config         *Config
	regionList     []*Region // cache region list
	commonResty    *resty.Client
	lock           sync.Mutex
)

func InitCrossRegionConfig(application *aconfig.Application) error {
	// config cross_region
	crossRegion := application.GetSecondaryPort("cross_region")
	if crossRegion == nil {
		return errors.New("未找到 cross_region 相关配置")
	}

	if alog.Sugar == nil {
		return errors.New("请先调用 alog.InitLogger(...) 初始化日志组件")
	}

	location := crossRegion.MatchedPrimaryPort.Location
	address := location.Address
	if !strings.HasPrefix(address, "http://") && !strings.HasPrefix(address, "https://") {
		address = "http://" + address
	}

	crossRegionUrl = fmt.Sprintf("%s:%d/apis/ops-console.i/cross-region/proxy", address, location.Port)

	c := &Config{}
	if err := crossRegion.Options.LoadTo(c); err != nil {
		return err
	}
	config = c

	commonResty = httpclient.NewResty(alog.Sugar).
		SetRetryCount(3).
		SetRetryWaitTime(3 * time.Second).
		SetRetryMaxWaitTime(5 * time.Second)

	go syncRegionListBackground()
	return nil
}

// get operation region list
func GetRegionList(ctx context.Context) (list []*Region, err error) {
	lock.Lock()
	defer lock.Unlock()

	//cache
	if regionList != nil {
		return regionList, nil
	}

	if regionList, err = getRegionListInternal(ctx); err != nil {
		return
	}

	return regionList, nil
}

func ClearCache() {
	lock.Lock()
	defer lock.Unlock()

	regionList = nil
}

func getRegionListInternal(ctx context.Context) (list []*Region, err error) {
	var authKey string
	if err = httpclient.ExecuteHttp(ctx, commonResty,
		&httpclient.HttpParams{Method: "GET", Url: normalizeUrl(config.AuthServer.Location)},
		&authKey); err != nil {
		err = errors.Wrap(err, "获取 auth key 失败")
		return
	}

	if len(authKey) == 0 {
		err = errors.New("获取auth key 为空")
		return
	}
	alog.CtxSugar(ctx).Infof("获取的auth key为: %s", authKey)

	token := md5.Sum([]byte(config.RegionServer.Location.SK + "." + authKey))
	params2 := &httpclient.HttpParams{
		Method: "GET",
		Url:    normalizeUrl(config.RegionServer.Location),
		Headers: map[string]string{
			"X-AUTH-APP":   config.RegionServer.Location.AK,
			"X-AUTH-TOKEN": hex.EncodeToString(token[:]),
		},
	}
	var result RegionResult
	if err = httpclient.ExecuteHttp(ctx, commonResty, params2, &result); err != nil {
		err = errors.Wrap(err, "获取 region 列表失败")
		return
	}
	if result.Status != 200 {
		err = errors.Errorf("获取 region 列表失败，原因为: %s %s", result.Message, result.ErrorMsg)
		return
	}

	for _, r := range result.Regions {
		if r.PropMap != nil {
			r.OpsEnv = r.PropMap.GetStringIgnoreCase("opsEnv")
			r.OpsArch = r.PropMap.GetStringIgnoreCase("opsArch")
			r.Default = r.PropMap.GetStringIgnoreCase("default") == "1"
		}
		for _, az := range r.AzList {
			if az.PropMap != nil {
				az.Default = r.PropMap.GetStringIgnoreCase("default") == "1"
			}
		}
	}

	return result.Regions, nil
}

func syncRegionListBackground() {
	ctx := context.Background()
	defer func() {
		if err := recover(); err != nil {
			alog.CtxSugar(ctx).Errorf("同步 region 列表失败, %+v", err)
		}
	}()

	t := time.NewTicker(time.Hour)
	for range t.C {
		regions, err := getRegionListInternal(ctx)
		if err != nil {
			alog.CtxSugar(ctx).Errorf("同步 region 列表失败, %+v", err)
		} else {
			lock.Lock()
			regionList = regions
			lock.Unlock()
			alog.CtxSugar(ctx).Info("同步 region 列表成功")
		}
	}
}

func normalizeUrl(loc *Location) string {
	address := loc.Address
	if !strings.HasPrefix(address, "http://") && !strings.HasPrefix(address, "https://") {
		address = "http://" + address
	}
	return fmt.Sprintf("%s:%d%s", address, loc.Port, loc.Path)
}
