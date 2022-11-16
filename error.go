package xf

import (
	"fmt"
)

type ErrorType interface {
	error
	ErrorCode() interface{}
	StatusCode() int
	Extra() interface{}
}

type ErrorTypeWriteable interface {
	SetExtra(interface{})
}

type ErrorTypeEntity struct {
	extra       interface{}
	originalErr any
	errStr      string
	errorCode   interface{}
	statusCode  int
}

// ErrorCode change it as you prefer.
func (e ErrorTypeEntity) ErrorCode() interface{} {
	return e.errorCode
}

// StatusCode refers to http response status code.
// Developer may want to set response status code based on error.
// For example, if the error is caused by bad request, then change the return value to 400.
// Ignore this function if no need for your project.
func (e ErrorTypeEntity) StatusCode() int {
	return e.statusCode
}

// Extra returns _extra_ which can be set by user. Usage of _extra_ is determined by user.
func (e ErrorTypeEntity) Extra() interface{} {
	return e.extra
}

// SetExtra sets _extra_ with a value by user. Usage of _extra_ is determined by user.
func (e *ErrorTypeEntity) SetExtra(extra interface{}) {
	e.extra = extra
}

// Error implementation to error interface.
func (e ErrorTypeEntity) Error() string {
	return e.errStr
}

// OriginalError returns original error if there is any.
func (e ErrorTypeEntity) OriginalError() any {
	return e.originalErr
}

func NewErrorType(errCode any, statusCode int, err any, a ...any) ErrorType {
	et := ErrorTypeEntity{
		errorCode:  errCode,
		statusCode: statusCode,
	}

	switch err.(type) {
	case error:
		et.originalErr = err
		et.errStr = fmt.Sprintf("%v", err)
	case string:
		et.errStr = fmt.Sprintf(err.(string), a...)
	default:
		et.errStr = fmt.Sprintf("%v", err)
	}

	return et
}

var notWorthLogging byte
var printErrAsInfo byte

// TryConvertToErrorType returns an ErrorType if err is an ErrorType. returns nil if not.
func TryConvertToErrorType(err interface{}) ErrorType {
	et, ok := err.(ErrorType)
	if ok {
		return et
	}
	return nil
}

func ErrModNoNeedToLog(et ErrorTypeWriteable) {
	et.SetExtra(&notWorthLogging)
}

func ErrModPrintAsInfo(et ErrorTypeWriteable) {
	et.SetExtra(&printErrAsInfo)
}

func ErrAnyError(err any) ErrorType {
	return NewErrorType("AnyError", 500, err)
}

func ErrGeneralError(err any) ErrorType {
	return NewErrorType("GeneralError", 500, err)
}

func ErrServerInternalError(err any) ErrorType {
	return NewErrorType("ServerInternalError", 500, err)
}

func ErrParamBindingError(err any) ErrorType {
	return NewErrorType("ParamBindingError", 400, err)
}

func ErrReadRequestBodyError(err any) ErrorType {
	return NewErrorType("ReadRequestBodyError", 500, err)
}

func ErrUnmarshalJSONError(err any) ErrorType {
	return NewErrorType("UnmarshalJSONError", 400, err)
}

func ErrMarshalJSONError(err any) ErrorType {
	return NewErrorType("MarshalJSONError", 500, err)
}

func ErrInvalidJWTPayload(err any) ErrorType {
	return NewErrorType("InvalidJWTPayload", 400, err)
}

func ErrGRPCDialError(host string, err any) ErrorType {
	return NewErrorType("GRPCDialError", 500, nil, fmt.Sprintf(`Can't dial to grpc server %v. error=%v`, host, err))
}

func ErrMongoQueryError(err any) ErrorType {
	return NewErrorType("MongoQueryError", 500, err)
}

func ErrMongoWriteError(err any) ErrorType {
	return NewErrorType("MongoWriteError", 500, err)
}

func ErrMongoTransactionError(err any) ErrorType {
	return NewErrorType("MongoTransactionError", 500, err)
}

func ErrMongoConnectionError(err any) ErrorType {
	return NewErrorType("MongoConnectionError", 500, err)
}

func ErrDBQueryError(query string, err any) ErrorType {
	return NewErrorType("DBQueryError", 500, nil, "query=%v; err=%v", query, err)
}

func ErrInvalidParameters(para string) ErrorType {
	return NewErrorType("InvalidParameters", 400, nil, `Invalid or missing parameter(s): '%v'`, para)
}

func ErrNotFound(err any) ErrorType {
	if err == nil {
		err = "Not found."
	}
	return NewErrorType("NotFound", 400, err)
}

func ErrForbidden(err any) ErrorType {
	return NewErrorType("Forbidden", 403, err)
}
