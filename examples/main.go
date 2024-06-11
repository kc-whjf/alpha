package main

import (
	"fmt"

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

func main() {
	fmt.Println("Hello world")
}
