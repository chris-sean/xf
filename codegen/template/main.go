package template

const MAIN_GO = `package main

import (
	"#ModuleName#/api"
	"#ModuleName#/dao"
)

func main() {
	dao.Init()

	api.Init()
}
`
