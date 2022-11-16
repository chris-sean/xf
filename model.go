package xf

import (
	"reflect"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const FieldID = "id"
const FieldIsDeleted = "is_deleted"
const FieldCreatedAt = "created_at"
const FieldUpdatedAt = "updated_at"

var notDeleted int8 = 0
var deleted *int8 = nil

func NotDeleted() int8 {
	return notDeleted
}

func Deleted() *int8 {
	return deleted
}

type CommonFields struct {
	ID        any        `json:"id,omitempty" form:"id" bson:"id" gorm:"primaryKey" xf:"filter,omit:mod"`
	CreatedAt *time.Time `json:"created_at,omitempty" bson:"created_at,omitempty" xf:"omit:mod" gorm:"type:datetime(3);index;default:CURRENT_TIMESTAMP(3);comment:Time of latest update."`
	UpdatedAt *time.Time `json:"updated_at,omitempty" bson:"updated_at" xf:"omit:mod" gorm:"type:datetime(3);index;default:CURRENT_TIMESTAMP(3);comment:Time of creation."`
	IsDeleted *int8      `json:"is_deleted,omitempty" bson:"is_deleted" xf:"omit:all" gorm:"type:tinyint(1);index;default:0;comment:0 means not deleted. NULL means deleted. Meaning of other value is undefined."`
}

func (r *CommonFields) GetCreatedAt() *time.Time {
	return r.CreatedAt
}

func (r *CommonFields) SetCreatedAt(t *time.Time) {
	r.CreatedAt = t
}

func (r *CommonFields) GetUpdatedAt() *time.Time {
	return r.UpdatedAt
}

func (r *CommonFields) SetUpdatedAt(t *time.Time) {
	r.UpdatedAt = t
}

func (r *CommonFields) GetIsDeleted() bool {
	return r.IsDeleted != deleted
}

func (r *CommonFields) SetIsDeleted(d bool) {
	if d {
		r.IsDeleted = deleted
	} else {
		r.IsDeleted = &notDeleted
	}
}

func (r *CommonFields) GetID() any {
	return r.ID
}

func (r *CommonFields) SetID(id any) {
	r.ID = id
}

var uniqueIndexOption = options.Index().SetUnique(true).SetPartialFilterExpression(bson.M{FieldIsDeleted: 0})

func UniqueIndexOption() *options.IndexOptions {
	return uniqueIndexOption
}

var indexID = mongo.IndexModel{
	Keys:    bson.M{FieldID: -1},
	Options: options.Index().SetUnique(true),
}

func IndexID() mongo.IndexModel {
	return indexID
}

var indexIsDeleted = mongo.IndexModel{
	Keys: bson.M{FieldIsDeleted: -1},
}

func IndexIsDeleted() mongo.IndexModel {
	return indexIsDeleted
}

var indexCreatedAt = mongo.IndexModel{
	Keys: bson.M{FieldCreatedAt: -1},
}

func IndexCreatedAt() mongo.IndexModel {
	return indexCreatedAt
}

var indexUpdatedAt = mongo.IndexModel{
	Keys: bson.M{FieldUpdatedAt: -1},
}

func IndexUpdatedAt() mongo.IndexModel {
	return indexUpdatedAt
}

type CommonModel[T any] interface {
	ID
	GetCreatedAt() *time.Time
	SetCreatedAt(*time.Time)
	GetUpdatedAt() *time.Time
	SetUpdatedAt(*time.Time)
	GetIsDeleted() bool
	SetIsDeleted(bool)
}

type ID interface {
	GetID() any
	SetID(any)
}

type PageResp[T any] struct {
	*PageMeta
	Data []*T `json:"data"`
}

// todo be able to config
var collation = options.Collation{Locale: "zh"}

func DefaultCollation() *options.Collation {
	return &collation
}

// MapFieldsMethod for caching purpose
type MapFieldsMethod string

const (
	MapFieldsMethodJsonToBson MapFieldsMethod = "JsonToBson"
	MapFieldsMethodJsonToGorm MapFieldsMethod = "JsonToGorm"
)

// MapFields 把json字段名改成bson或gorm字段名。
func MapFields[T any](m map[string]any, method MapFieldsMethod) {
	if len(m) == 0 {
		return
	}

	var fromTagName string
	var toTagName string

	switch method {
	case MapFieldsMethodJsonToBson:
		fromTagName = "json"
		toTagName = "bson"
	case MapFieldsMethodJsonToGorm:
		fromTagName = "json"
		toTagName = "gorm"
	default:
		return
	}

	var v T
	t := reflect.TypeOf(v)
	name := t.Name()

	var rule map[string]string

	structFieldMapsLock.Lock()
	defer structFieldMapsLock.Unlock()

	methodMap, ok := structFieldMaps[method]

	if ok {
		rule, ok = methodMap[name]
	} else {
		methodMap = map[string]map[string]string{}
		structFieldMaps[method] = methodMap
	}

	if !ok {
		rule = map[string]string{}

		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)

			fromTag := field.Tag.Get(fromTagName)
			from := fieldNameFromTag(fromTagName, fromTag)

			if from == "" || from == "-" {
				continue
			}

			toTag := field.Tag.Get(toTagName)
			to := fieldNameFromTag(toTagName, toTag)

			if to == "" || to == "-" {
				continue
			}

			if from == to {
				continue
			}

			rule[from] = to
		}

		methodMap[name] = rule
	}

	for k, v := range m {
		if newK, ok := rule[k]; ok {
			delete(m, k)
			m[newK] = v
		}
	}
}

func fieldNameFromTag(typ, value string) string {
	switch typ {
	case "json":
		s := strings.Split(value, ",")
		if len(s) > 0 {
			return s[0]
		}
		return value
	case "bson":
		s := strings.Split(value, ",")
		if len(s) > 0 {
			return s[0]
		}
		return value
	case "gorm":
		s := strings.Split(value, ";")
		if len(s) == 0 {
			s = []string{value}
		}
		for _, kv := range s {
			if strings.Index(kv, "column:") == 0 {
				return kv[7:]
			}
		}
	}
	return ""
}

var structFieldMaps = map[MapFieldsMethod]map[string]map[string]string{}
var structFieldMapsLock sync.Mutex

func parseXFTag2(t reflect.Type, dbTag string,
	filterFields map[string][]string,
	searchableFields []string,
	listFields, detailFields map[string]any,
	modFields map[string]struct{},
) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// 只处理公开参数
		if !field.IsExported() {
			continue
		}

		if field.Anonymous {
			tt := field.Type
			for tt.Kind() == reflect.Ptr {
				tt = tt.Elem()
			}
			switch tt.Kind() {
			case reflect.Struct:
				parseXFTag2(tt, dbTag, filterFields, searchableFields, listFields, detailFields, modFields)
			}
			continue
		}

		fn := dbNameOfField(field, dbTag)
		if fn == "" {
			continue
		}

		listFields[fn] = 1
		detailFields[fn] = 1
		modFields[fn] = struct{}{}

		xf := field.Tag.Get("xf")
		if xf == "" {
			continue
		}

		opts := strings.Split(xf, ",")

		for _, opt := range opts {
			kv := strings.Split(opt, ":")
			switch kv[0] {
			case "filter":
				writeFilterFieldsMap(kv, fn, filterFields)
			case "search":
				searchableFields = append(searchableFields, fn)
			case "omit":
				writeOmitFieldsMap(kv, fn, listFields, detailFields, modFields)
			}
		}
	}
}

func ParseXFTag(obj any, dbTag string) (
	filterFields map[string][]string,
	searchableFields []string,
	listFields, detailFields map[string]any,
	modFields map[string]struct{},
) {
	t := reflect.TypeOf(obj)
	filterFields = map[string][]string{}
	listFields = map[string]any{}
	detailFields = map[string]any{}
	modFields = map[string]struct{}{}

	parseXFTag2(t, dbTag, filterFields, searchableFields, listFields, detailFields, modFields)
	return
}

func dbNameOfField(field reflect.StructField, preferredTag string) string {
	switch preferredTag {
	case "bson":
		bsonName := strings.Split(field.Tag.Get("bson"), ",")
		if len(bsonName) > 0 {
			return bsonName[0]
		}
	case "gorm":
		//todo tbd
	}
	return field.Name
}

func writeFilterFieldsMap(kv []string, fieldName string, filterFieldsMap map[string][]string) {
	if len(kv) == 0 {
		return
	}

	filterGroup := ""
	if len(kv) > 1 {
		filterGroup = kv[1]
	}

	fields, ok := filterFieldsMap[filterGroup]
	if !ok {
		fields = make([]string, 0, 1)
	}
	fields = append(fields, fieldName)
	filterFieldsMap[filterGroup] = fields
}

// 默认实现的接口需要隐藏的输入输出参数。自定义接口实现不会生效，有需要可以参考相应的参数。
func writeOmitFieldsMap(
	kv []string,
	fieldName string,
	listFields, detailFields map[string]any,
	modFields map[string]struct{},
) {
	if len(kv) < 2 {
		return
	}

	opts := strings.Split(kv[1], "&")

	for _, opt := range opts {
		switch opt {
		case "list": // 列表的输出参数不显示
			delete(listFields, fieldName)
		case "detail": // 详情的输出参数不显示
			delete(detailFields, fieldName)
		case "output": // 列表和详情的输出参数均不显示
			delete(listFields, fieldName)
			delete(detailFields, fieldName)
		case "mod": // 不允许外部直接修改的字段
			delete(modFields, fieldName)
		case "all": // 应用以上全部
			delete(listFields, fieldName)
			delete(detailFields, fieldName)
			delete(modFields, fieldName)
		}
	}
}

func Auth2JSONMap(obj any) map[string]string {
	m := map[string]string{}
	t := reflect.TypeOf(obj)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		authName := field.Tag.Get("auth")
		if authName == "" {
			continue
		}

		jsonName := strings.Split(field.Tag.Get("json"), ",")
		if len(jsonName) == 0 {
			continue
		}

		m[jsonName[0]] = authName
	}
	return m
}
