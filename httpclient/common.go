package httpclient

import (
	"context"
	"github.com/go-resty/resty/v2"
	"github.com/kc-whjf/alpha/aerror"
	"github.com/pkg/errors"
	"reflect"
)

type HttpParams struct {
	Method  string
	Url     string
	Headers map[string]string
	Queries map[string]string
	Body    interface{}
}

func ExecuteHttp(ctx context.Context, commonResty *resty.Client, p *HttpParams, result interface{}) (err error) {
	if result == nil {
		return errors.New("result 不能为空")
	}

	typ := reflect.TypeOf(result)
	if typ.Kind() != reflect.Ptr {
		return errors.New("result 必须为指针类型")
	}

	wrapper := Wrapper(commonResty.R().
		SetContext(ctx).
		SetHeaders(p.Headers).
		SetQueryParams(p.Queries).
		SetBody(p.Body).
		Execute(p.Method, p.Url))
	if wrapper.FuncError != nil {
		return wrapper.FuncError
	}

	if typ.Elem().Kind() == reflect.String {
		err = wrapper.
			WithError(&aerror.Error{}).
			Parse()
	} else {
		err = wrapper.
			WithResult(result).
			WithError(&aerror.Error{}).
			Parse()
	}

	rawResBody := wrapper.Response.String()
	if err != nil {
		err = errors.Wrapf(err, "响应body:%s", rawResBody)
		return
	}

	if wrapper.Error() != nil {
		err = wrapper.Error().(*aerror.Error)
		return
	}

	if wrapper.Result() != nil {
		result = wrapper.Result()
	} else {
		*(result.(*string)) = rawResBody
	}
	return
}
