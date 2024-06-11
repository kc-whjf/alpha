package ginwrapper

import (
	"bytes"
	"io/ioutil"
	"net"
	"net/http/httputil"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/kc-whjf/alpha/aerror"
	"github.com/kc-whjf/alpha/alog"
	"github.com/kc-whjf/alpha/autil"
	"github.com/kc-whjf/alpha/autil/ahttp"
	"github.com/kc-whjf/alpha/autil/ahttp/request"
)

const (
	logMaxBodySize = 1024
)

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w bodyLogWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

func GinResponseBodyLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw
		setResponseBody(c, blw.body)

		c.Next()
	}
}

func setResponseBody(c *gin.Context, body *bytes.Buffer) {
	c.Set("__response_body", body)
}

func getResponseBody(c *gin.Context) string {
	body, exists := c.Get("__response_body")
	if !exists || body == nil {
		return ""
	}

	if b, ok := body.(*bytes.Buffer); ok {
		return b.String()
	}

	return ""
}

func Ginzap(logger *zap.Logger, timeFormat string, utc bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		var reqBody string
		if ahttp.IsTextContentType(c.ContentType()) && c.Request.Body != nil {
			bodyBytes, _ := ioutil.ReadAll(c.Request.Body)
			c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

			reqBody = autil.Substr(string(bodyBytes), 0, logMaxBodySize)
		}

		c.Next()

		end := time.Now()
		latency := end.Sub(start)
		if utc {
			end = end.UTC()
		}

		requestId := request.RequestIdFromGin(c)

		if len(c.Errors) > 0 {
			// Append error field if this is an erroneous request.
			for _, e := range c.Errors.Errors() {
				logger.Error(e,
					zap.String(alog.RequestIdKey, requestId))
			}
		}

		req := map[string]interface{}{
			"method":  c.Request.Method,
			"url":     c.Request.URL.String(),
			"headers": c.Request.Header,
			"body":    reqBody,
		}

		var respBody string
		if ahttp.IsTextContentType(ahttp.GetContentType(c.Writer.Header())) {
			respBody = autil.Substr(getResponseBody(c), 0, logMaxBodySize)
		}
		resp := map[string]interface{}{
			"status":  c.Writer.Status(),
			"headers": c.Writer.Header(),
			"body":    respBody,
		}

		logger.Info("access_log",
			zap.Any("request", req),
			zap.Any("response", resp),
			zap.String("ip", c.ClientIP()),
			zap.Duration("latency", latency),
			zap.String(alog.RequestIdKey, requestId),
		)
	}
}

func RecoveryWithZap(logger *zap.Logger, stack bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestId := request.RequestIdFromGin(c)
		defer func() {
			if err := recover(); err != nil {
				// Check for a broken connection, as it is not really a
				// condition that warrants a panic stack trace.
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						if strings.Contains(strings.ToLower(se.Error()), "broken pipe") || strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
							brokenPipe = true
						}
					}
				}

				httpRequest, _ := httputil.DumpRequest(c.Request, false)
				if brokenPipe {
					logger.Error(c.Request.URL.Path,
						zap.Any("error", err),
						zap.String("request", string(httpRequest)),
						zap.String(alog.RequestIdKey, requestId),
					)
					// If the connection is dead, we can't write a status to it.
					c.Error(err.(error)) // nolint: errcheck
					c.JSON(500, aerror.ErrInternalError().WithMessagef("internal error: %v", err))
					c.Abort()
					return
				}

				if stack {
					// logger.Error("[Recovery from panic]",
					logger.Error("[Recovery from panic]"+string(debug.Stack()),
						zap.Time("time", time.Now()),
						zap.Any("error", err),
						zap.String("request", string(httpRequest)),
						zap.String("stack", string(debug.Stack())),
						zap.String(alog.RequestIdKey, requestId),
					)
				} else {
					logger.Error("[Recovery from panic]",
						zap.Time("time", time.Now()),
						zap.Any("error", err),
						zap.String("request", string(httpRequest)),
						zap.String(alog.RequestIdKey, requestId),
					)
				}
				c.JSON(500, aerror.ErrInternalError().WithMessagef("internal error: %v", err))
				c.Abort()
			}
		}()
		c.Next()
	}
}

func Tracer() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestId := c.GetHeader("X-Request-Id")
		if requestId == "" {
			requestId = autil.GenerateUuid()
		}
		request.RequestIdToGin(c, requestId)
		c.Next()
	}
}

func NoRoute() gin.HandlerFunc {
	return func(c *gin.Context) {
		err := aerror.ErrNotFound().WithMessagef("resource not found: %v", c.Request.URL.Path)
		c.JSON(err.HTTPStatusCode, err)
	}
}

func NoMethod() gin.HandlerFunc {
	return func(c *gin.Context) {
		err := aerror.ErrMethodNotAllowed().WithMessagef("method not allowed: %v", c.Request.Method)
		c.JSON(err.HTTPStatusCode, err)
	}
}
