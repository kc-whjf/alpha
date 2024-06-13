package main

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/kc-whjf/alpha/aconfig"
	"github.com/kc-whjf/alpha/alog"
	"github.com/kc-whjf/alpha/aregion"
	"github.com/kc-whjf/alpha/autil/acrypto/pbe"
	"github.com/kc-whjf/alpha/ginwrapper"
	"regexp"
	"time"

	_ "github.com/kc-whjf/alpha/aconfig"
	_ "github.com/kc-whjf/alpha/aerror"
	_ "github.com/kc-whjf/alpha/alog"
	_ "github.com/kc-whjf/alpha/alog/gormwrapper"
	_ "github.com/kc-whjf/alpha/autil"
	_ "github.com/kc-whjf/alpha/autil/ahttp"
	_ "github.com/kc-whjf/alpha/autil/ahttp/request"
	_ "github.com/kc-whjf/alpha/database"
	_ "github.com/kc-whjf/alpha/ginwrapper"
	_ "github.com/kc-whjf/alpha/httpclient"
	_ "github.com/kc-whjf/alpha/httpserver/rsp"
)

var Ctx = context.Background()

func main() {
	fmt.Println("Hello world")

	serverConfig, err := InitDefaultConfig("examples/config.yaml")
	if err != nil {
		panic(err)
	}

	if err = alog.InitLogger(
		serverConfig.ThisApplication.Name,
		serverConfig.CustomConfig.Common.Log.Directory,
		serverConfig.CustomConfig.Common.Log.Level,
		serverConfig.CustomConfig.Common.Log.Format); err != nil {
		panic(err)
	}

	if err = aregion.InitCrossRegionConfig(serverConfig.ThisApplication); err != nil {
		panic(err)
	}

	go func() {
		for range time.Tick(1 * time.Minute) {
			list, _ := aregion.GetRegionList(Ctx)
			alog.CtxSugar(Ctx).Info("region count:", len(list))
		}
	}()

	apiServer, err := NewAPIServer(serverConfig)
	if err != nil {
		panic(err)
	}
	if err = apiServer.Prepare(); err != nil {
		panic(err)
	}

	if err = apiServer.Run(); err != nil {
		panic(err)
	}
}

// -------- server --------

type APIServer struct {
	serverConfig *ServerConfig
	engine       *gin.Engine
}

func NewAPIServer(config *ServerConfig) (*APIServer, error) {
	alog.Sugar.Infow("showing configuration", "server_config", config)

	return &APIServer{
		config,
		ginwrapper.New(&ginwrapper.Options{}),
	}, nil
}

func (s *APIServer) Prepare() error {
	return s.InstallAPI()
}

func (s *APIServer) Run() error {
	serving := s.serverConfig.CustomConfig.Serving
	addr := fmt.Sprintf("%s:%d", serving.Address, serving.Port)
	return s.engine.Run(addr)
}

func (s *APIServer) InstallAPI() error {
	s.engine.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"hello": "world"})
	})
	return nil
}

// -------- config --------
var (
	originalServerConfig *ServerConfig
	defaultServerConfig  *ServerConfig
)

type ServerConfig struct {
	ConfigFile      string
	ThisApplication *aconfig.Application
	CustomConfig    *CustomConfig
}

func (c *ServerConfig) Validate() error {
	// Add validations here
	return nil
}

func InitDefaultConfig(configFile string) (*ServerConfig, error) {
	var err error
	defaultServerConfig, err = newServerConfig(configFile)
	if err != nil {
		return nil, err
	}
	if err = defaultServerConfig.Validate(); err != nil {
		return nil, err
	}

	defaultServerConfig.CustomConfig.Common.Complete(defaultServerConfig.ThisApplication.Name)

	return defaultServerConfig, nil
}

func newServerConfig(configFile string) (*ServerConfig, error) {
	thisApplication, err := aconfig.New(configFile)
	if err != nil {
		return nil, err
	}

	var customConfig = &CustomConfig{}
	if err = thisApplication.GetCustomConfig().LoadTo(customConfig); err != nil {
		return nil, err
	}
	originalServerConfig = &ServerConfig{
		ConfigFile:      configFile,
		ThisApplication: thisApplication,
		CustomConfig:    customConfig,
	}

	funcs := []aconfig.PreProcessFunc{}
	if customConfig.Common.Encryptor.S != "" {
		funcs = append(funcs, pbeWithMD5AndDES_Decrypt(customConfig.Common.Encryptor))
	}
	if len(funcs) > 0 {
		thisApplication, err = aconfig.New(configFile, funcs...)
		if err != nil {
			return nil, err
		}
		if err = thisApplication.GetCustomConfig().LoadTo(customConfig); err != nil {
			return nil, err
		}
	}

	return &ServerConfig{
		ConfigFile:      configFile,
		ThisApplication: thisApplication,
		CustomConfig:    customConfig,
	}, nil
}

type Serving struct {
	Address string `json:"address,omitempty"`
	Port    int    `json:"port,omitempty"`
}

type CustomConfig struct {
	Serving Serving        `json:"serving,omitempty"`
	Common  aconfig.Common `json:"common,omitempty"`
}

func pbeWithMD5AndDES_Decrypt(encryptor aconfig.Encryptor) aconfig.PreProcessFunc {
	return func(data []byte) ([]byte, error) {
		expr := `ENC\(([A-Za-z\d+/=]+)\)`
		re, err := regexp.Compile(expr)
		if err != nil {
			return nil, err
		}
		ret := re.ReplaceAllFunc(data, _pbeWithMD5AndDES_Decrypt(encryptor.S, re))
		return ret, nil
	}
}

func _pbeWithMD5AndDES_Decrypt(password string, re *regexp.Regexp) func([]byte) []byte {
	return func(s []byte) []byte {
		ret := re.FindStringSubmatch(string(s))
		if len(ret) > 1 {
			decrypt, err := pbe.PBEWithMD5AndDES_Decrypt(ret[1], password)
			if err != nil {
				panic("pbe with md5 and des decrypt err: " + err.Error())
			}
			return []byte(decrypt)
		}
		return s
	}
}
