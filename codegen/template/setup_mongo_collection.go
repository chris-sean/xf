package template

const SETUP_MONGO_COLLECTION_GO = `package dao

import (
	"#ModuleName#/model"
)

const mongoDBName = "#ModuleName#"

func setupCollections() {
	//DO_NOT_TOUCH_THIS_COMMENT:SETUP_COLLECTION
}

`
