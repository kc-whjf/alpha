package aregion

import (
	"github.com/go-resty/resty/v2"
	"github.com/kc-whjf/alpha/aconfig"
	"github.com/kc-whjf/alpha/httpclient"
	"github.com/pkg/errors"
	"net/url"
)

func NewRestyWith(portName aconfig.PortName, appConfig *aconfig.Application, protocol string) (*resty.Client, error) {
	if len(crossRegionUrl) == 0 {
		return nil, errors.New("请先调用 aregion.InitCrossRegionConfig(...) 初始化跨越调用组件")
	}

	client, err := httpclient.NewRestyWith(portName, appConfig, protocol)
	if err != nil {
		return nil, err
	}

	AddCrossRegionMiddleware(client)

	return client, nil
}

func MustNewRestyWith(portName aconfig.PortName, appConfig *aconfig.Application, protocol string) *resty.Client {
	client, err := NewRestyWith(portName, appConfig, protocol)
	if err != nil {
		panic(err)
	}

	return client
}

func AddCrossRegionMiddleware(client *resty.Client) *resty.Client {
	return client.OnBeforeRequest(crossRegionMiddleware)
}

func crossRegionMiddleware(c *resty.Client, req *resty.Request) (err error) {
	opsRegion := req.Header.Get("X-Ops-Region")
	if len(opsRegion) == 0 {
		err = errors.New("未获取到 X-Ops-Region 请求头")
		return
	}

	rawUrl := req.URL
	reqUrl, err := url.Parse(req.URL)
	if err != nil {
		return
	}
	if !reqUrl.IsAbs() {
		if len(rawUrl) > 0 && rawUrl[0] != '/' {
			rawUrl = "/" + rawUrl
		}
		rawUrl = c.HostURL + rawUrl
	}
	req.Header.Add("X-Ops-Raw-Url", rawUrl)
	//rewrite to cross region server
	req.URL = crossRegionUrl
	return nil
}
