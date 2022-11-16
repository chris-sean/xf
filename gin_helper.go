package xf

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"unsafe"

	"github.com/gin-gonic/gin"
)

//const TraceIDKey = "TID"

func traceIDForGinCreateIfNil(c *gin.Context) (traceID string) {
	if c == nil {
		return "gin_Context_is_nil!"
	}
	traceID = c.GetHeader("X-Request-ID")
	if traceID == "" {
		traceID = UUID12()
		c.Header("X-Request-ID", traceID)
	}
	return traceID
}

func traceIDFromGin(c *gin.Context) (traceID string) {
	return c.GetHeader("X-Request-ID")
}

// GinHelper provides some helper functions. Respond JSON only.
type GinHelper struct {
	*gin.Context
}

//var g = GinHelper{}
//
//func G() GinHelper {
//	return g
//}

func GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		injectCTX(c)
		defer handlePanic(c)
		c.Next()
	}
}

func injectCTX(c *gin.Context) *CTX {
	if _, ok := c.Get("ctx"); ok {
		return nil
	}

	ctx := &CTX{
		traceID: UUID12(),
	}
	c.Set("ctx", ctx)

	return ctx
}

func getCTX(c *gin.Context) *CTX {
	if v, ok := c.Get("ctx"); ok {
		if ctx, ok := v.(*CTX); ok {
			return ctx
		}
	}

	return injectCTX(c)
}

func NewGinHelper(c *gin.Context) *GinHelper {
	return &GinHelper{
		Context: c,
	}
}

func (r *GinHelper) CTX() *CTX {
	return getCTX(r.Context)
}

type ErrorPayload struct {
	Code interface{} `json:"code,omitempty"`
	Desc string      `json:"desc"`
	TID  string      `json:"tid,omitempty"`
}

type KV = map[string]interface{}

const errorKey = "error"

func commonResponseBody() map[string]interface{} {
	return map[string]interface{}{
		errorKey: nil,
	}
}

// MustBind binds parameters to obj which must be a pointer. If any error occurred, respond 400.
func (r *GinHelper) MustBind(obj interface{}) {
	if err := r.BindUri(obj); err != nil {
		panic(ErrParamBindingError(err))
	}
	if err := r.ShouldBind(obj); err != nil {
		panic(ErrParamBindingError(err))
	}
}

// Bind Deprecated. binds parameters to obj which must be a pointer. If any error occurred, respond 400.
// return true if binding succeed, vice versa.
func (r *GinHelper) Bind(obj interface{}) bool {
	_ = r.ShouldBindUri(obj)
	if err := r.ShouldBind(obj); err != nil {
		r.RespondError(ErrParamBindingError(err))
		return false
	}
	return true
}

func respondJSON(c *gin.Context, status int, body interface{}) {
	if c == nil {
		Errorf("calling respondJSON(*gin.Context, status, body) with nil context")
		return
	}
	c.JSON(status, body)
}

// Respond Example: payload 1 is {k: "msg" v: "ok"}; payload 2 is {k: "data" v:{id: 1}}.
// Response JSON will be
//
//	{
//		"error": null,
//		"msg": "ok",
//		"data": {
//			"id": 1
//		}
//	}
func (r *GinHelper) Respond(status int, payload KV) {
	body := commonResponseBody()
	for k, v := range payload {
		body[k] = v
	}
	respondJSON(r.Context, status, body)
}

// RespondError responds error in a response in JSON format.
func (r *GinHelper) RespondError(et ErrorType) {
	respondError(r.Context, et)
}

// respondError responds error in a response in JSON format.
func respondError(gc *gin.Context, et ErrorType) {
	if gc == nil {
		Errorf("calling respondError(*gin.Context, et) with nil context")
		return
	}
	if et == nil {
		Errorf("calling respondError(*gin.Context, et) with nil error")
		gc.Abort()
	}

	// Discussion
	// Code in this function may panic.
	// For example, if et is (*ErrorType, nil), call et's function can either result success or failure, depends on each function's implementation.
	// If panicked, respondError will recover once, create an internalError object and call respondError with the new et object.
	// If panicked again, respondError will give up. let http.serve() recover.
	defer func() {
		_, ok := gc.Get("respondErrorFailure")
		if ok {
			// We are here only because recovery below panicked.
			// If code panicked again. Process won't crash because http.serve() will recover.
			Errorf("[%s] respondError panicked twice. et={type=%T; value=%v}", traceIDFromGin(gc), et, et)
			gc.Abort()
			return
		}

		if err := recover(); err != nil {
			// err could be (*ErrorType, nil)
			// So create a solid ErrorType object
			gc.Set("respondErrorFailure", true)
			e := ErrServerInternalError(fmt.Errorf("[%s] unparseable error. type=%T; value=%v", traceIDFromGin(gc), et, et))
			respondError(gc, e)
		}
	}()

	body := commonResponseBody()
	payload := ErrorPayload{
		Code: et.ErrorCode(),
		Desc: et.Error(),
	}

	payload.TID = traceIDForGinCreateIfNil(gc)
	body[errorKey] = payload

	if gin.IsDebugging() || (et.Extra() != &notWorthLogging && et.StatusCode() >= 500) {
		// get raw string of http request using reflect.
		requestLog := requestAsText(gc.Request)

		log := fmt.Sprintf("tid=%v; %scode=%v; error=%v; status=%v", payload.TID, requestLog, et.ErrorCode(), et.Error(), et.StatusCode())

		if et.Extra() == &printErrAsInfo {
			Info(log)
		} else {
			Error(log)
		}
	}

	respondJSON(gc, et.StatusCode(), body)
}

var MaxLengthOfRequestDump = 4 * 1024

func requestAsText(request *http.Request) (requestLog string) {
	if request.ContentLength > 0 && request.Body != nil {
		// Possible to fail with future go version.
		v := reflect.ValueOf(request.Body).Elem()
		v = v.FieldByName("src").Elem().Elem()
		v = v.FieldByName("R").Elem().Elem()
		lv := v.FieldByName("w") // get length of package
		lv = reflect.NewAt(lv.Type(), unsafe.Pointer(lv.UnsafeAddr())).Elem()
		length := lv.Interface().(int)
		v = v.FieldByName("buf")
		v = reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem()
		buf, ok := v.Interface().([]byte)

		if length > MaxLengthOfRequestDump {
			length = MaxLengthOfRequestDump
		}

		if ok {
			requestLog = fmt.Sprintf("request:\n%s\n---EOR---\n", string(buf[:length]))
		}
	} else {
		// assemble headers
		headers := make([]string, 0, len(request.Header))
		for k, vs := range request.Header {
			for _, v := range vs {
				headers = append(headers, k+": "+v)
			}
		}
		requestLog = fmt.Sprintf(`request:
%s %s
%v
---EOR---
`, request.Method, request.RequestURI, strings.Join(headers, "\n"))
	}

	return
}

// RespondKV caller provides one object/array/slice and error, nil if no error.
// Function will make response properly.
func (r *GinHelper) RespondKV(successStatusCode int, key string, value interface{}, et ErrorType) {
	if et != nil {
		r.RespondError(et)
		return
	}
	r.Respond(successStatusCode, KV{key: value})
}

func (r *GinHelper) RespondKV200(key string, value interface{}, et ErrorType) {
	r.RespondKV(200, key, value, et)
}

// RespondKVs caller provides multiple KV objects and error, nil if no error.
// Function will make response properly.
func (r *GinHelper) RespondKVs(successStatusCode int, et ErrorType, payload KV) {
	if et != nil {
		r.RespondError(et)
		return
	}
	r.Respond(successStatusCode, payload)
}

func (r *GinHelper) RespondKVs200(et ErrorType, payload KV) {
	r.RespondKVs(200, et, payload)
}

// RespondFirst caller provide an slice/array. Only the first element if exists will be in the response JSON.
// todo generic
func (r *GinHelper) RespondFirst(successStatusCode int, key string, values interface{}, et ErrorType) {
	if et != nil {
		r.RespondError(et)
		return
	}
	//todo 等范型推出后，删除反射代码。
	if reflect.TypeOf(values).Kind() != reflect.Slice {
		panic(fmt.Sprintf("GeneralRespondFirst values %v", values))
	}
	s := reflect.ValueOf(values)
	if s.Len() > 0 {
		r.Respond(successStatusCode, KV{key: s.Index(0).Interface()})
	} else {
		r.Respond(successStatusCode, KV{key: nil})
	}
}

func (r *GinHelper) RespondFirst200(key string, values interface{}, et ErrorType) {
	r.RespondFirst(200, key, values, et)
}

// RespondErrorElse if error is not nil, respond error.StatusCode() and error in the response JSON;
// Otherwise, respond successStatusCode and error: null in the response JSON
func (r *GinHelper) RespondErrorElse(successStatusCode int, et ErrorType) {
	if et != nil {
		r.RespondError(et)
		return
	}
	r.Respond(successStatusCode, nil)
}

func (r *GinHelper) RespondErrorElse200(et ErrorType) {
	r.RespondErrorElse(200, et)
}

func (r *GinHelper) UnmarshalJSONToMap() (m map[string]interface{}, et ErrorType) {
	bytes, err := io.ReadAll(r.Request.Body)
	if err != nil {
		return nil, ErrReadRequestBodyError(err)
	}

	if len(bytes) == 0 {
		m = map[string]interface{}{}
		return
	}

	err = json.Unmarshal(bytes, &m)
	if err != nil {
		return nil, ErrUnmarshalJSONError(err)
	}

	return
}

func (r *GinHelper) BodyAsJSONSlice() (s []map[string]interface{}, et ErrorType) {
	bytes, err := io.ReadAll(r.Request.Body)
	if err != nil {
		return nil, ErrReadRequestBodyError(err)
	}
	err = json.Unmarshal(bytes, &s)
	if err != nil {
		return nil, ErrUnmarshalJSONError(err)
	}
	return s, nil
}

func handlePanic(c *gin.Context) {
	if err := recover(); err != nil {
		et, ok := err.(ErrorType)
		if ok {
			respondError(c, et)
			c.Abort()
		} else {
			respondError(c, ErrAnyError(et))
			c.Abort()
			//Errorf("Gin has caught a panic. traceID=%s; error=%v", traceIDFromGin(c), err)
			//c.AbortWithStatus(http.StatusInternalServerError)
		}
	}
}

// CreateGRPCContext create a context.Context with header "tid".
func (r *GinHelper) CreateGRPCContext() context.Context {
	context := context.Background()
	return getCTX(r.Context).FillGRPCContext(context)
}

var invalidJWTErr = errors.New("Invalid JWT.")

func GetJWTClaims(c *gin.Context, claimsPointer any) {
	a := c.GetHeader("Authorization")
	//if !strings.HasPrefix(strings.ToLower(a), "Bearer ") {
	if !strings.HasPrefix(a, "Bearer ") {
		panic(ErrInvalidJWTPayload(invalidJWTErr))
	}

	tokenString := a[7:]
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		panic(ErrInvalidJWTPayload(invalidJWTErr))
	}

	claimBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		panic(ErrInvalidJWTPayload(err))
	}

	err = json.Unmarshal(claimBytes, claimsPointer)
	if err != nil {
		panic(ErrInvalidJWTPayload(err))
	}
}

func GetJWTMapClaims(c *gin.Context) map[string]any {
	claims := map[string]any{}
	GetJWTClaims(c, &claims)
	return claims
}

func OverwrittenByJWT(c *gin.Context, mapping map[string]string, json map[string]any) {
	if len(mapping) == 0 || json == nil {
		return
	}
	claims := GetJWTMapClaims(c)

	if len(claims) == 0 {
		return
	}

	for js, jw := range mapping {
		if w, ok := claims[jw]; ok {
			json[js] = w
		}
	}
}

type HandlerFunc func(helper *GinHelper)

func HandlePost(group *gin.RouterGroup, relativePath string, handlers ...HandlerFunc) {
	for _, handler := range handlers {
		group.POST(relativePath, func(c *gin.Context) {
			h := NewGinHelper(c)
			handler(h)
		})
	}
}
