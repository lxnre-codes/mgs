package mgs

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	mopt "github.com/0x-buidl/go-mongoose/options"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type sessFn func(sessCtx mongo.SessionContext) (interface{}, error)

type Model[T Schema] struct {
	collection *mongo.Collection
}

func NewModel[T Schema](collection *mongo.Collection) *Model[T] {
	return &Model[T]{collection}
}

func (model *Model[T]) Collection() *mongo.Collection {
	return model.collection
}

func (model *Model[T]) NewDocument(data T) *Document[T] {
	doc := Document[T]{
		Doc:        &data,
		collection: model.collection,
		doc:        data,
		isNew:      true,
	}
	doc.generateId()
	doc.generateCreatedAt()
	doc.generateUpdatedAt()
	return &doc
}

func (model *Model[T]) CreateOne(
	ctx context.Context, doc T,
	opts ...*mopt.InsertOneOptions,
) (*Document[T], error) {
	newDoc := model.NewDocument(doc)

	err := runBeforeCreateHooks(ctx, newDoc.Doc, newHookArg[T](newDoc, CreateQuery))
	if err != nil {
		return nil, err
	}

	_, iopt := mopt.MergeInsertOneOptions(opts...)

	callback := func(sCtx mongo.SessionContext) (interface{}, error) {
		_, err = model.collection.InsertOne(sCtx, newDoc, iopt)
		if err != nil {
			return nil, err
		}

		err = runAfterCreateHooks(sCtx, newDoc.Doc, newHookArg[T](newDoc, CreateQuery))
		return nil, err
	}

	_, err = withTransaction(ctx, model.collection, callback)
	if err != nil {
		return nil, err
	}

	newDoc.isNew = false
	newDoc.collection = model.collection

	return newDoc, nil
}

func (model *Model[T]) CreateMany(
	ctx context.Context,
	docs []T,
	opts ...*mopt.InsertManyOptions,
) ([]*Document[T], error) {
	newDocs, docsToInsert, err := model.beforeCreateMany(ctx, docs)
	if err != nil {
		return nil, err
	}

	_, iopt := mopt.MergeInsertManyOptions(opts...)

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		_, err := model.collection.InsertMany(sessCtx, docsToInsert, iopt)
		if err != nil {
			return nil, err
		}

		err = model.afterCreateMany(sessCtx, newDocs)
		return nil, err
	}

	_, err = withTransaction(ctx, model.collection, callback)
	if err != nil {
		return nil, err
	}

	return newDocs, nil
}

func (model *Model[T]) DeleteOne(
	ctx context.Context,
	query bson.M,
	opts ...*options.DeleteOptions,
) (*mongo.DeleteResult, error) {
	d := (&Document[T]{}).Doc

	qarg := NewQuery[T]().SetFilter(query).SetOperation(DeleteQuery).SetOptions(opts)

	err := runBeforeDeleteHooks(ctx, d, newHookArg[T](qarg, DeleteQuery))
	if err != nil {
		return nil, err
	}

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		res, err := model.collection.DeleteOne(ctx, query, opts...)
		if err != nil {
			return nil, err
		}
		err = runAfterDeleteHooks(sessCtx, d, newHookArg[T](res, DeleteQuery))
		if err != nil {
			return nil, err
		}
		return res, nil
	}
	res, err := withTransaction(ctx, model.collection, callback)
	if err != nil {
		return nil, err
	}
	return res.(*mongo.DeleteResult), err
}

func (model *Model[T]) DeleteMany(
	ctx context.Context,
	query bson.M,
	opts ...*options.DeleteOptions,
) (*mongo.DeleteResult, error) {
	d := (&Document[T]{}).Doc

	qarg := NewQuery[T]().SetFilter(query).SetOperation(DeleteQuery).SetOptions(opts)

	err := runBeforeDeleteHooks(ctx, d, newHookArg[T](qarg, DeleteQuery))
	if err != nil {
		return nil, err
	}

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		res, err := model.collection.DeleteMany(ctx, query, opts...)
		if err != nil {
			return nil, err
		}

		err = runAfterDeleteHooks(sessCtx, d, newHookArg[T](res, DeleteQuery))
		if err != nil {
			return nil, err
		}
		return res, nil
	}

	res, err := withTransaction(ctx, model.collection, callback)
	return res.(*mongo.DeleteResult), err
}

func (model *Model[T]) FindById(
	ctx context.Context, id any,
	opts ...*mopt.FindOneOptions,
) (*Document[T], error) {
	oid, err := getObjectId(id)
	if err != nil {
		return nil, err
	}

	doc := &Document[T]{}

	query := bson.M{}
	qarg := NewQuery[T]().SetFilter(query).SetOperation(FindQuery).SetOptions(opts)

	err = runBeforeFindHooks(ctx, doc.Doc, newHookArg[T](qarg, FindQuery))
	if err != nil {
		return nil, err
	}
	query["_id"] = *oid

	qopt, fopt := mopt.MergeFindOneOptions(opts...)

	if qopt.PopulateOption != nil {
		opt := mopt.FindOne()
		opt.FindOneOptions = fopt
		opt.QueryOptions = qopt
		docs, err := findWithPopulate(ctx, model.collection, query, doc.doc, opt)
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

	err = runAfterFindHooks(ctx, doc.Doc, newHookArg[T](doc, FindQuery))
	if err != nil {
		return nil, err
	}

	return doc, nil
}

func (model *Model[T]) FindOne(
	ctx context.Context,
	query bson.M,
	opts ...*mopt.FindOneOptions,
) (*Document[T], error) {
	doc := &Document[T]{}

	qarg := NewQuery[T]().SetFilter(query).SetOperation(FindQuery).SetOptions(opts)

	err := runBeforeFindHooks(ctx, doc.Doc, newHookArg[T](qarg, FindQuery))
	if err != nil {
		return nil, err
	}

	qopt, fopt := mopt.MergeFindOneOptions(opts...)
	if qopt.PopulateOption != nil {
		opt := mopt.FindOne()
		opt.FindOneOptions = fopt
		opt.QueryOptions = qopt
		docs, err := findWithPopulate(ctx, model.collection, query, doc.doc, opt)
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

	err = runAfterFindHooks(ctx, doc.Doc, newHookArg[T](doc, FindQuery))
	if err != nil {
		return nil, err
	}

	return doc, nil
}

func (model *Model[T]) Find(
	ctx context.Context,
	query bson.M,
	opts ...*mopt.FindOptions,
) ([]*Document[T], error) {
	d := (&Document[T]{})

	qarg := NewQuery[T]().SetFilter(query).SetOperation(FindQuery).SetOptions(opts)

	err := runBeforeFindHooks(ctx, d.Doc, newHookArg[T](qarg, FindQuery))
	if err != nil {
		return nil, err
	}

	docs := make([]*Document[T], 0)

	qopt, fopt := mopt.MergeFindOptions(opts...)
	if qopt.PopulateOption != nil {
		opt := mopt.Find()
		opt.FindOptions = fopt
		opt.QueryOptions = qopt
		docs, err = findWithPopulate(ctx, model.collection, query, d.doc, opt)
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

	err = runAfterFindHooks(ctx, d, newHookArg[T](docs, FindQuery))
	if err != nil {
		return nil, err
	}

	return docs, nil
}

func (model *Model[T]) FindOneAndUpdate(
	ctx context.Context,
	query bson.M,
	update bson.M,
	opts ...*options.FindOneAndUpdateOptions,
) (*Document[T], error) {
	// doc := &Document[T]{}
	// err := model.collection.FindOneAndUpdate(ctx, query, update, opts...).Decode(doc)
	// if err != nil {
	// 	return nil, err
	// }
	// doc.collection = model.collection
	// return doc, nil
	return nil, nil
}

func (model *Model[T]) UpdateOne(ctx context.Context,
	query bson.M, update bson.M,
	opts ...*options.UpdateOptions,
) (*mongo.UpdateResult, error) {
	d := (&Document[T]{}).Doc

	qa := NewQuery[T]().SetFilter(query).
		SetUpdate(&update).
		SetOperation(UpdateQuery).
		SetOptions(opts)

	err := runBeforeUpdateHooks(ctx, d, newHookArg[T](qa, UpdateQuery))
	if err != nil {
		return nil, err
	}

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		if _, ok := update["$set"]; ok {
			update["$set"].(bson.M)["updatedAt"] = time.Now()
		} else {
			update["$set"] = bson.M{"updatedAt": time.Now()}
		}
		res, err := model.collection.UpdateOne(sessCtx, query, update, opts...)
		if err != nil {
			return nil, err
		}
		err = runAfterUpdateHooks(sessCtx, d, newHookArg[T](res, UpdateQuery))
		if err != nil {
			return nil, err
		}

		return res, nil
	}
	res, err := withTransaction(ctx, model.collection, callback)
	if err != nil {
		return nil, err
	}
	return res.(*mongo.UpdateResult), nil
}

func (model *Model[T]) UpdateMany(ctx context.Context,
	query bson.M, update bson.M, opts ...*options.UpdateOptions,
) (*mongo.UpdateResult, error) {
	d := (&Document[T]{}).doc

	qa := NewQuery[T]().SetFilter(query).
		SetUpdate(&update).
		SetOperation(UpdateQuery).
		SetOptions(opts)

	err := runBeforeUpdateHooks(ctx, d, newHookArg[T](qa, UpdateQuery))
	if err != nil {
		return nil, err
	}

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		if _, ok := update["$set"]; ok {
			update["$set"].(bson.M)["updatedAt"] = time.Now()
		} else {
			update["$set"] = bson.M{"updatedAt": time.Now()}
		}
		res, err := model.collection.UpdateMany(sessCtx, query, update, opts...)
		if err != nil {
			return nil, err
		}
		err = runAfterUpdateHooks(sessCtx, d, newHookArg[T](res, UpdateQuery))
		return res, err
	}

	res, err := withTransaction(ctx, model.collection, callback)
	if err != nil {
		return nil, err
	}
	return res.(*mongo.UpdateResult), nil
}

func (model *Model[T]) CountDocuments(ctx context.Context,
	query bson.M, opts ...*options.CountOptions,
) (int64, error) {
	return model.collection.CountDocuments(ctx, query, opts...)
}

func (model *Model[T]) Aggregate(
	ctx context.Context,
	pipeline mongo.Pipeline,
	res interface{},
) error {
	cursor, err := model.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return err
	}

	return cursor.All(ctx, res)
}

func (model *Model[T]) AggregateWithCursor(
	ctx context.Context,
	pipeline mongo.Pipeline,
) (*mongo.Cursor, error) {
	return model.collection.Aggregate(ctx, pipeline)
}

func findWithPopulate[P unionFindOpts, T Schema](
	ctx context.Context, c *mongo.Collection,
	q bson.M, d T, opt P,
) ([]*Document[T], error) {
	pipelineOpts, aggrOpts, queryOpts := mergeFindOptsWithAggregatOpts(opt)
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

	docs := make([]*Document[T], 0)
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
		return nil, fmt.Errorf("invalid ObjectID")
	}
	return &oid, nil
}

type unionFindOpts interface {
	*mopt.FindOptions | *mopt.FindOneOptions
}

// TODO: merge these options with aggregate options if possible
// if opt.AllowPartialResults != nil {
// }
// if opt.CursorType != nil {
// }
// if opt.Max != nil {
// }
// if opt.Min != nil {
// }
// if opt.NoCursorTimeout != nil {
// }
// if opt.OplogReplay != nil {
// }
// if opt.ReturnKey != nil {
// }
// if opt.ShowRecordID != nil {
// }
// if opt.Snapshot != nil {
// }

func mergeFindOptsWithAggregatOpts[T unionFindOpts](
	opt T,
) (mongo.Pipeline, *options.AggregateOptions, *mopt.QueryOptions) {
	aggOpts, pipelineOpts, queryOpts := options.Aggregate(), mongo.Pipeline{}, mopt.Query()
	switch opt := any(opt).(type) {
	case *mopt.FindOptions:
		if opt.AllowDiskUse != nil {
			aggOpts.SetAllowDiskUse(*opt.AllowDiskUse)
		}
		if opt.BatchSize != nil {
			aggOpts.SetBatchSize(*opt.BatchSize)
		}
		if opt.Collation != nil {
			aggOpts.SetCollation(opt.Collation)
		}
		if opt.Comment != nil {
			aggOpts.SetComment(*opt.Comment)
		}
		if opt.Hint != nil {
			aggOpts.SetHint(opt.Hint)
		}
		if opt.MaxAwaitTime != nil {
			aggOpts.SetMaxAwaitTime(*opt.MaxAwaitTime)
		}
		if opt.MaxTime != nil {
			aggOpts.SetMaxTime(*opt.MaxTime)
		}
		if opt.Let != nil {
			aggOpts.SetLet(opt.Let)
		}
		if opt.Sort != nil {
			pipelineOpts = append(pipelineOpts, bson.D{{Key: "$sort", Value: opt.Sort}})
		}
		if opt.Limit != nil {
			pipelineOpts = append(pipelineOpts, bson.D{{Key: "$limit", Value: *opt.Limit}})
		}
		if opt.Skip != nil {
			pipelineOpts = append(pipelineOpts, bson.D{{Key: "$skip", Value: *opt.Skip}})
		}
		if opt.Projection != nil {
			pipelineOpts = append(pipelineOpts, bson.D{{Key: "$project", Value: opt.Projection}})
		}
		queryOpts = opt.QueryOptions
	case *mopt.FindOneOptions:
		if opt.BatchSize != nil {
			aggOpts.SetBatchSize(*opt.BatchSize)
		}
		if opt.Collation != nil {
			aggOpts.SetCollation(opt.Collation)
		}
		if opt.Comment != nil {
			aggOpts.SetComment(*opt.Comment)
		}
		if opt.Hint != nil {
			aggOpts.SetHint(opt.Hint)
		}
		if opt.MaxAwaitTime != nil {
			aggOpts.SetMaxAwaitTime(*opt.MaxAwaitTime)
		}
		if opt.MaxTime != nil {
			aggOpts.SetMaxTime(*opt.MaxTime)
		}
		if opt.Sort != nil {
			pipelineOpts = append(pipelineOpts, bson.D{{Key: "$sort", Value: opt.Sort}})
		}
		if opt.Skip != nil {
			pipelineOpts = append(pipelineOpts, bson.D{{Key: "$skip", Value: *opt.Skip}})
		}
		if opt.Projection != nil {
			pipelineOpts = append(pipelineOpts, bson.D{{Key: "$project", Value: opt.Projection}})
		}
		queryOpts = opt.QueryOptions
	}
	return pipelineOpts, aggOpts, queryOpts
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
