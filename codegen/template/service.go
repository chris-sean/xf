package template

const SERVICE_GO = `// #Annotation#
package service

import (
	"#ModuleName#/dao"
	"#ModuleName#/model"
)

type #TypeName# = model.#TypeName#

type #TypeName#Svc struct {
	*xf.CommonSvc[#TypeName#, *#TypeName#]
}

func New#TypeName#Svc(ctx *xf.CTX) *#TypeName#Svc {
	return &#TypeName#Svc{
		CommonSvc: &xf.CommonSvc[#TypeName#, *#TypeName#]{
			CTX:             ctx,
			GenericDAO:      dao.New#TypeName#DAOGeneric,
			ListFields:      model.#TypeName#ListFields(),
			DetailFields:    model.#TypeName#DetailFields(),
			FilterFields:    model.#TypeName#FilterFields(),
			ModFields:       model.#TypeName#ModFields(),
			KeywordToFields: model.#TypeName#SearchableFields(),
			HardDeletion:    false,
			AllowsModMany:   true,
		},
	}
}
`
