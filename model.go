package mgs

import (
	"context"
	"fmt"
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
