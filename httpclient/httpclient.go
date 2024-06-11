package httpclient

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/kc-whjf/alpha/aconfig"
	"github.com/kc-whjf/alpha/alog"
	"go.uber.org/zap"
)

const (
	defaultTimeout = 10 * time.Second
)

func NewResty(sugar *zap.SugaredLogger) *resty.Client {
	client := resty.New()

	client.SetTimeout(defaultTimeout)
	client.SetLogger(NewLogger(sugar))

	client.OnAfterResponse(logRequestMiddleware())

	return client
}

func NewRestyWith(portName aconfig.PortName, appConfig *aconfig.Application, protocol string) (*resty.Client, error) {
	location := appConfig.GetMatchedPrimaryPortLocation(portName)
	if location == nil {
		return nil, fmt.Errorf("missing matched primaryport location (port_name: %s)", portName)
	}

	client := NewResty(alog.Sugar)
	// Eliminat the interference of protocol in address
	address := strings.TrimPrefix(location.Address, "http://")
	address = strings.TrimPrefix(address, "https://")
	hostURL := fmt.Sprintf("%s%s:%d", protocol, address, location.Port)
	client.SetHostURL(hostURL)

	return client, nil
}

func MustNewRestyWith(portName aconfig.PortName, appConfig *aconfig.Application, protocol string) *resty.Client {
	client, err := NewRestyWith(portName, appConfig, protocol)
	if err != nil {
		panic(err)
	}

	return client
}
