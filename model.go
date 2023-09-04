package mgs

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	int "github.com/0x-buidl/mgs/internal"
	mopt "github.com/0x-buidl/mgs/options"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type sessFn func(sessCtx mongo.SessionContext) (interface{}, error)

type Model[T Schema, P IDefaultSchema] struct {
	collection *mongo.Collection
}

// NewModel creates a new model. T represents the schema type while P represents default schema type.
// Panics if T or P is not a struct .
func NewModel[T Schema, P IDefaultSchema](collection *mongo.Collection) *Model[T, P] {
	var defSchema P
	defType := reflect.ValueOf(defSchema).Type().Elem()
	if defType.Kind() != reflect.Struct {
		panic("DefaultSchema must be a struct")
	}

	var t T
	if reflect.ValueOf(t).Type().Kind() != reflect.Struct {
		panic("Schema must be a struct")
	}

	return &Model[T, P]{collection}
}

// Collection returns the [*mongo.Collection] that the model is using.
func (model *Model[T, P]) Collection() *mongo.Collection {
	return model.collection
}

// NewDocument creates a new [*Document] with the given data.
func (model *Model[T, P]) NewDocument(data T) *Document[T, P] {
	defType := reflect.ValueOf(*new(P)).Type().Elem()
	defSchema := reflect.New(defType).Interface().(P)

	doc := Document[T, P]{
		IDefaultSchema: defSchema,
		Doc:            &data,
		collection:     model.collection,
		doc:            data,
		isNew:          true,
	}

	doc.GenerateID()
	doc.GenerateCreatedAt()
	doc.GenerateUpdatedAt()

	return &doc
}

// CreateOne creates a single document in the collection.
// It returns the created document or an error if one occurred.
// Document is not created if any of the hooks return an error.
func (model *Model[T, P]) CreateOne(
	ctx context.Context, doc T,
	opts ...*mopt.InsertOneOptions,
) (*Document[T, P], error) {
	newDoc := model.NewDocument(doc)

	callback := func(sCtx mongo.SessionContext) (interface{}, error) {
		arg := newHookArg[T](newDoc, CreateOne)
		if err := runValidateHooks(ctx, newDoc, arg); err != nil {
			return nil, err
		}

		err := runBeforeCreateHooks(ctx, newDoc, arg)
		if err != nil {
			return nil, err
		}

		_, iopt := mopt.MergeInsertOneOptions(opts...)

		_, err = model.collection.InsertOne(sCtx, newDoc, iopt)
		if err != nil {
			return nil, err
		}

		newDoc.isNew = false

		err = runAfterCreateHooks(sCtx, newDoc, newHookArg[T](newDoc, CreateOne))
		return nil, err
	}

	_, err := withTransaction(ctx, model.collection, callback)
	if err != nil {
		return nil, err
	}

	return newDoc, nil
}

// CreateMany creates multiple documents in the collection.
// It returns the created documents or an error if one occurred.
// Documents are not created if any of the hooks return an error.
func (model *Model[T, P]) CreateMany(
	ctx context.Context,
	docs []T,
	opts ...*mopt.InsertManyOptions,
) ([]*Document[T, P], error) {
	callback := func(sCtx mongo.SessionContext) (interface{}, error) {
		newDocs, docsToInsert, err := model.beforeCreateMany(ctx, docs)
		if err != nil {
			return nil, err
		}

		_, iopt := mopt.MergeInsertManyOptions(opts...)
		_, err = model.collection.InsertMany(sCtx, docsToInsert, iopt)
		if err != nil {
			return nil, err
		}

		for _, doc := range newDocs {
			doc.isNew = false
		}
		ds := model.docSample()
		err = runAfterCreateHooks(sCtx, ds, newHookArg[T](&newDocs, CreateMany))
		return newDocs, err
	}

	newDocs, err := withTransaction(ctx, model.collection, callback)
	if err != nil {
		return nil, err
	}

	return newDocs.([]*Document[T, P]), nil
}

// DeleteOne deletes a single document from the collection.
// It returns the deleted result or an error if one occurred.
// Document is not deleted if any of the hooks return an error.
func (model *Model[T, P]) DeleteOne(
	ctx context.Context,
	query bson.M,
	opts ...*options.DeleteOptions,
) (*mongo.DeleteResult, error) {
	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		ds := model.docSample()

		qarg := NewQuery[T]().SetFilter(&query).SetOperation(DeleteOne).SetOptions(opts)
		err := runBeforeDeleteHooks(ctx, ds, newHookArg[T](qarg, DeleteOne))
		if err != nil {
			return nil, err
		}

		res, err := model.collection.DeleteOne(ctx, query, opts...)
		if err != nil {
			return nil, err
		}

		err = runAfterDeleteHooks(sessCtx, ds, newHookArg[T](res, DeleteOne))
		return res, err
	}

	res, err := withTransaction(ctx, model.collection, callback)
	if err != nil {
		return nil, err
	}

	return res.(*mongo.DeleteResult), err
}

// DeleteMany deletes multiple documents from the collection.
// It returns the deleted result or an error if one occurred.
// Documents are not deleted if any of the hooks return an error.
func (model *Model[T, P]) DeleteMany(
	ctx context.Context,
	query bson.M,
	opts ...*options.DeleteOptions,
) (*mongo.DeleteResult, error) {
	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		ds := model.docSample()

		qarg := NewQuery[T]().SetFilter(&query).SetOperation(DeleteMany).SetOptions(opts)
		err := runBeforeDeleteHooks(ctx, ds, newHookArg[T](qarg, DeleteMany))
		if err != nil {
			return nil, err
		}

		res, err := model.collection.DeleteMany(ctx, query, opts...)
		if err != nil {
			return nil, err
		}

		err = runAfterDeleteHooks(sessCtx, ds, newHookArg[T](res, DeleteMany))
		return res, err
	}

	res, err := withTransaction(ctx, model.collection, callback)
	if err != nil {
		return nil, err
	}

	return res.(*mongo.DeleteResult), err
}

// FindById finds a single document by its id.
// It returns the document or an error if one occurred.
// If no document is found, it returns [mongo.ErrNoDocuments].
func (model *Model[T, P]) FindById(
	ctx context.Context, id any,
	opts ...*mopt.FindOneOptions,
) (*Document[T, P], error) {
	oid, err := getObjectId(id)
	if err != nil {
		return nil, err
	}

	doc := model.docSample()

	query := bson.M{}
	qarg := NewQuery[T]().SetFilter(&query).SetOperation(FindOne).SetOptions(opts)
	err = runBeforeFindHooks(ctx, doc, newHookArg[T](qarg, FindOne))
	if err != nil {
		return nil, err
	}

	query["_id"] = *oid

	qopt, fopt := mopt.MergeFindOneOptions(opts...)

	if qopt.PopulateOption != nil {
		opt := mopt.FindOne()
		opt.FindOneOptions = fopt
		opt.QueryOptions = qopt

		docs, err := findWithPopulate[*mopt.FindOneOptions, T, P](
			ctx, model.collection, query, doc.doc, opt)
		if err != nil {
			return nil, err
		}

		if len(docs) == 0 {
			return nil, mongo.ErrNoDocuments
		}
	} else {
		err = model.collection.FindOne(ctx, bson.M{"_id": *oid}, fopt).Decode(doc)
		if err != nil {
			return nil, err
		}
	}

	doc.collection = model.Collection()

	err = runAfterFindHooks(ctx, doc, newHookArg[T](doc, FindOne))
	if err != nil {
		return nil, err
	}

	return doc, nil
}

// FindOne finds a single document from the collection.
// It returns the document or an error if one occurred.
// If no document is found, it returns [mongo.ErrNoDocuments].
func (model *Model[T, P]) FindOne(
	ctx context.Context,
	query bson.M,
	opts ...*mopt.FindOneOptions,
) (*Document[T, P], error) {
	doc := model.docSample()

	qarg := NewQuery[T]().SetFilter(&query).SetOperation(FindOne).SetOptions(opts)
	err := runBeforeFindHooks(ctx, doc, newHookArg[T](qarg, FindOne))
	if err != nil {
		return nil, err
	}

	qopt, fopt := mopt.MergeFindOneOptions(opts...)
	if qopt.PopulateOption != nil {
		opt := mopt.FindOne()
		opt.FindOneOptions = fopt
		opt.QueryOptions = qopt

		docs, err := findWithPopulate[*mopt.FindOneOptions, T, P](
			ctx, model.collection, query, doc.doc, opt)
		if err != nil {
			return nil, err
		}

		if len(docs) == 0 {
			return nil, mongo.ErrNoDocuments
		}
	} else {
		err = model.collection.FindOne(ctx, query, fopt).Decode(doc)
		if err != nil {
			return nil, err
		}
	}

	doc.collection = model.collection

	err = runAfterFindHooks(ctx, doc, newHookArg[T](doc, FindOne))
	if err != nil {
		return nil, err
	}

	return doc, nil
}

// Find finds multiple documents from the collection.
// It returns the documents or an error if one occurred.
func (model *Model[T, P]) Find(
	ctx context.Context,
	query bson.M,
	opts ...*mopt.FindOptions,
) ([]*Document[T, P], error) {
	d := model.docSample()

	qarg := NewQuery[T]().SetFilter(&query).SetOperation(FindMany).SetOptions(opts)
	err := runBeforeFindHooks(ctx, d, newHookArg[T](qarg, FindMany))
	if err != nil {
		return nil, err
	}

	docs := make([]*Document[T, P], 0)

	qopt, fopt := mopt.MergeFindOptions(opts...)
	if qopt.PopulateOption != nil {
		opt := mopt.Find()
		opt.FindOptions = fopt
		opt.QueryOptions = qopt
		docs, err = findWithPopulate[*mopt.FindOptions, T, P](
			ctx, model.collection, query,
			d.doc, opt)
		if err != nil {
			return nil, err
		}
	} else {
		cursor, err := model.collection.Find(ctx, query, fopt)
		if err != nil {
			return nil, err
		}

		err = cursor.All(ctx, &docs)
		if err != nil {
			return nil, err
		}
	}

	for _, doc := range docs {
		doc.collection = model.collection
	}

	err = runAfterFindHooks(ctx, d, newHookArg[T](&docs, FindMany))
	if err != nil {
		return nil, err
	}

	return docs, nil
}

// func (model *Model[T, P]) FindOneAndUpdate(
// 	ctx context.Context,
// 	query bson.M,
// 	update bson.M,
// 	opts ...*options.FindOneAndUpdateOptions,
// ) (*Document[T, P], error) {
// doc := &Document[T,P]{}
// err := model.collection.FindOneAndUpdate(ctx, query, update, opts...).Decode(doc)
// if err != nil {
// 	return nil, err
// }
// doc.collection = model.collection
// return doc, nil
// 	return nil, nil
// }

// UpdateOne updates a single document in the collection.
// It returns the update result or an error if one occurred.
// Document is not updated if any of the hooks return an error.
func (model *Model[T, P]) UpdateOne(ctx context.Context,
	query bson.M, update bson.M,
	opts ...*options.UpdateOptions,
) (*mongo.UpdateResult, error) {
	ds := model.docSample()

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		qa := NewQuery[T]().SetFilter(&query).
			SetUpdate(&update).
			SetOperation(UpdateOne).
			SetOptions(opts)

		err := runBeforeUpdateHooks(ctx, ds, newHookArg[T](qa, UpdateOne))
		if err != nil {
			return nil, err
		}

		if ut := ds.GetUpdatedAtTag("bson"); ut != "" && ut != "-" {
			if _, ok := update["$set"]; ok {
				update["$set"].(bson.M)[ut] = time.Now()
			} else {
				update["$set"] = bson.M{ut: time.Now()}
			}
		}

		res, err := model.collection.UpdateOne(sessCtx, query, update, opts...)
		if err != nil {
			return nil, err
		}

		err = runAfterUpdateHooks(sessCtx, ds, newHookArg[T](res, UpdateOne))
		return res, err
	}
	res, err := withTransaction(ctx, model.collection, callback)
	if err != nil {
		return nil, err
	}
	return res.(*mongo.UpdateResult), nil
}

// UpdateMany updates multiple documents in the collection.
// It returns the update result or an error if one occurred.
// Documents are not updated if any of the hooks return an error.
func (model *Model[T, P]) UpdateMany(ctx context.Context,
	query bson.M, update bson.M, opts ...*options.UpdateOptions,
) (*mongo.UpdateResult, error) {
	ds := model.docSample()

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		qa := NewQuery[T]().SetFilter(&query).
			SetUpdate(&update).
			SetOperation(UpdateMany).
			SetOptions(opts)

		err := runBeforeUpdateHooks(ctx, ds, newHookArg[T](qa, UpdateMany))
		if err != nil {
			return nil, err
		}

		if ut := ds.GetUpdatedAtTag("bson"); ut != "" && ut != "-" {
			if _, ok := update["$set"]; ok {
				update["$set"].(bson.M)[ut] = time.Now()
			} else {
				update["$set"] = bson.M{ut: time.Now()}
			}
		}
		res, err := model.collection.UpdateMany(sessCtx, query, update, opts...)
		if err != nil {
			return nil, err
		}
		err = runAfterUpdateHooks(sessCtx, ds, newHookArg[T](res, UpdateMany))
		return res, err
	}

	res, err := withTransaction(ctx, model.collection, callback)
	if err != nil {
		return nil, err
	}
	return res.(*mongo.UpdateResult), nil
}

// func (model *Model[T, P]) CountDocuments(ctx context.Context,
// 	query bson.M, opts ...*options.CountOptions,
// ) (int64, error) {
// 	return model.collection.CountDocuments(ctx, query, opts...)
// }

// func (model *Model[T, P]) Aggregate(
// 	ctx context.Context,
// 	pipeline mongo.Pipeline,
// 	res interface{},
// ) error {
// 	cursor, err := model.collection.Aggregate(ctx, pipeline)
// 	if err != nil {
// 		return err
// 	}
//
// 	return cursor.All(ctx, res)
// }

func (model *Model[T, P]) docSample() *Document[T, P] {
	data := *new(T)

	defType := reflect.ValueOf(*new(P)).Type().Elem()
	defSchema := reflect.New(defType).Interface().(P)

	doc := Document[T, P]{
		IDefaultSchema: defSchema,
		Doc:            &data,
		collection:     model.collection,
		doc:            data,
		isNew:          false,
	}
	return &doc
}

func findWithPopulate[U int.UnionFindOpts, T Schema, P IDefaultSchema](
	ctx context.Context, c *mongo.Collection,
	q bson.M, d T, opt U,
) ([]*Document[T, P], error) {
	pipelineOpts, aggrOpts, queryOpts := int.MergeFindOptsWithAggregatOpts(opt)
	pipeline := append(mongo.Pipeline{bson.D{{Key: "$match", Value: q}}}, pipelineOpts...)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var err error
	for _, pop := range *queryOpts.PopulateOption {
		wg.Add(1)
		func(pop *mopt.PopulateOptions) {
			defer wg.Done()
			mu.Lock()
			if err != nil {
				mu.Unlock()
				return
			}
			mu.Unlock()

			pipe, pErr := getPopulateStages(d, pop)

			mu.Lock()
			if pErr != nil {
				err = pErr
				mu.Unlock()
				return
			}
			pipeline = append(pipeline, pipe...)
			mu.Unlock()
		}(pop)
	}
	wg.Wait()

	docs := make([]*Document[T, P], 0)
	cursor, err := c.Aggregate(ctx, pipeline, aggrOpts)
	if err != nil {
		return nil, err
	}

	err = cursor.All(ctx, &docs)
	if err != nil {
		return nil, err
	}
	return docs, nil
}

func withTransaction(
	ctx context.Context,
	coll *mongo.Collection,
	fn sessFn,
	opts ...*options.TransactionOptions,
) (interface{}, error) {
	session, err := coll.Database().Client().StartSession()
	if err != nil {
		return nil, err
	}
	defer session.EndSession(ctx)

	res, err := session.WithTransaction(ctx, fn, opts...)
	return res, err
}

func getObjectId(id any) (*primitive.ObjectID, error) {
	var oid primitive.ObjectID
	switch id := id.(type) {
	case primitive.ObjectID:
		oid = id
	case string:
		var err error
		oid, err = primitive.ObjectIDFromHex(id)
		if err != nil {
			return nil, err
		}
	default:
		return nil, primitive.ErrInvalidHex
	}
	return &oid, nil
}

// TODO: custom populate errors
func getPopulateStages(doc any, opt *mopt.PopulateOptions) (mongo.Pipeline, error) {
	lookupPipeline := mongo.Pipeline{
		bson.D{
			{
				Key: "$match",
				Value: bson.M{
					"$expr": bson.M{"$eq": bson.A{"$$localField", "$" + *opt.ForeignField}},
				},
			},
		},
	}

	// populated nested populations
	lookups := make([]bson.D, 0)
	if opt.Populate != nil {
		var wg sync.WaitGroup
		var mu sync.Mutex
		var err error
		for _, p := range *opt.Populate {
			wg.Add(1)
			go func(p *mopt.PopulateOptions) {
				defer wg.Done()
				mu.Lock()
				if err != nil {
					mu.Unlock()
					return
				}
				mu.Unlock()

				pipe, pErr := getPopulateStages(opt.Schema, p)

				mu.Lock()
				if pErr != nil {
					err = pErr
					mu.Unlock()
					return
				}
				lookups = append(lookups, pipe...)
				mu.Unlock()
			}(p)
		}
		wg.Wait()
	}

	// merge options into aggregate pipeline
	if opt.Options != nil {
		popt := opt.Options
		if popt.Sort != nil {
			lookupPipeline = append(lookupPipeline, bson.D{{Key: "$sort", Value: popt.Sort}})
		}
		if popt.Skip != nil {
			lookupPipeline = append(lookupPipeline, bson.D{{Key: "$skip", Value: popt.Skip}})
		}
		var limit int64 = 1
		if popt.Limit != nil && !*opt.OnlyOne {
			limit = *popt.Limit
		}
		lookupPipeline = append(lookupPipeline, bson.D{{Key: "$limit", Value: limit}})
		if len(lookups) > 0 {
			lookupPipeline = append(lookupPipeline, lookups...)
		}
		if popt.Projection != nil {
			lookupPipeline = append(
				lookupPipeline,
				bson.D{{Key: "$project", Value: popt.Projection}},
			)
		}
	} else {
		if *opt.OnlyOne {
			lookupPipeline = append(lookupPipeline, bson.D{{Key: "$limit", Value: 1}})
		}
		if len(lookups) > 0 {
			lookupPipeline = append(lookupPipeline, lookups...)
		}
	}

	lookup := bson.M{
		"from":     *opt.Collection,
		"let":      bson.M{"localField": "$" + *opt.Path},
		"pipeline": lookupPipeline,
		"as":       *opt.Path,
	}
	populatePipeline := mongo.Pipeline{}
	v, t := reflect.ValueOf(doc), reflect.TypeOf(doc)

	// document must be a struct
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("document must be a struct to populate nested path")
	}

	paths := strings.Split(*opt.Path, ".")
	windPaths := make([]bson.D, 0)
	// traverse and unwind all slice fields
	for len(paths) > 0 {
		currPath := paths[0] // set current path
		// get all bson tags of the struct
		fields := getStructFields(t)

		// get the struct field of the path to populate
		field, ok := fields[currPath]
		if !ok {
			return nil, fmt.Errorf("field %s not found in struct", currPath)
		}

		// check if it's a pointer
		ft := field.Type
		if ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}

		switch {
		case ft.Kind() == reflect.Slice:
			elemType := ft.Elem()
			if elemType.Kind() == reflect.Pointer {
				elemType = elemType.Elem()
			}
			// check if it's slice of struct
			if len(paths) > 1 && elemType.Kind() != reflect.Struct {
				return nil, fmt.Errorf("field %s must be a slice of struct", currPath)
			}
			// if the field is a slice, we need to unwind it
			populatePipeline = append(
				populatePipeline,
				bson.D{{Key: "$unwind", Value: "$" + currPath}},
			)
			group := bson.E{
				Key:   "$group",
				Value: bson.M{"_id": "$_id"},
			}

			if len(paths) < len(strings.Split(*opt.Path, ".")) {
				group.Value.(bson.M)["_id"] = nil
			}

			// wind paths in descending order
			for k := range fields {
				if k == currPath {
					group.Value.(bson.M)[k] = bson.M{"$push": "$" + k}
				} else {
					group.Value.(bson.M)[k] = bson.M{"$first": "$" + k}
				}
			}
			windPaths = append([]bson.D{{group}}, windPaths...)
			t = elemType
		case ft.Kind() == reflect.Struct:
			t = ft
		default:
			if len(paths) > 1 {
				return nil, fmt.Errorf("field %s must be a struct or slice of struct", currPath)
			}
		}
		paths = paths[1:] // trim off first key

	}

	populatePipeline = append(populatePipeline, bson.D{{Key: "$lookup", Value: lookup}})
	if *opt.OnlyOne {
		populatePipeline = append(
			populatePipeline,
			bson.D{
				{
					Key:   "$unwind",
					Value: bson.M{"path": "$" + *opt.Path, "preserveNullAndEmptyArrays": true},
				},
			},
		)
	}
	populatePipeline = append(populatePipeline, windPaths...)
	return populatePipeline, nil
}

func getStructFields(t reflect.Type) map[string]reflect.StructField {
	fields := make(map[string]reflect.StructField)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("bson")
		if tag != "" && tag != "-" {
			fields[tag] = field
		}
	}
	return fields
}
