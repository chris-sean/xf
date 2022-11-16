package xf

import (
	"go.mongodb.org/mongo-driver/bson"
	"reflect"
	"strings"
)

type CommonSvc[T any, P CommonModel[T]] struct {
	*CTX
	GenericDAO func(*CTX) DAO[T, P]
	// 查询列表数据，返回的每个对象包含的字段，不填返回全部。值为1表示需要，值为0表示不需要。
	ListFields map[string]any
	// 查询详情数据包含的字段，不填返回全部。值为1表示需要，值为0表示不需要。
	DetailFields map[string]any
	// 查删改记录必填的查询字段，可以有多组，组名自定义，至少有一组中的所有参数必须全部命中。查询参数可以比必填参数多。
	FilterFields map[string][]string
	// 允许外部修改的字段，不在其中的将被剔除。
	//todo 不适用于增加记录
	ModFields map[string]struct{}
	// 通过关键字搜索时，要查哪些字段名。
	KeywordToFields []string
	// 通用代码不会查询这些字段
	HiddenFields []string
	// false，标记删除；true，物理删除；
	HardDeletion bool
	// 是否允许修改或删除多个记录。
	AllowsModMany bool
}

func (r *CommonSvc[T, P]) PrepareGet(filter map[string]any) {
	// 必须包含唯一标识参数
	mustIncludeFields(filter, r.FilterFields)
}

func (r *CommonSvc[T, P]) MustGet(filter map[string]any) P {
	r.PrepareGet(filter)
	return r.GenericDAO(r.CTX).MustGet(filter, r.DetailFields)
}

func (r *CommonSvc[T, P]) fixSearchParameters(page *PageMeta) {
	if len(r.KeywordToFields) == 0 {
		return
	}

	text := strings.TrimSpace(page.SearchText)
	if text == "" {
		return
	}

	fields := make(map[string]any, len(r.KeywordToFields))
	for _, f := range r.KeywordToFields {
		fields[f] = text
	}

	if page.Search == nil {
		page.Search = map[string]interface{}{}
	}

	page.Search["$or"] = fields
}

func (r *CommonSvc[T, P]) PreparePageRequest(page *PageMeta) {
	r.fixSearchParameters(page)
}

func (r *CommonSvc[T, P]) FixPageForResponse(page *PageMeta) {
	page.Match = nil
	page.Search = nil
	page.SearchText = ""
}

func (r *CommonSvc[T, P]) MustGetPage(page *PageMeta) []P {
	r.PreparePageRequest(page)

	defer r.FixPageForResponse(page)

	return r.GenericDAO(r.CTX).MustGetPage(page, r.ListFields)
}

func (r *CommonSvc[T, P]) MustGetList(page *PageMeta) []P {
	r.PreparePageRequest(page)
	return r.GenericDAO(r.CTX).MustGetList(page, r.ListFields)
}

func (r *CommonSvc[T, P]) MustCount(page *PageMeta) int64 {
	r.PreparePageRequest(page)
	return r.GenericDAO(r.CTX).MustCount(page)
}

func (r *CommonSvc[T, P]) PrepareAdd(doc P) {
	// nothing to do yet
}

func (r *CommonSvc[T, P]) MustAdd(doc P) {
	r.PrepareAdd(doc)
	r.GenericDAO(r.CTX).MustAdd(doc)
}

func (r *CommonSvc[T, P]) PrepareSave(doc P) {
	// nothing to do yet
}

func (r *CommonSvc[T, P]) MustSave(doc P) {
	r.PrepareSave(doc)
	r.GenericDAO(r.CTX).MustSave(doc)
}

func (r *CommonSvc[T, P]) PrepareUpdates(updates map[string]any) (filter map[string]any) {
	// 必须包含唯一标识参数
	idFields := mustIncludeFields(updates, r.FilterFields)

	filter = map[string]any{}

	// 将唯一标识参数分离到filter
	for _, field := range idFields {
		extractFilterFromUpdates(filter, updates, field)
	}

	MustFixFilter(filter)

	// 将不允许修改的字段剔除
	LimitModFields(r.ModFields, updates)

	return
}

func (r *CommonSvc[T, P]) MustUpdate(updates map[string]any) int64 {
	filter := r.PrepareUpdates(updates)

	if len(updates) == 0 {
		return 0
	}

	if r.AllowsModMany {
		return r.GenericDAO(r.CTX).MustUpdateMany(filter, updates, nil)
	}
	return r.GenericDAO(r.CTX).MustUpdate(filter, updates, nil)
}

func (r *CommonSvc[T, P]) PrepareDelete(filter map[string]any) {
	// 必须包含唯一标识参数
	mustIncludeFields(filter, r.FilterFields)
	MustFixFilter(filter)
}

func (r *CommonSvc[T, P]) MustDelete(filter map[string]any) {
	r.PrepareDelete(filter)

	if r.HardDeletion {
		if r.AllowsModMany {
			r.GenericDAO(r.CTX).MustHardDeleteMany(filter)
			return
		}
		r.GenericDAO(r.CTX).MustHardDelete(filter)
	} else {
		if r.AllowsModMany {
			r.GenericDAO(r.CTX).MustSoftDeleteMany(filter)
			return
		}
		r.GenericDAO(r.CTX).MustSoftDelete(filter)
	}
}

// 将查询属性和要修改的属性分离
func extractFilterFromUpdates(filter, updates map[string]any, paraName string) {
	v, ok := updates[paraName]
	if !ok {
		panic(ErrInvalidParameters(paraName))
	}
	filter[paraName] = v
	delete(updates, paraName)
}

func MustFixFilter(filter map[string]any) {
	for k, v := range filter {
		if a, ok := v.([]any); ok {
			for _, e := range a {
				switch reflect.TypeOf(e).Kind() {
				case reflect.Struct, reflect.Map, reflect.Slice, reflect.Array:
					// 过滤条件里的数组元素不可以是对象或数组。以防通过操作符查询越权数据。
					panic(ErrInvalidParameters(k))
				}
			}
			filter[k] = bson.M{"$in": v}
		}
	}
}

func mustIncludeFields(para map[string]any, must map[string][]string) []string {
	var missingField string
	var filteredGroup []string
	var mustGroup []string
L:
	for k, group := range must {
		if len(group) == 0 {
			continue
		}

		if k == "must" {
			for _, field := range group {
				if _, ok := para[field]; !ok {
					panic(ErrInvalidParameters(missingField))
				}
			}
			mustGroup = group
			continue
		}

		for _, field := range group {
			if _, ok := para[field]; !ok {
				missingField = field
				continue L
			}
		}
		filteredGroup = group
	}

	if len(mustGroup) > 0 {
		combined := make([]string, 0, len(mustGroup)+len(filteredGroup))
		combined = append(combined, mustGroup...)
		combined = append(combined, filteredGroup...)
		return combined
	}

	if len(filteredGroup) > 0 {
		return filteredGroup
	}

	panic(ErrInvalidParameters(missingField))
}

// LimitModFields 将不允许修改的字段剔除
func LimitModFields(modFields map[string]struct{}, updates map[string]any) {
	for field, _ := range updates {
		if _, ok := modFields[field]; !ok {
			delete(updates, field)
		}
	}
}

func MustGetStringInMap(paraName string, m map[string]any) string {
	s, ok := GetStringInMap(paraName, m)
	if !ok {
		panic(ErrInvalidParameters(paraName))
	}
	return s
}

func GetStringInMap(paraName string, m map[string]any) (string, bool) {
	v, ok := m[paraName]
	if !ok {
		return "", false
	}

	s, ok := v.(string)
	if !ok || s == "" {
		return "", false
	}

	return s, true
}

func MustGetSliceInMap(paraName string, m map[string]any) []any {
	s, ok := GetSliceInMap(paraName, m)
	if !ok {
		str, ok := GetStringInMap(paraName, m)
		if !ok {
			panic(ErrInvalidParameters(paraName))
		}

		return []any{str}
	}
	return s
}

func GetSliceInMap(paraName string, m map[string]any) ([]any, bool) {
	v, ok := m[paraName]
	if !ok {
		return nil, false
	}

	s, ok := v.([]any)
	if !ok || len(s) == 0 {
		return nil, false
	}

	return s, true
}
