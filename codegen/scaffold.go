package codegen

import (
	"github.com/chris-sean/xf/codegen/template"
	"os"
	"strings"
)

var ModuleName string
var MODULE_NAME string

func init() {
	ModuleName = ReadModuleName()
	MODULE_NAME = strings.ToUpper(ModuleName)
	MODULE_NAME = strings.ReplaceAll(MODULE_NAME, "-", "_")
}

var files = map[string][]file{
	"api": {
		{
			fileName: "init.go",
			content:  template.API_INIT_GO,
		},
	},
	"api/http": {
		{
			fileName: "server.go",
			content:  template.API_HTTP_SERVER_GO,
		},
		{
			fileName: "api_register.go",
			content:  template.API_HTTP_REGISTER_GO,
		},
	},
	"client": {},
	"cmd": {
		{
			fileName: "main.go",
			content:  template.MAIN_GO,
		},
	},
	"config": {
		{
			fileName: "config.go",
			content:  template.CONFIG_GO,
		},
		{
			fileName: "dev.env",
			content:  template.DEV_ENV,
		},
	},
	"dao": {
		{
			fileName: "init.go",
			content:  template.DAO_INIT_GO,
		},
		{
			fileName: "mongo.go",
			content:  template.DAO_MONGO_GO,
		},
		{
			fileName: "mongo_register.go",
			content:  template.DAO_MONGO_REGISTER_GO,
		},
	},
	"deploy": {},
	"errs": {
		{
			fileName: "error.go",
			content:  template.ERROR_GO,
		},
	},
	"model":   {},
	"service": {},
	"test":    {},
}

type file struct {
	fileName string
	content  string
}

func MustGenerateScaffold() {
	for dir, fs := range files {
		os.MkdirAll(dir, 0666)

		for _, f := range fs {
			genScaffoldFile(f.content, dir+"/"+f.fileName)
		}
	}

}

func genScaffoldFile(template, targetFile string) {
	if fileExists(targetFile) {
		return
	}

	template = strings.ReplaceAll(template, "#ModuleName#", ModuleName)
	template = strings.ReplaceAll(template, "#MODULE_NAME#", MODULE_NAME)

	os.WriteFile(targetFile, []byte(template), 0666)
}

func ReadModuleName() string {
	b, err := os.ReadFile("go.mod")
	if err != nil {
		panic(err)
	}

	content := strings.Split(string(b), "\n")

	for _, line := range content {
		if strings.HasPrefix(line, "module ") {
			words := strings.Split(line, " ")
			for _, word := range words {
				w := strings.TrimSpace(word)
				if w == "module" || len(w) == 0 {
					continue
				}
				return w
			}
			break
		}
	}

	return ""
}
