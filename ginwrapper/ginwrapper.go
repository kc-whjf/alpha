package ginwrapper

import (
	"github.com/kc-whjf/alpha/aregion"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kc-whjf/alpha/alog"
)

func defaultHealthHandler(c *gin.Context) {
	c.Header("Content-type", "application/json; charset=utf-8")
	c.String(200, `{"status": 200, "message": "success"}`)
}

type Options struct {
	// LivezHandler checks if the app is alive
	LivezHandler func(c *gin.Context)
	// ReadyzHandler checks if the app is ready for incoming requests
	ReadyzHandler func(c *gin.Context)
	// ConfigzHandler checks if the app's config is working
	ConfigzHandler func(c *gin.Context)
}

func (o *Options) complete() {
	if o.LivezHandler == nil {
		o.LivezHandler = defaultHealthHandler
	}
	if o.ReadyzHandler == nil {
		o.ReadyzHandler = defaultHealthHandler
	}
	if o.ConfigzHandler == nil {
		o.ConfigzHandler = defaultHealthHandler
	}
}

func New(options *Options) *gin.Engine {
	if options == nil {
		options = &Options{}
	}
	options.complete()

	r := gin.New()
	r.Use(GinResponseBodyLogMiddleware())
	r.Use(Tracer())
	r.Use(Ginzap(alog.Logger, time.RFC3339, true))
	r.Use(RecoveryWithZap(alog.Logger, true))

	r.NoRoute(NoRoute())
	r.NoMethod(NoMethod())

	addHealthEndpoints(r, options)

	return r
}

func addHealthEndpoints(r *gin.Engine, options *Options) {
	r.GET("/livez", options.LivezHandler)
	r.GET("/readyz", options.ReadyzHandler)
	r.GET("/configz", options.ConfigzHandler)
	r.GET("/clearz", clearzHandler)
}

// clear all framework cache
func clearzHandler(c *gin.Context) {
	aregion.ClearCache()
	c.JSON(200, gin.H{"status": 200, "message": "success"})
}
