package template

const DAO_GO = `// #Annotation#
package dao

import (
	"#ModuleName#/model"
)

type #TypeName# = model.#TypeName#

type #TypeName#DAO struct {
	*xf.#DAOType#[#TypeName#, *#TypeName#]
}

func New#TypeName#DAO(ctx *xf.CTX) *#TypeName#DAO {
	return &#TypeName#DAO{
		#DAOType#: xf.New#DAOType#[#TypeName#, *#TypeName#](#NewDAOTypeFunParaVars#),
	}
}

func New#TypeName#DAOGeneric(ctx *xf.CTX) xf.DAO[#TypeName#, *#TypeName#] {
	return xf.New#DAOType#[#TypeName#, *#TypeName#](#NewDAOTypeFunParaVars#)
}
`

const DAO_INIT_GO = `package dao

import (
	"database/sql"
	"github.com/chris-sean/xf"
)

func Init() {
	xf.Infof("Initializing database related resources...")

	initMongo()

	createSchemaIfNotExists()

	// initGorm()

	upsertStaticData()
}

func configDB(db *sql.DB) {
	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(10)
}

func createSchemaIfNotExists() {
	//dbx.MustExec("CREATE SCHEMA IF NOT EXISTS ` + "`" + `mapping` + "`" + `")
}

func upsertStaticData() {

}

// 插入或更新静态数据。
//func upsertStaticData() {
//	funcSpecs := model.BuiltinFuncSpecs()
//	gdb.Clauses(clause.OnConflict{
//		Columns:   []clause.Column{{Day: "name"}}, // key colume
//		UpdateAll: true,
//		//DoUpdates: clause.AssignmentColumns([]string{"title", "desc", "struct_json"}),
//	}).Create(&funcSpecs)
//}
`

const DAO_MONGO_GO = `package dao

import (
	"context"
	"github.com/chris-sean/xf"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
	"ts-device-mgmt/config"
)

var mongoClient *mongo.Client

var mongoDB *mongo.Database

func initMongo() {
	// create mongoClient and mongoDBMapping
	opts := options.Client().ApplyURI(config.MongodbHost())
	client, err := mongo.Connect(context.Background(), opts)
	if err != nil {
		xf.Panic(xf.ErrMongoConnectionError(err))
	}

	err = client.Ping(context.Background(), nil)
	if err != nil {
		// mongodb server might finish starting later
		time.AfterFunc(10*time.Second, func() {
			err = client.Ping(context.Background(), nil)
			if err != nil {
				xf.Panic(xf.ErrMongoConnectionError(err))
			}
		})
	}

	mongoClient = client
	mongoDB = mongoClient.Database(mongoDBName)

	// create collection, index, validator etc.
	setupCollections()
}

func MongoDatabase(name string) *mongo.Database {
	return mongoClient.Database(name)
}
`

const DAO_MONGO_REGISTER_GO = `package dao

import (
	"github.com/chris-sean/xf"
	"#ModuleName#/model"
)

const mongoDBName = "#ModuleName#"

func setupCollections() {
	//DO_NOT_TOUCH_THIS_COMMENT:SETUP_COLLECTION
}
`
