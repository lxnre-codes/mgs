package mgs

import (
	"context"
	"fmt"
	"reflect"
	"strings"
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

	_, fopt := mopt.MergeFindOneOptions(opts...)

	err = model.collection.FindOne(ctx, bson.M{"_id": *oid}, fopt).Decode(doc)
	if err != nil {
		return nil, err
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

	_, fopt := mopt.MergeFindOneOptions(opts...)
	err = model.collection.FindOne(ctx, query, fopt).Decode(doc)
	if err != nil {
		return nil, err
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
	d := (&Document[T]{}).Doc

	qarg := NewQuery[T]().SetFilter(query).SetOperation(FindQuery).SetOptions(opts)

	err := runBeforeFindHooks(ctx, d, newHookArg[T](qarg, FindQuery))
	if err != nil {
		return nil, err
	}

	_, fopt := mopt.MergeFindOptions(opts...)
	cursor, err := model.collection.Find(ctx, query, fopt)
	if err != nil {
		return nil, err
	}
	docs := make([]*Document[T], 0)

	err = cursor.All(ctx, &docs)
	if err != nil {
		return nil, err
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

type populateChan struct {
	i   int
	val interface{}
}

// TODO: custom errors for populate path
func Populate(ctx context.Context, doc primitive.M, opt *mopt.PopulateOptions) error {
	paths := strings.Split(*opt.Path, ".")
	pathVal := doc[paths[0]]

	for len(paths) > 1 && pathVal != nil {
		// if current path is a slice, dive into the slice
		if v := reflect.ValueOf(pathVal); v.Kind() == reflect.Slice {
			// copy populate options and trim off first path key
			nopt := *opt
			np := strings.Join(paths[1:], ".")
			nopt.Path = &np

			// new context that we can cancel whenever error occurs while populating nested paths
			ctx2, cancel := context.WithCancel(ctx)
			defer cancel()

			errChan := make(chan error) // to keep track of errors
			doneChan := make(chan bool) // to keep track of populated count
			wg := 0                     // also to keep track of populated count

			for i := 0; i < v.Len(); i++ {
				// if value is bson.M (i.e map) we can index it's keys, traverse the map
				if iv, ok := v.Index(i).Interface().(bson.M); ok {
					wg++
					go func(pCtx context.Context, val bson.M) {
						for {
							select {
							case <-pCtx.Done():
								return
							default:
								// populate the keys
								err := Populate(pCtx, val, &nopt)
								if err != nil {
									errChan <- err
								}
								doneChan <- true
								return
							}
						}
					}(ctx2, iv)
				}
			}

			for wg > 0 {
				select {
				case <-ctx2.Done():
					return ctx2.Err()
				case err := <-errChan:
					return err
				case <-doneChan:
					wg--
				}
			}
			return nil
		} else if v, ok := pathVal.(bson.M); ok {
			// if path is a map, we can also it keys. Set the path value one step deep
			pathVal = v[paths[1]]
			// trim off the first path key
			paths = paths[1:]
		}
	}

	// we only get here if path is nested, but encountered a non map value while traversing. return
	if len(paths) > 1 {
		return nil
	}

	// we've arrived at the final key here. If it's a slice, dive into the slice and populate,
	if v := reflect.ValueOf(pathVal); v.Kind() == reflect.Slice {
		ctx2, cancel := context.WithCancel(ctx)
		defer cancel()
		errChan := make(chan error)
		popChan := make(chan populateChan)
		wg := 0

		for i := 0; i < v.Len(); i++ {
			wg++
			go func(fCtx context.Context, i int, val any) {
				for {
					select {
					case <-fCtx.Done():
						return
					default:
						res, err := findPopulation(fCtx, bson.M{*opt.ForeignField: val}, opt)
						if err != nil {
							errChan <- err
							return
						}
						popChan <- populateChan{i, res}
						return
					}
				}
			}(ctx2, i, v.Index(i).Interface())
		}

		newVals := reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(opt.Schema)), v.Len(), v.Cap())
		for wg > 0 {
			select {
			case <-ctx2.Done():
				return ctx2.Err()
			case err := <-errChan:
				return err
			case res := <-popChan:
				newVals.Index(res.i).Set(reflect.ValueOf(res.val))
				wg--
			}
		}
		pathVal = newVals.Interface()
	} else {
		// otherwise populate the value
		query := bson.M{*opt.ForeignField: pathVal}
		res, err := findPopulation(ctx, query, opt)
		if err != nil {
			return err
		}
		pathVal = res
	}

	// set the value of the populated path
	reflect.ValueOf(doc).SetMapIndex(reflect.ValueOf(paths[0]), reflect.ValueOf(pathVal))
	return nil
}

func findPopulation(ctx context.Context, query bson.M, popt *mopt.PopulateOptions) (any, error) {
	res := reflect.New(reflect.SliceOf(reflect.TypeOf(popt.Schema))).Interface()

	fopt := options.Find()
	if popt.Options != nil {
		_, fopt = mopt.MergeFindOptions(*popt.Options...)
	}
	if *popt.OnlyOne {
		fopt.SetLimit(1)
	}

	cursor, err := popt.Collection.Find(ctx, query, fopt)
	if err != nil {
		return nil, err
	}
	err = cursor.All(ctx, res)
	if err != nil {
		return nil, err
	}

	if *popt.OnlyOne {
		if reflect.ValueOf(res).Elem().Len() > 0 {
			res = reflect.ValueOf(res).Elem().Index(0).Interface()
		} else {
			res = nil
		}
	} else {
		res = reflect.ValueOf(res).Elem().Interface()
	}
	return res, nil
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
