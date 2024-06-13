package aregion

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/kc-whjf/alpha/aconfig"
	"github.com/kc-whjf/alpha/alog"
	"github.com/kc-whjf/alpha/httpclient"
	"strings"
)

func NewRestyWith(portName aconfig.PortName, appConfig *aconfig.Application, protocol string) (*resty.Client, error) {
	location := appConfig.GetMatchedPrimaryPortLocation(portName)
	if location == nil {
		return nil, fmt.Errorf("missing matched primaryport location (port_name: %s)", portName)
	}

	client := httpclient.NewResty(alog.Sugar)
	// Eliminat the interference of protocol in address
	address := strings.TrimPrefix(location.Address, "http://")
	address = strings.TrimPrefix(address, "https://")
	hostURL := fmt.Sprintf("%s%s:%d", protocol, address, location.Port)
	client.SetHostURL(hostURL)
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

func AddCrossRegionMiddleware(client *resty.Client) {
	client.OnBeforeRequest(crossRegion)
}

func crossRegion(c *resty.Client, req *resty.Request) (err error) {
	opsRegion := req.Header.Get("X-Ops-Region")
	if len(opsRegion) == 0 {
		return
	}

	req.Header.Add("X-Ops-Raw-Url", req.URL)
	//rewrite to ops-console
	req.URL = crossRegionUrl
	return nil
}
