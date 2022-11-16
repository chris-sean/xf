package xf

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDAO[T any, P CommonModel[T]] struct {
	*CTX
	client         *mongo.Client
	collection     *mongo.Collection
	sessionContext context.Context
}

func NewMongoDAO[T any, P CommonModel[T]](ctx *CTX, client *mongo.Client, collection *mongo.Collection) *MongoDAO[T, P] {
	return &MongoDAO[T, P]{
		CTX:            ctx,
		client:         client,
		collection:     collection,
		sessionContext: context.Background(),
	}
}

func (r *MongoDAO[T, P]) mapFields(m map[string]any) {
	MapFields[T](m, MapFieldsMethodJsonToBson)
}

// 如果是复杂慢查询，每页都获取总数浪费资源。所以将分页和查询总数接口分开。按需调用。
//func (r *MongoDAO[T, P]) fixPageTotal(page *xf.PageMeta, filter map[string]any) {
//	// get total
//	var total int64
//	var err error
//
//	//todo cache to redis for a while
//	total, err = r.collection.CountDocuments(r.sessionContext, filter)
//
//	if err != nil {
//		panic(ErrMongoQueryError(err))
//	}
//
//	page.Total = &total
//
//	//todo implement page.Start
//}

func (r *MongoDAO[T, P]) configurePage(findOptions *options.FindOptions, page *PageMeta, filter map[string]any) {
	configurePage(page)

	findOptions.SetSkip(((*page.Page) - 1) * page.Size)
	findOptions.SetLimit(page.Size)

	//r.fixPageTotal(page, filter)
}

type AggregateParameters struct {
	Page         *PageMeta
	Fields       map[string]any
	Lookups      []map[string]any
	BeforeLookup bson.A
	AfterLookup  bson.A
}

func AggregateLookup(from, localField, foreignField, as string, pipeline []bson.M) map[string]any {
	m := map[string]any{
		"from":         from,
		"localField":   localField,
		"foreignField": foreignField,
		"as":           as,
		//"pipeline":     []bson.M{{"$match": bson.M{FieldIsDeleted: notDeleted}}},
	}

	if len(pipeline) == 0 {
		m["pipeline"] = []bson.M{{"$match": bson.M{FieldIsDeleted: notDeleted}}}
	} else {
		var match bson.M
		for _, e := range pipeline {
			if ma, ok := e["$match"].(bson.M); ok {
				match = ma
				break
			}
		}

		if match == nil {
			match = bson.M{}
		}

		match[FieldIsDeleted] = notDeleted

		m["pipeline"] = pipeline
	}

	return m
}

func (r *MongoDAO[T, P]) MustGetByAggregate(paras *AggregateParameters, otherCommand ...any) []P {
	return r.MustGetByAggregate2(paras, nil, otherCommand...)
}

func (r *MongoDAO[T, P]) MustGetByAggregate2(paras *AggregateParameters, alterPipeline func(pipeline *bson.A), otherCommand ...any) []P {
	pipeline := r.pipelineFromPage(paras.Page)

	pipeline = append(pipeline, paras.BeforeLookup...)

	// lookup
	for _, lookup := range paras.Lookups {
		pipeline = append(pipeline, bson.M{
			"$lookup": lookup,
		})
	}

	pipeline = append(pipeline, paras.AfterLookup...)

	// projection
	if len(paras.Fields) > 0 {
		pipeline = append(pipeline, bson.M{
			"$project": paras.Fields,
		})
	}

	pipeline = append(pipeline, otherCommand...)

	opts := options.Aggregate()
	opts.SetCollation(DefaultCollation())

	if alterPipeline != nil {
		alterPipeline(&pipeline)
	}

	cursor, err := r.collection.Aggregate(r.sessionContext, pipeline, opts)

	if err != nil {
		panic(ErrMongoQueryError(err))
	}

	data := make([]P, 0, cursor.RemainingBatchLength())
	err = cursor.All(r.sessionContext, &data)

	if err != nil {
		panic(ErrMongoQueryError(err))
	}

	return data
}

func (r *MongoDAO[T, P]) preprocessFilter(filter map[string]any) {
	r.mapFields(filter)

	//// change id to _id
	//id, ok := filter["id"]
	//if ok {
	//	filter[FieldMongoID] = id
	//	delete(filter, "id")
	//}

	// find undeleted by default
	_, ok := filter[FieldIsDeleted]
	if !ok {
		filter[FieldIsDeleted] = notDeleted
	}
}

func (r *MongoDAO[T, P]) pipelineFromPage(page *PageMeta) bson.A {
	if page == nil {
		return bson.A{
			bson.M{"$match": bson.M{FieldIsDeleted: notDeleted}},
			bson.M{"$sort": bson.M{FieldID: -1}},
			bson.M{"$limit": 1000},
		}
	}

	var pipeline bson.A

	// match
	filter := page.Match

	if filter == nil {
		filter = map[string]any{}
	}
	r.preprocessFilter(filter)

	r.fixSearch(page.Search, filter)

	pipeline = append(pipeline, bson.M{"$match": filter})

	// sort
	sort := page.SortBy
	if len(sort) == 0 {
		sort = bson.M{FieldID: -1}
	} else {
		r.mapFields(sort)
	}

	pipeline = append(pipeline, bson.M{"$sort": sort, "$collation": DefaultCollation()})

	// skip
	if page.Page != nil && *page.Page > 1 {
		pipeline = append(pipeline, bson.M{
			"$skip": ((*page.Page) - 1) * page.Size,
		})
	}

	// limit
	var limit int64

	if page.Size > 0 && page.Size < 1000 {
		limit = page.Size
	} else {
		limit = 1000
	}

	pipeline = append(pipeline, bson.M{"$limit": limit})

	//r.fixPageTotal(page, filter)

	return pipeline
}

func (r *MongoDAO[T, P]) filterFromPage(page *PageMeta) map[string]any {
	filter := page.Match

	if filter == nil {
		filter = map[string]any{}
	}

	r.preprocessFilter(filter)

	r.fixSearch(page.Search, filter)

	return filter
}

func (r *MongoDAO[T, P]) fixSearch(search, filter bson.M) {
	if search == nil {
		return
	}
	r.mapFields(search)
	for k, v := range search {
		switch v.(type) {
		case bson.M: // $or $and
			m := v.(bson.M)
			a := make(bson.A, 0, len(m))
			for mk, mv := range m {
				a = append(a, bson.M{mk: primitive.Regex{Pattern: fmt.Sprintf("%v", mv)}})
			}
			filter[k] = a
		case map[string]any: // $or $and
			m := v.(map[string]any)
			a := make(bson.A, 0, len(m))
			for mk, mv := range m {
				a = append(a, bson.M{mk: primitive.Regex{Pattern: fmt.Sprintf("%v", mv)}})
			}
			filter[k] = a
		case []any:
			filter[k] = bson.M{"$in": v}
		default:
			filter[k] = primitive.Regex{Pattern: fmt.Sprintf("%v", v)}
		}
	}
}

func (r *MongoDAO[T, P]) MustGetPage(page *PageMeta, fields map[string]any) []P {
	// filter
	filter := r.filterFromPage(page)

	opt := options.Find()

	r.configurePage(opt, page, filter)

	// sort
	if len(page.SortBy) == 0 {
		opt.SetSort(bson.M{FieldID: -1})
	} else {
		sort := page.SortBy
		r.mapFields(sort)
		opt.SetSort(sort)
	}
	opt.SetCollation(DefaultCollation())

	r.mapFields(fields)
	if fields == nil {
		fields = map[string]any{}
	}
	// projection
	opt.SetProjection(fields)

	cursor, err := r.collection.Find(r.sessionContext, filter, opt)

	if err != nil {
		panic(ErrMongoQueryError(err))
	}

	data := make([]P, 0, cursor.RemainingBatchLength())
	err = cursor.All(r.sessionContext, &data)

	if err != nil {
		panic(ErrMongoQueryError(err))
	}

	return data
}

func (r *MongoDAO[T, P]) MustGetList(page *PageMeta, fields map[string]any) []P {
	page.Size = 1000
	return r.MustGetPage(page, fields)
}

func (r *MongoDAO[T, P]) preGet(filter, fields map[string]any) *options.FindOneOptions {
	if len(filter) == 0 {
		panic(ErrInvalidParameters("No filter"))
	}

	r.mapFields(fields)
	r.preprocessFilter(filter)

	opt := options.FindOne()

	opt.SetSort(bson.M{FieldID: -1})

	if fields == nil {
		fields = map[string]any{}
	}

	opt.SetProjection(fields)

	return opt
}

func (r *MongoDAO[T, P]) get(filter map[string]any, opt *options.FindOneOptions) P {
	sr := r.collection.FindOne(r.sessionContext, filter, opt)
	err := sr.Err()

	if err == mongo.ErrNoDocuments {
		var null P
		return null
	}

	if err != nil {
		panic(ErrMongoQueryError(err))
	}

	var doc P
	err = sr.Decode(&doc)

	if err != nil {
		panic(ErrMongoQueryError(err))
	}

	return doc
}

func (r *MongoDAO[T, P]) MustGet(filter, fields map[string]any) P {
	if len(filter) == 0 {
		panic(ErrInvalidParameters("No filter"))
	}

	opt := r.preGet(filter, fields)

	doc := r.get(filter, opt)

	return doc
}

func (r *MongoDAO[T, P]) Exist(filter map[string]any) bool {
	r.mapFields(filter)
	r.preprocessFilter(filter)

	count, err := r.collection.CountDocuments(r.sessionContext, filter)

	if err != nil {
		panic(ErrMongoQueryError(err))
	}

	return count > 0
}

func (r *MongoDAO[T, P]) MustGetCommonFields(id any) *CommonFields {
	if id == "" || id == 0 || id == nil {
		panic(ErrInvalidParameters(FieldID))
	}

	opt := options.FindOne()

	opt.SetProjection(bson.M{FieldCreatedAt: 1, FieldUpdatedAt: 1, FieldIsDeleted: 1})

	sr := r.collection.FindOne(r.sessionContext, bson.M{FieldID: id}, opt)
	err := sr.Err()

	if err == mongo.ErrNoDocuments {
		return nil
	}

	if err != nil {
		panic(ErrMongoQueryError(err))
	}

	var doc CommonFields
	err = sr.Decode(&doc)

	if err != nil {
		panic(ErrMongoQueryError(err))
	}

	return &doc
}

func (r *MongoDAO[T, P]) MustAdd(doc P) {
	now := time.Now()
	doc.SetCreatedAt(&now)
	doc.SetUpdatedAt(&now)
	doc.SetIsDeleted(false)
	doc.SetID(primitive.NewObjectID().Hex())

	_, err := r.collection.InsertOne(r.sessionContext, doc)

	if err != nil {
		panic(ErrMongoWriteError(err))
	}
}

func (r *MongoDAO[T, P]) MustAddMany(docs []P) {
	now := time.Now()
	for _, doc := range docs {
		doc.SetCreatedAt(&now)
		doc.SetUpdatedAt(&now)
		doc.SetIsDeleted(false)
		doc.SetID(primitive.NewObjectID().Hex())
	}

	all := make([]any, 0, len(docs))

	for _, doc := range docs {
		all = append(all, doc)
	}

	_, err := r.collection.InsertMany(r.sessionContext, all)

	if err != nil {
		panic(ErrMongoWriteError(err))
	}
}

func (r *MongoDAO[T, P]) MustUpdateMany(filter, updates map[string]any, advancedUpdates any) int64 {
	return r.mustUpdate(filter, updates, advancedUpdates, true, false)
}

func (r *MongoDAO[T, P]) MustUpdate(filter, updates map[string]any, advancedUpdates any) int64 {
	return r.mustUpdate(filter, updates, advancedUpdates, false, false)
}

func (r *MongoDAO[T, P]) MustUpsertMany(filter, updates map[string]any, advancedUpdates any) int64 {
	return r.mustUpdate(filter, updates, advancedUpdates, true, false)
}

func (r *MongoDAO[T, P]) MustUpsert(filter, updates map[string]any, advancedUpdates any) int64 {
	return r.mustUpdate(filter, updates, advancedUpdates, false, false)
}

func (r *MongoDAO[T, P]) mustUpdate(filter, updates map[string]any, advancedUpdates any, updateMany bool, upsert bool) int64 {
	if len(filter) == 0 {
		panic(ErrInvalidParameters("No filter"))
	}

	updatesLen := len(updates)
	if updatesLen == 0 && IsValueNil(advancedUpdates) {
		return 0
	}

	r.mapFields(filter)
	r.mapFields(updates)

	MustValidateMap[T](updates)

	if updates == nil {
		updates = bson.M{}
	}

	regulateUpdates(updates)

	//// change id to _id
	//id, ok := filter["id"]
	//if ok {
	//	filter[FieldID] = id
	//	delete(filter, "id")
	//}

	// remove forbidden fields
	delete(updates, FieldID)

	filter[FieldIsDeleted] = notDeleted

	var up any

	if updatesLen > 0 {
		up = bson.M{"$set": updates}
	} else {
		up = advancedUpdates
	}

	updateFn := r.collection.UpdateOne

	if updateMany {
		updateFn = r.collection.UpdateMany
	}

	var result *mongo.UpdateResult
	var err error

	if upsert {
		opt := options.Update()
		opt.SetUpsert(true)
		result, err = updateFn(r.sessionContext, filter, up, opt)
	} else {
		result, err = updateFn(r.sessionContext, filter, up)
	}

	if err != nil {
		panic(ErrMongoWriteError(err))
	}

	return result.ModifiedCount
}

func (r *MongoDAO[T, P]) SessionContext() context.Context {
	return r.sessionContext
}

func (r *MongoDAO[T, P]) MustAggregate(pipeline []bson.M, customDecoding bool) ([]P, *mongo.Cursor) {
	cursor, err := r.collection.Aggregate(r.sessionContext, pipeline)

	if err != nil {
		panic(ErrMongoQueryError(err))
	}

	if customDecoding {
		return nil, cursor
	}

	data := make([]P, 0, cursor.RemainingBatchLength())
	err = cursor.All(r.sessionContext, &data)

	if err != nil {
		panic(ErrMongoQueryError(err))
	}

	return data, nil
}

func (r *MongoDAO[T, P]) MustDistinct(field string, filter bson.M) []any {
	r.preprocessFilter(filter)

	result, err := r.collection.Distinct(r.sessionContext, field, filter)

	if err != nil {
		panic(ErrMongoQueryError(err))
	}

	return result
}

func (r *MongoDAO[T, P]) MustSave(doc P) {
	id := doc.GetID()
	if id == "" || id == 0 || id == nil {
		panic(ErrInvalidParameters(FieldID))
	}

	ocf := r.MustGetCommonFields(id)

	if ocf.GetIsDeleted() {
		// It's forbidden to modify deleted entry
		return
	}

	if ocf == nil {
		panic(ErrNotFound(""))
	}

	doc.SetCreatedAt(ocf.GetCreatedAt())
	now := time.Now()
	doc.SetUpdatedAt(&now)
	doc.SetIsDeleted(ocf.GetIsDeleted())

	result, err := r.collection.ReplaceOne(r.sessionContext, bson.M{FieldID: id}, doc)

	if err != nil {
		panic(ErrMongoWriteError(err))
	}

	if result.MatchedCount == 0 {
		panic(ErrNotFound(""))
	}
}

func (r *MongoDAO[T, P]) MustSoftDelete(filter map[string]any) int64 {
	return r.mustSoftDelete(filter, false)
}

func (r *MongoDAO[T, P]) MustSoftDeleteMany(filter map[string]any) int64 {
	return r.mustSoftDelete(filter, true)
}

func (r *MongoDAO[T, P]) mustSoftDelete(filter map[string]any, deleteAll bool) int64 {
	if len(filter) == 0 {
		panic(ErrInvalidParameters("No filter"))
	}

	r.mapFields(filter)

	//// change id to _id
	//id, ok := filter["id"]
	//if ok {
	//	filter[FieldID] = id
	//	delete(filter, "id")
	//}

	filter[FieldIsDeleted] = notDeleted

	up := bson.M{
		"$set": bson.M{FieldIsDeleted: deleted},
	}

	updateFunc := r.collection.UpdateOne

	if deleteAll {
		updateFunc = r.collection.UpdateMany
	}

	result, err := updateFunc(r.sessionContext, filter, up)

	if err != nil {
		panic(ErrMongoWriteError(err))
	}

	return result.ModifiedCount
}

func (r *MongoDAO[T, P]) MustHardDelete(filter map[string]any) int64 {
	return r.mustHardDelete(filter, false)
}

func (r *MongoDAO[T, P]) MustHardDeleteMany(filter map[string]any) int64 {
	return r.mustHardDelete(filter, true)
}

func (r *MongoDAO[T, P]) mustHardDelete(filter map[string]any, deleteAll bool) int64 {
	if len(filter) == 0 {
		panic(ErrInvalidParameters("No filter"))
	}

	r.mapFields(filter)

	deletionFunc := r.collection.DeleteOne

	if deleteAll {
		deletionFunc = r.collection.DeleteMany
	}

	result, err := deletionFunc(r.sessionContext, filter)

	if err != nil {
		panic(ErrMongoWriteError(err))
	}

	return result.DeletedCount
}

func (r *MongoDAO[T, P]) UseSession(sessionContext context.Context) {
	r.sessionContext = sessionContext
}

// Transaction only available for replica set
func (r *MongoDAO[T, P]) Transaction(do func(session context.Context)) {
	session := r.startTransaction()
	defer r.endTransaction()

	defer func() {
		if err := recover(); err != nil {
			r.abortTransaction()
			panic(err)
		}
	}()

	do(session)

	r.commitTransaction()
}

func (r *MongoDAO[T, P]) startTransaction() context.Context {
	session, err := r.client.StartSession()

	if err != nil {
		panic(ErrServerInternalError(fmt.Errorf("Failed to start a session. %v", err)))
	}

	r.sessionContext = mongo.NewSessionContext(context.Background(), session)

	err = session.StartTransaction()

	if err != nil {
		panic(ErrServerInternalError(err))
	}

	return r.sessionContext
}

func (r *MongoDAO[T, P]) commitTransaction() {
	if sc, ok := r.sessionContext.(mongo.SessionContext); ok {
		err := sc.CommitTransaction(sc)
		if err != nil {
			panic(ErrMongoTransactionError(err))
		}
	}
}

func (r *MongoDAO[T, P]) abortTransaction() {
	if sc, ok := r.sessionContext.(mongo.SessionContext); ok {
		err := sc.AbortTransaction(context.Background())
		if err != nil {
			panic(ErrMongoTransactionError(err))
		}
	}
}

func (r *MongoDAO[T, P]) endTransaction() {
	if sc, ok := r.sessionContext.(mongo.SessionContext); ok {
		sc.EndSession(sc)
	}
}

func (r *MongoDAO[T, P]) MustAddJSON(json map[string]any) any {
	delete(json, FieldIsDeleted)

	json[FieldID] = primitive.NewObjectID().Hex()
	now := time.Now()
	json[FieldCreatedAt] = &now
	json[FieldUpdatedAt] = &now

	result, err := r.collection.InsertOne(r.sessionContext, json)

	if err != nil {
		panic(ErrMongoWriteError(err))
	}

	return result.InsertedID
}

func (r *MongoDAO[T, P]) MustCount(page *PageMeta) int64 {
	filter := r.filterFromPage(page)

	count, err := r.collection.CountDocuments(r.sessionContext, filter)

	if err != nil {
		panic(ErrMongoQueryError(err))
	}

	return count
}
