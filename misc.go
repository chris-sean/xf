package xf

import (
	"encoding/json"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"reflect"
	"strings"
	"unsafe"
)

// MustValidateMap 检查map里的键值类型，是否符合T类型中相同参数名的类型。
// map中缺少的参数，或增加的参数，无法被验证。
func MustValidateMap[T any](m map[string]interface{}) {
	bytes, err := json.Marshal(m)
	if err != nil {
		panic(ErrMarshalJSONError(err))
	}

	var t T
	err = json.Unmarshal(bytes, &t)
	if err != nil {
		panic(ErrUnmarshalJSONError(err))
	}
}

// StringToBytes converts string to byte slice without a memory allocation.
// Export from github.com/gin-gonic/gin.
func StringToBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(
		&struct {
			string
			Cap int
		}{s, len(s)},
	))
}

// BytesToString converts byte slice to string without a memory allocation.
// Export from github.com/gin-gonic/gin.
func BytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// UUID12 returns a length of 12 characters UUID string.
func UUID12() string {
	return uuid.NewString()[24:]
}

// UUID8 returns a length of 8 characters UUID string.
func UUID8() string {
	return uuid.NewString()[:8]
}

// UUID4 returns a length of 4 characters UUID string.
func UUID4() string {
	return uuid.NewString()[:4]
}

var shortUUIDConstraintError = NewErrorType("ShortUUIDConstraintError", 500, "length must be in [1, 32]")

// ShortUUID returns a certain length of UUID string.
// length must between [1, 32].
// There is no '-' in returned uuid.
// Will panic if length is not in the range.
func ShortUUID(length int) string {
	if length > 32 || length <= 0 {
		panic(shortUUIDConstraintError)
	}
	u := uuid.NewString()
	if length <= 8 {
		return u[:length]
	}
	if length <= 12 {
		return u[24 : 24+length]
	}
	u = strings.ReplaceAll(u, "-", "")
	return u[:length]
}

func AutoRecover(ctx *CTX, job func()) {
	defer func() {
		if err := recover(); err != nil {
			var traceID string
			if ctx != nil {
				traceID = ctx.traceID
			}
			Errorf("[%v] %v", traceID, err)
		}
	}()
	job()
}

func AutoRecoverReturns[T any](ctx *CTX, job func() T) T {
	defer func() {
		if err := recover(); err != nil {
			var traceID string
			if ctx != nil {
				traceID = ctx.traceID
			}
			Errorf("[%v] %v", traceID, err)
		}
	}()
	return job()
}

func AutoRecoverAsync(ctx *CTX, job func()) {
	go func() {
		AutoRecover(ctx, job)
	}()
}

func IsValueNil(i interface{}) bool {
	if i == nil {
		return true
	}
	switch reflect.TypeOf(i).Kind() {
	case reflect.Ptr,
		reflect.Interface,
		reflect.Map,
		reflect.Slice,
		reflect.Func,
		reflect.Chan,
		reflect.UnsafePointer:
		return reflect.ValueOf(i).IsNil()
	}
	return false
}

func MapToType(m map[string]interface{}, t interface{}) {
	bytes, err := json.Marshal(m)
	if err != nil {
		panic(ErrMarshalJSONError(err))
	}

	err = json.Unmarshal(bytes, &t)
	if err != nil {
		panic(ErrMarshalJSONError(err))
	}
}

func MapSliceToBSONA(input []map[string]any) bson.A {
	output := make(bson.A, 0, len(input))
	for _, m := range input {
		for k, v := range m {
			a := bson.E{Key: k, Value: v}
			output = append(output, a)
		}
	}
	return output
}
