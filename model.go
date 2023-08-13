package mgs

import (
	"context"
	"fmt"
	"time"

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
	opts ...*options.InsertOneOptions,
) (*Document[T], error) {
	newDoc := model.NewDocument(doc)

	err := runBeforeCreateHooks(ctx, newDoc)
	if err != nil {
		return nil, err
	}

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		_, err = model.collection.InsertOne(sessCtx, newDoc, opts...)
		if err != nil {
			return nil, err
		}

		err = runAfterCreateHooks(sessCtx, newDoc)
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
	opts ...*options.InsertManyOptions,
) ([]*Document[T], error) {
	newDocs, docsToInsert, err := model.beforeCreateMany(ctx, docs)
	if err != nil {
		return nil, err
	}

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		_, err := model.collection.InsertMany(sessCtx, docsToInsert, opts...)
		if err != nil {
			return nil, err
		}

		err = model.bulkHookRun(sessCtx, newDocs, runAfterCreateHooks[T])
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
) error {
	doc := &Document[T]{}
	err := model.collection.FindOne(ctx, query).Decode(doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil
		}
		return err
	}
	err = runBeforeDeleteHooks(ctx, doc)
	if err != nil {
		return err
	}
	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		_, err := model.collection.DeleteOne(ctx, query, opts...)
		if err != nil {
			return nil, err
		}
		err = runAfterDeleteHooks(sessCtx, doc)
		return nil, err
	}
	_, err = withTransaction(ctx, model.collection, callback)
	return err
}

func (model *Model[T]) DeleteMany(
	ctx context.Context,
	query bson.M,
	opts ...*options.DeleteOptions,
) error {
	docs := make([]*Document[T], 0)
	cursor, err := model.collection.Find(ctx, query)
	if err != nil {
		return err
	}
	err = cursor.All(ctx, &docs)
	if err != nil {
		return err
	}
	err = model.bulkHookRun(ctx, docs, runBeforeDeleteHooks[T])
	if err != nil {
		return err
	}

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		_, err := model.collection.DeleteMany(ctx, query, opts...)
		if err != nil {
			return nil, err
		}

		err = model.bulkHookRun(sessCtx, docs, runAfterDeleteHooks[T])
		return nil, err
	}
	_, err = withTransaction(ctx, model.collection, callback)
	return err
}

func (model *Model[T]) FindById(
	ctx context.Context, id any,
	opts ...*options.FindOneOptions,
) (*Document[T], error) {
	oid, err := getObjectId(id)
	if err != nil {
		return nil, err
	}

	doc := &Document[T]{}

	query := bson.M{"_id": *oid}
	err = runBeforeFindHooks(ctx, doc, query)
	if err != nil {
		return nil, err
	}

	err = model.collection.FindOne(ctx, bson.M{"_id": *oid}, opts...).Decode(doc)
	if err != nil {
		return nil, err
	}
	doc.collection = model.Collection()

	err = runAfterFindHooks(ctx, doc)
	if err != nil {
		return nil, err
	}

	return doc, nil
}

func (model *Model[T]) FindOne(
	ctx context.Context,
	query bson.M,
	opts ...*options.FindOneOptions,
) (*Document[T], error) {
	doc := &Document[T]{}
	err := runBeforeFindHooks(ctx, doc, query)
	if err != nil {
		return nil, err
	}
	err = model.collection.FindOne(ctx, query, opts...).Decode(doc)
	if err != nil {
		return nil, err
	}
	doc.collection = model.collection

	err = runAfterFindHooks(ctx, doc)
	if err != nil {
		return nil, err
	}

	return doc, nil
}

func (model *Model[T]) Find(
	ctx context.Context,
	query bson.M,
	opts ...*options.FindOptions,
) ([]*Document[T], error) {
	err := runBeforeFindHooks(ctx, &Document[T]{}, query)
	if err != nil {
		return nil, err
	}

	cursor, err := model.collection.Find(ctx, query, opts...)
	if err != nil {
		return nil, err
	}
	docs := make([]*Document[T], 0)

	err = cursor.All(ctx, &docs)
	if err != nil {
		return nil, err
	}

	err = model.bulkHookRun(ctx, docs, runAfterFindHooks[T])
	if err != nil {
		return nil, err
	}

	return docs, nil
}

// func (model *Model[T]) FindOneAndUpdate(
// 	ctx context.Context,
// 	query bson.M,
// 	update bson.M,
// 	opts ...*options.FindOneAndUpdateOptions,
// ) (*Document[T], error) {
// 	doc := &Document[T]{}
// 	err := model.collection.FindOneAndUpdate(ctx, query, update, opts...).Decode(doc)
// 	if err != nil {
// 		return nil, err
// 	}
// 	doc.collection = model.collection
// 	return doc, nil
// }

func (model *Model[T]) UpdateOne(ctx context.Context,
	query bson.M, update bson.M,
	opts ...*options.UpdateOptions,
) error {
	docs := make([]*Document[T], 0)
	cursor, err := model.collection.Find(ctx, query)
	if err != nil {
		return err
	}
	err = cursor.All(ctx, &docs)
	if err != nil {
		return err
	}

	err = model.bulkHookRun(ctx, docs, runBeforeUpdateHooks[T])
	if err != nil {
		return err
	}

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		if _, ok := update["$set"]; ok {
			update["$set"].(bson.M)["updatedAt"] = time.Now()
		} else {
			update["$set"] = bson.M{"updatedAt": time.Now()}
		}
		_, err := model.collection.UpdateOne(sessCtx, query, update, opts...)
		if err != nil {
			return nil, err
		}
		err = model.bulkHookRun(sessCtx, docs, runAfterUpdateHooks[T])
		return nil, err
	}
	_, err = withTransaction(ctx, model.collection, callback)
	return err
}

func (model *Model[T]) UpdateMany(ctx context.Context,
	query bson.M, update bson.M, opts ...*options.UpdateOptions,
) error {
	doc := &Document[T]{}
	err := model.collection.FindOne(ctx, query).Decode(doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil
		}
		return err
	}
	err = runBeforeUpdateHooks(ctx, doc)
	if err != nil {
		return err
	}

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		if _, ok := update["$set"]; ok {
			update["$set"].(bson.M)["updatedAt"] = time.Now()
		} else {
			update["$set"] = bson.M{"updatedAt": time.Now()}
		}
		_, err := model.collection.UpdateMany(sessCtx, query, update, opts...)
		if err != nil {
			return nil, err
		}
		err = runAfterUpdateHooks(sessCtx, doc)
		return nil, err
	}

	_, err = withTransaction(ctx, model.collection, callback)
	return err
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
		return nil, fmt.Errorf("invalid id type")
	}
	return &oid, nil
}
