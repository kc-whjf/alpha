package aregion

import (
	"github.com/go-resty/resty/v2"
	"github.com/kc-whjf/alpha/aconfig"
	"github.com/kc-whjf/alpha/httpclient"
)

func NewRestyWith(portName aconfig.PortName, appConfig *aconfig.Application, protocol string) (*resty.Client, error) {
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
		return
	}

	req.Header.Add("X-Ops-Raw-Url", req.URL)
	//rewrite to cross region server
	req.URL = crossRegionUrl
	return nil
}
