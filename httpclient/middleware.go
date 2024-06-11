package httpclient

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime/debug"

	"github.com/go-resty/resty/v2"
	"github.com/kc-whjf/alpha/alog"
	"github.com/kc-whjf/alpha/autil"
	"github.com/kc-whjf/alpha/autil/ahttp"
	"go.uber.org/zap"
)

const (
	logMaxBodySize = 1024
)

func formatRequestBodyString(req *http.Request, maxBodySize int) (string, error) {
	var err error
	bodyReader, err := req.GetBody()
	if err != nil || bodyReader == nil {
		return "", err
	}
	body, err := ioutil.ReadAll(bodyReader)
	if err != nil {
		return "", err
	}

	bodyString := string(body)
	s := autil.Strlen(bodyString)
	if s > maxBodySize {
		return autil.Substr(string(body), 0, maxBodySize) + fmt.Sprintf("...[truncated:%d/%d]", maxBodySize, s), nil
	}

	return bodyString, nil
}

func formatResponseBodyString(body []byte, maxBodySize int) string {
	bodyString := string(body)
	s := autil.Strlen(bodyString)
	if s > maxBodySize {
		return autil.Substr(string(body), 0, maxBodySize) + fmt.Sprintf("...[truncated:%d/%d]", maxBodySize, s)
	}

	return bodyString
}

func logRequestMiddleware() func(c *resty.Client, r *resty.Response) error {
	return func(c *resty.Client, r *resty.Response) error {
		logRequest(r, nil)
		return nil
	}
}

func logRequest(r *resty.Response, funcError error) {
	logger := alog.Logger
	if r == nil {
		logger.Warn("httpclient: Response is nil",
			zap.Any("error", funcError),
		)
		return
	}
	if r.Request != nil {
		logger = alog.Ctx(r.Request.Context())
	}

	defer func() {
		if err := recover(); err != nil {
			logger.Warn("httpclient: logRequestMiddleware() recovery from panic",
				zap.Any("error", err),
				zap.Any("stack", string(debug.Stack())),
			)
		}
	}()

	statusCode := r.StatusCode()
	rawReq := r.Request.RawRequest

	var (
		reqBody  string
		respBody string
		err      error
	)
	if ahttp.IsTextContentType(ahttp.GetContentType(r.Request.Header)) {
		reqBody, err = formatRequestBodyString(rawReq, logMaxBodySize)
		if err != nil {
			logger.Sugar().Warnf("httpclient: formatting request body failed: %v", err)
		}
	}

	if ahttp.IsTextContentType(ahttp.GetContentType(r.Header())) {
		respBody = formatResponseBodyString(r.Body(), logMaxBodySize)
	}

	req := map[string]interface{}{
		"method":  r.Request.Method,
		"url":     r.Request.URL,
		"headers": r.Request.Header,
		"body":    reqBody,
	}
	resp := map[string]interface{}{
		"status":  r.StatusCode(),
		"headers": r.Header(),
		"body":    respBody,
	}
	if funcError != nil {
		resp["func_error"] = funcError.Error()
	}

	// Success
	if statusCode >= 200 && statusCode <= 299 {
		logger.Info("httpclient rpc",
			zap.Any("request", req),
			zap.Any("response", resp),
		)
	} else if statusCode >= 500 || statusCode == 0 {
		logger.Error("httpclient rpc",
			zap.Any("request", req),
			zap.Any("response", resp),
		)
	} else {
		logger.Warn("httpclient rpc",
			zap.Any("request", req),
			zap.Any("response", resp),
		)
	}
}
