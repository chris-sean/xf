package template

const API_GO = `// #Annotation#
package http

import (
	"#ModuleName#/model"
	"#ModuleName#/service"

	"github.com/gin-gonic/gin"
)

type #TypeName# = model.#TypeName#

var #typeName#SvcNew = service.New#TypeName#Svc

type #typeName#API struct {
	xf.API[#TypeName#]
}

var #typeName# = #typeName#API{
	xf.API[#TypeName#]{
        Dir:        "#type-name#",
        ItemName:   "#type_name#",
        ListName:   "#list_name#",
		GLASUD:     "#glasud#",
		Pagination: true,

		Auth2JSON: func() map[string]string {
			return model.#TypeName#Auth2JSONFields()
		},

        PageGetter: func(h *xf.GinHelper, req *xf.PageMeta) []*#TypeName# {
            return #typeName#SvcNew(h.CTX()).MustGetPage(req)
        },

        ListGetter: func(h *xf.GinHelper, req *xf.PageMeta) []*#TypeName# {
            return #typeName#SvcNew(h.CTX()).MustGetList(req)
        },

        Adder: func(h *xf.GinHelper, req *#TypeName#) any {
            #typeName#SvcNew(h.CTX()).MustAdd(req)
            return req.GetID()
        },

        Saver: func(h *xf.GinHelper, req *#TypeName#) {
            #typeName#SvcNew(h.CTX()).MustSave(req)
        },

        Deleter: func(h *xf.GinHelper, filter map[string]any) {
            #typeName#SvcNew(h.CTX()).MustDelete(filter)
        },

        Setter: func(h *xf.GinHelper, updates map[string]any) {
            #typeName#SvcNew(h.CTX()).MustUpdate(updates)
        },

        Getter: func(h *xf.GinHelper, filter map[string]any) *#TypeName# {
            return #typeName#SvcNew(h.CTX()).MustGet(filter)
        },
    },
}

func (r #typeName#API) registerAPI(parent *gin.RouterGroup) (group *gin.RouterGroup) {
    group = r.API.RegisterAPI(parent)
	// group.POST("awesome-api", r.awesome)
	return
}

//func (r #typeName#API) awesome(c *gin.Context) {
//	h := xf.NewGinHelper(c)
//	var req struct {
//		ID   string ` + "`json:\"id\"`" + `
//	}
//
//	if !h.MustBind(&req) {
//		return
//	}
//
//	data := #typeName#SvcNew(h.CTX()).MustBeAwesome(req.ID)
//
//	h.RespondKV200("awesome", data, nil)
//}
`

const API_INIT_GO = `package api

import "#ModuleName#/api/http"

func Init() {
	http.StartHttpServer()
}
`

const API_HTTP_SERVER_GO = `package http

import (
	"fmt"
	"github.com/chris-sean/xf"
	"ts-device-mgmt/config"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap/zapcore"
)

var router *gin.Engine

func StartHttpServer() {
	xf.Infof("API Server is starting on :%v", config.HTTPPort())
	router = gin.New()
	if config.LogLevel() != zapcore.DebugLevel {
		gin.SetMode(gin.ReleaseMode)
	}

	router.Use(gzip.Gzip(gzip.DefaultCompression))
	router.Use(xf.GinLogger())
	//corsConfig := cors.DefaultConfig()
	//corsConfig.AllowOrigins = []string{"*"}
	//router.Use(cors.New(corsConfig))

	// xf.GinMiddleware()必须在gzip后面
	router.Use(xf.GinMiddleware())

	registerAPI()

	xf.Infof("API server is listening at :%v", config.HTTPPort())
	xf.Panic(router.Run(fmt.Sprintf(":%s", config.HTTPPort())))
}
`

const API_HTTP_REGISTER_GO = `package http

import (
	"github.com/gin-contrib/pprof"
)

func registerAPI() {
	pprof.Register(router, "/pprof")
	root := router.Group("api")

	//DO_NOT_TOUCH_THIS_COMMENT:REGISTER_API
}
`
