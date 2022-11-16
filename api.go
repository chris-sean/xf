package xf

import (
	"github.com/gin-gonic/gin"
)

type API[T any] struct {
	Dir        string
	ItemName   string
	ListName   string
	GLASUD     string // 默认开放的接口
	Pagination bool   // 列表接口是否分页

	Auth2JSON map[string]string

	PageGetter  func(h *GinHelper, req *PageMeta) []*T
	ListGetter  func(h *GinHelper, req *PageMeta) []*T
	CountGetter func(h *GinHelper, req *PageMeta) int64
	Adder       func(h *GinHelper, req *T) (id any)
	Saver       func(h *GinHelper, req *T)
	Deleter     func(h *GinHelper, filter map[string]any)
	Setter      func(h *GinHelper, updates map[string]any)
	Getter      func(h *GinHelper, filter map[string]any) *T
}

func (r *API[T]) RegisterAPI(parent *gin.RouterGroup) (group *gin.RouterGroup) {
	group = parent.Group(r.Dir)

	for _, c := range r.GLASUD {
		switch c {
		case 'g':
			group.POST("get", r.get)
		case 'l':
			if r.Pagination {
				group.POST("page", r.getPage)
			} else {
				group.POST("list", r.getList)
			}
			group.POST("count", r.count)
		case 'a':
			group.POST("add", r.add)
		case 's':
			group.POST("save", r.save)
		case 'u':
			group.POST("set", r.set)
		case 'd':
			group.POST("del", r.del)
		}
	}

	return
}

func (r *API[T]) getPage(c *gin.Context) {
	h := NewGinHelper(c)
	req := r.MustGetPageReq(h)

	data := r.PageGetter(h, req)

	resp := &PageResp[T]{
		PageMeta: req,
		Data:     data,
	}

	h.RespondKV200(r.ListName, resp, nil)
}

func (r *API[T]) getList(c *gin.Context) {
	h := NewGinHelper(c)
	req := r.MustGetPageReq(h)

	data := r.ListGetter(h, req)

	h.RespondKV200(r.ListName, data, nil)
}

func (r *API[T]) count(c *gin.Context) {
	h := NewGinHelper(c)
	req := r.MustGetPageReq(h)

	data := r.CountGetter(h, req)

	h.RespondKV200("count", data, nil)
}

func (r *API[T]) add(c *gin.Context) {
	h := NewGinHelper(c)
	var req = new(T)
	r.MustGetObjReq(h, req)

	id := r.Adder(h, req)

	if id == nil {
		h.RespondErrorElse200(nil)
	} else {
		h.RespondKV200(FieldID, id, nil)
	}
}

func (r *API[T]) save(c *gin.Context) {
	h := NewGinHelper(c)
	var req = new(T)
	r.MustGetObjReq(h, req)

	r.Saver(h, req)

	h.RespondErrorElse200(nil)
}

func (r *API[T]) del(c *gin.Context) {
	h := NewGinHelper(c)
	req := r.MustGetJSONReq(h)

	r.Deleter(h, req)

	h.RespondErrorElse200(nil)
}

func (r *API[T]) set(c *gin.Context) {
	h := NewGinHelper(c)
	req := r.MustGetJSONReq(h)

	r.Setter(h, req)

	h.RespondErrorElse200(nil)
}

func (r *API[T]) get(c *gin.Context) {
	h := NewGinHelper(c)
	req := r.MustGetJSONReq(h)

	data := r.Getter(h, req)

	h.RespondKV200(r.ItemName, data, nil)
}

func (r *API[T]) MustGetJSONReq(h *GinHelper) map[string]interface{} {
	m, et := h.UnmarshalJSONToMap()
	if et != nil {
		panic(et)
	}
	OverwrittenByJWT(h.Context, r.Auth2JSON, m)
	return m
}

func (r *API[T]) MustGetPageReq(h *GinHelper) *PageMeta {
	req := &PageMeta{}

	h.MustBind(req)

	if req.Match == nil {
		req.Match = map[string]interface{}{}
	}

	OverwrittenByJWT(h.Context, r.Auth2JSON, req.Match)

	return req
}

func (r *API[T]) MustGetObjReq(h *GinHelper, reqPtr any) {
	if len(r.Auth2JSON) == 0 {
		h.MustBind(reqPtr)
		return
	}

	j := r.MustGetJSONReq(h)

	MapToType(j, reqPtr)
}
