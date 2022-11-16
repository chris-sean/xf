package template

const MODEL_GO = `// #Annotation#
package model

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type #TypeName# struct {
    xf.CommonFields ` + "`bson:\",inline\"`" + `
	//Name      string ` + "`json:\"name,omitempty\" bson:\"name\" xf:\"search\" gorm:\"column:name;type:varchar(50);default:NULL;index;comment:名称\"`" + `
}
#ModelComplement#
// ----GENERATED HELPER CODE BEGIN----

func init() {
    obj := #TypeName#{}
    #typeName#FilterFields, #typeName#SearchableFields, #typeName#ListFields, #typeName#DetailFields, #typeName#ModFields = xf.ParseXFTag(obj, "#tag#")
    #typeName#Auth2JSONFields = xf.Auth2JSONMap(obj)
}

var #typeName#SearchableFields []string

func #TypeName#SearchableFields() []string {
	return #typeName#SearchableFields
}

var #typeName#ListFields map[string]any

func #TypeName#ListFields() map[string]any {
	return #typeName#ListFields
}

var #typeName#DetailFields map[string]any

func #TypeName#DetailFields() map[string]any {
	return #typeName#DetailFields
}

var #typeName#ModFields map[string]struct{}

func #TypeName#ModFields() map[string]struct{} {
	return #typeName#ModFields
}

var #typeName#Auth2JSONFields map[string]string

func #TypeName#Auth2JSONFields() map[string]string {
	return #typeName#Auth2JSONFields
}

// ----GENERATED HELPER CODE END----
`
