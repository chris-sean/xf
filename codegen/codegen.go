package codegen

import (
	"fmt"
	"github.com/chris-sean/xf/codegen/template"
	"os"
	"os/exec"
	"strings"
)

type Vars struct {
	TypeName              string
	typeName              string
	type_name             string
	type_dash_name        string
	list_name             string
	Annotation            string
	ids                   []string
	idsQuote              string // get/update/delete操作时，过滤条件json参数名
	glasud                string // 默认开放的接口
	tableName             string // 指定表名
	ModelCommonType       string
	DAOType               string
	NewDAOTypeFunParaVars string
	mongoIndexCode        string
	tag                   string

	_dbType string
}

func newVars(dbType, TypeName, typeName, type_name, glasud, tableName, Annotation string, additionalIDs []string) Vars {
	if glasud == "" {
		glasud = "glaud"
	}

	var idsquote []string
	for _, id := range additionalIDs {
		idsquote = append(idsquote, "\""+id+"\"")
	}

	var idsq string
	if len(idsquote) > 0 {
		idsq = ", {" + strings.Join(idsquote, ", ") + "}"
	}

	v := Vars{
		TypeName:       TypeName,
		typeName:       typeName,
		type_name:      type_name,
		type_dash_name: strings.ReplaceAll(type_name, "_", "-"),
		//item_name:      item_name,
		list_name:             type_name + "_list",
		Annotation:            Annotation,
		glasud:                glasud,
		ids:                   additionalIDs,
		idsQuote:              idsq,
		_dbType:               dbType,
		tableName:             tableName,
		ModelCommonType:       "CommonMongo",
		DAOType:               "MongoDAO",
		NewDAOTypeFunParaVars: "ctx, mongoClient, mongoDB.Collection(Collection" + TypeName + ")",
		tag:                   "bson",
	}

	if dbType == "sql" {
		v.ModelCommonType = "CommonSQL"
		v.DAOType = "MySQLDAO"
		v.NewDAOTypeFunParaVars = "ctx"
		v.tag = "gorm"

		if tableName == "" {
			v.tableName = type_name
		}

		//v.constructModelDBFuncs()
	}

	return v
}

func MustGenerateAndExitMongoRule(genTest bool, TypeName, typeName, type_name, glasud, tableName, Annotation string, additionalIDs []string) {
	MustGenerateAndExit(genTest, "mongo", TypeName, typeName, type_name, glasud, "", Annotation, additionalIDs)
}

func MustGenerateAndExitSQLRule(genTest bool, TypeName, typeName, type_name, glasud, tableName, Annotation string, additionalIDs []string) {
	MustGenerateAndExit(genTest, "sql", TypeName, typeName, type_name, glasud, tableName, Annotation, additionalIDs)
}

func MustGenerateAndExit(genTest bool, dbType, TypeName, typeName, type_name, glasud, tableName, Annotation string, additionalIDs []string) {
	fmt.Println("**GENERATING CODE**")

	v := newVars(dbType, TypeName, typeName, type_name, glasud, tableName, Annotation, additionalIDs)

	if genTest {
		v.gen(template.TEST_GO, "./api/http/"+type_name+"_test.go", v.replace, false)
		fmt.Println("**DONE GENERATING**")
		os.Exit(0)
	}

	v.fixModelFile()

	//v.gen("./xf/codegen/model.go.tmpl", "./model/"+type_name+".go", v.replace, false)
	v.gen(template.DAO_GO, "./dao/"+type_name+".go", v.replace, false)
	v.gen(template.SERVICE_GO, "./service/"+type_name+".go", v.replace, false)
	v.gen(template.API_GO, "./api/http/"+type_name+".go", v.replace, false)
	//v.gen("./test/test.go.tmpl", "./api/http/"+type_name+"_test.go", v.replace, false)

	v.overwrite(registerAPIFile, v.registerAPI)

	if dbType == "sql" {
		v.overwrite(setupTablesFile, v.setupTable)
	} else {
		v.overwrite(setupCollectionsFile, v.setupCollection)
	}

	v.overwrite("./xf/codegen/run.go", v.commentCodegen)

	fmt.Println("**DONE GENERATING**")
	os.Exit(0)
}

func fileExists(file string) bool {
	_, err := os.Stat(file)
	return !os.IsNotExist(err)
}

func (r *Vars) overwrite(file string, handler func(string) string) {
	if !fileExists(file) {
		panic(fmt.Errorf("\"%s\" doesn't exist", file))
	}

	tmplBytes, err := os.ReadFile(file)

	if err != nil {
		panic(err)
	}

	tmpl := string(tmplBytes)

	r.gen(tmpl, file, handler, true)
}

func (r *Vars) gen(template, sourceFile string, handler func(string) string, overwrite bool) {
	exists := fileExists(sourceFile)
	if !overwrite && exists {
		panic(fmt.Errorf("\"%s\" already exists", sourceFile))
	}

	source := handler(template)

	err := os.WriteFile(sourceFile, []byte(source), 0666)

	if err != nil {
		panic(err)
	}

	op := "ADDED"
	if exists {
		op = "MODIFIED"
	}

	fmt.Printf("%v %v\n", op, sourceFile)
}

func (r *Vars) append(sourceFile, code string) {
	bytes, err := os.ReadFile(sourceFile)

	if err != nil {
		panic(err)
	}

	sourceCode := string(bytes)

	sourceCode += code

	err = os.WriteFile(sourceFile, []byte(sourceCode), 0666)

	if err != nil {
		panic(err)
	}

	fmt.Printf("APPENDED %v\n", sourceFile)
}

func (r *Vars) replaceFileContent(file, tag, replacement string) {
	tmplBytes, err := os.ReadFile(file)

	if err != nil {
		panic(err)
	}

	tmpl := string(tmplBytes)

	source := strings.ReplaceAll(tmpl, tag, replacement)

	err = os.WriteFile(file, []byte(source), 0666)

	if err != nil {
		panic(err)
	}

	fmt.Printf("MODIFIED %v\n", file)
}

func (r *Vars) replace(tmpl string) string {
	source := strings.ReplaceAll(tmpl, "#TypeName#", r.TypeName)
	source = strings.ReplaceAll(source, "#typeName#", r.typeName)
	source = strings.ReplaceAll(source, "#type_name#", r.type_name)
	//source = strings.ReplaceAll(source, "#item_name#", r.item_name)
	source = strings.ReplaceAll(source, "#list_name#", r.list_name)
	source = strings.ReplaceAll(source, "#type-name#", r.type_dash_name)
	source = strings.ReplaceAll(source, "#idsQuote#", r.idsQuote)
	source = strings.ReplaceAll(source, "#glasud#", r.glasud)
	source = strings.ReplaceAll(source, "#Annotation#", r.Annotation)
	source = strings.ReplaceAll(source, "#ModelCommonType#", r.ModelCommonType)
	source = strings.ReplaceAll(source, "#DAOType#", r.DAOType)
	source = strings.ReplaceAll(source, "#NewDAOTypeFunParaVars#", r.NewDAOTypeFunParaVars)
	source = strings.ReplaceAll(source, "#tableName#", r.tableName)
	source = strings.ReplaceAll(source, "#mongoIndexCode#", r.mongoIndexCode)
	source = strings.ReplaceAll(source, "#ModuleName#", ModuleName)
	source = strings.ReplaceAll(source, "#MODULE_NAME#", MODULE_NAME)
	source = strings.ReplaceAll(source, "#tag#", r.tag)

	return source
}

const setupCollectionsFile = "./dao/mongo_register.go"
const setupCollectionAnchor = "//DO_NOT_TOUCH_THIS_COMMENT:SETUP_COLLECTION"
const setupCollectionTmpl = "xf.MustSetupMongoCollection(mongoDB, Collection#TypeName#, model.#TypeName#MongoValidator(), model.#TypeName#MongoIndexes())\n\t" + setupCollectionAnchor
const setupCollectionNameTmpl = "const Collection#TypeName# = \"#type_name#\"\n"

func (r *Vars) setupCollection(tmpl string) string {
	setupCollectionCode := r.replace(setupCollectionTmpl)

	source := strings.ReplaceAll(tmpl, setupCollectionAnchor, setupCollectionCode)

	setupCollectionNameCode := r.replace(setupCollectionNameTmpl)

	source = source + setupCollectionNameCode

	return source
}

const registerAPIFile = "./api/http/register_api.go"
const registerAPIAnchor = "//DO_NOT_TOUCH_THIS_COMMENT:REGISTER_API"
const registerAPITmpl = "#typeName#.registerAPI(root)\n\t" + registerAPIAnchor

func (r *Vars) registerAPI(tmpl string) string {
	code := r.replace(registerAPITmpl)

	source := strings.ReplaceAll(tmpl, registerAPIAnchor, code)

	return source
}

const genCall = "MustGenerateAndExit"
const genCallCommented = "//" + genCall

//const genImport = "\t\"erp/codegen\"\n"
//const genImportOneLine = "import \"erp/codegen\"\n"

func tmplImportCodegen() string {
	return "\t\"" + ModuleName + "/codegen\"\n"
}

func tmplImportCodegenOneLine() string {
	return "import \"" + ModuleName + "/codegen\"\n"
}

func (r *Vars) commentCodegen(tmpl string) string {
	if strings.Contains(tmpl, genCallCommented) {
		return tmpl
	}

	source := strings.ReplaceAll(tmpl, genCall, genCallCommented)
	source = strings.ReplaceAll(source, tmplImportCodegen(), "")
	source = strings.ReplaceAll(source, tmplImportCodegenOneLine(), "")

	return source
}

const setupTablesFile = "./dao/setup_tables.go"
const setupTableAnchor = "//DO_NOT_TOUCH_THIS_COMMENT:SETUP_TABLE"
const setupTableTmpl = "&#TypeName#{},\n\t\t" + setupTableAnchor
const setupTriggerAnchor = "//DO_NOT_TOUCH_THIS_COMMENT:APPEND_TRIGGER_DDL"
const setupTriggerTmpl = "triggerDDLs = append(triggerDDLs, model.#TypeName#TriggerDDL)\n\t" + setupTriggerAnchor

func (r *Vars) setupTable(tmpl string) string {
	code := r.replace(setupTableTmpl)

	source := strings.ReplaceAll(tmpl, setupTableAnchor, code)

	code = r.replace(setupTriggerTmpl)

	source = strings.ReplaceAll(source, setupTriggerAnchor, code)

	return source
}

//func (r *Vars) constructModelDBFuncs() {
//	r.ModelDBFuncs = "\nfunc (r *" + r.TypeName + ") TableName() string {\n" +
//		"\treturn \"" + r.tableName + "\"\n}\n"
//
//	r.ModelDBFuncs += "\nfunc " + r.TypeName + "TriggerDDL() string {\n" +
//		"\treturn \"create trigger `trg_" + r.tableName + "_updated_at` before update on `" + r.tableName + "` for each row set new.updated_at = now();\"\n}\n"
//}

const mongoModelTmpl = `
var #typeName#MongoIndexes = []mongo.IndexModel{
	xf.IndexID(), xf.IndexIsDeleted(),#mongoIndexCode#
}

func #TypeName#MongoIndexes() []mongo.IndexModel {
	return #typeName#MongoIndexes
}

var #typeName#MongoValidator = bson.M{}

func #TypeName#MongoValidator() bson.M {
	return #typeName#MongoValidator
}
`

const gormModelTmpl = `
func (r *#TypeName#) TableName() string {
	return "#tableName#"
}

const #TypeName#TriggerDDL = "create trigger ` + "`trg_#tableName#_updated_at`" + ` before update on ` + "`#tableName#`" + ` for each row set new.updated_at = CURRENT_TIMESTAMP(3);"
`

func (r *Vars) modelAppendCode() string {
	var code string

	if r._dbType == "sql" {
		code = gormModelTmpl
	} else {
		code = mongoModelTmpl

		var indexCode string

		for _, id := range r.ids {
			if id == "id" {
				// id有默认索引
				continue
			}
			indexCode += `
	{
		Keys: bson.M{"` + id + `": 1},
	},`
		}

		r.mongoIndexCode = indexCode
	}

	code = r.replace(code)

	return code
}

func (r *Vars) fixModelFile() {
	srcFile := "./model/" + r.type_name + ".go"

	if !fileExists(srcFile) {
		r.gen(template.MODEL_GO, srcFile, r.replace, false)
		r.fixModelFile()
		return
	}

	//r.append(srcFile, r.modelAppendCode())
	r.replaceFileContent(srcFile, "#ModelComplement#", r.modelAppendCode())

	fixImports(srcFile)
}

func fixImports(file string) {
	cmd := exec.Command("goimports", "-w", file)
	err := cmd.Run()

	if err != nil {
		fmt.Printf("[ERROR] run goimports failed. %v\n", err)
	}
}

func MustGenUnitTest(TypeName, typeName, type_name, Annotation string) {
	fmt.Println("**GENERATING UNIT TEST CODE**")

	v := newVars("", TypeName, typeName, type_name, "", "", Annotation, nil)

	v.gen(template.TEST_GO, "./test/"+type_name+"_test.go", v.replace, false)

	fmt.Println("**DONE GENERATING**")
}
