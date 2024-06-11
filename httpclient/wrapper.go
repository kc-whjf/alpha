package httpclient

import (
	"encoding/json"
	"fmt"
	"github.com/kc-whjf/alpha/aerror"
	"net/http"

	"github.com/go-resty/resty/v2"
	"github.com/kc-whjf/alpha/autil"
)

var (
	errMustWithError = fmt.Errorf("httpclient: must WithError()")
	errRespIsNil     = fmt.Errorf("httpclient: resp is nil")
)

type RespWrapper struct {
	Response  *resty.Response
	FuncError error

	result interface{}
	error  interface{}
}

func Wrapper(r *resty.Response, funcError error) *RespWrapper {
	if funcError != nil {
		logRequest(r, funcError)
	}

	return &RespWrapper{
		Response:  r,
		FuncError: funcError,
	}
}

func (rp *RespWrapper) isSuccess() bool {
	return rp.Response.StatusCode() > 199 && rp.Response.StatusCode() < 300
}

func (rp *RespWrapper) isError() bool {
	return rp.Response.StatusCode() > 399
}

func (rp *RespWrapper) WithResult(result interface{}) *RespWrapper {
	rp.result = autil.GetPointer(result)
	return rp
}

func (rp *RespWrapper) WithError(err interface{}) *RespWrapper {
	rp.error = autil.GetPointer(err)
	return rp
}

func (rp *RespWrapper) Result() interface{} {
	return rp.result
}

func (rp *RespWrapper) Error() interface{} {
	return rp.error
}

func (rp *RespWrapper) Parse() (err error) {
	if rp.Response == nil {
		return errRespIsNil
	}
	if rp.error == nil {
		return errMustWithError
	}

	statusCode := rp.Response.StatusCode()
	// Success
	if statusCode == http.StatusNoContent {
		rp.result = nil
		rp.error = nil
		return
	}

	body := rp.Response.Body()

	// HTTP status code > 199 and < 300, considered as result
	if rp.isSuccess() {
		rp.error = nil
		if len(body) == 0 {
			rp.result = nil
			return
		}
		if rp.result == nil {
			return
		}
		err = json.Unmarshal(body, rp.result)
		if err != nil {
			err = fmt.Errorf("htttpclient: upstream[%d]: unmarshal body failed: %v", statusCode, err)
		}
		return
	}

	// HTTP status code > 399, considered as error
	if rp.isError() {
		rp.result = nil
		if len(body) == 0 {
			rp.error = nil
			err = fmt.Errorf("htttpclient: upstream[%d]: unexpected empty response", statusCode)
			return
		}
		err = json.Unmarshal(body, rp.error)
		if err != nil {
			err = fmt.Errorf("htttpclient: upstream[%d]: unmarshal body failed: %v", statusCode, err)
			return
		}
		err = rp.checkIfValidAError(rp.error)
		return
	}

	// Other status
	rp.result = nil
	rp.error = nil
	err = fmt.Errorf("htttpclient: upstream[%d]: unexpected status: %d", statusCode, statusCode)

	return
}

func (rp *RespWrapper) checkIfValidAError(err interface{}) error {
	if e, ok := err.(*aerror.Error); ok {
		if e.Err.Code == "" {
			return fmt.Errorf("invalid aerror response")
		}
	}

	return nil
}
