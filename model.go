package mgs

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Model[T interface{}] struct {
	collection *mongo.Collection
}

func NewModel[T interface{}](collection *mongo.Collection) *Model[T] {
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

	_, err = model.collection.InsertOne(ctx, newDoc, opts...)
	if err != nil {
		return nil, err
	}

	err = runAfterCreateHooks(ctx, newDoc)
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
	newDocs := make([]*Document[T], len(docs))
	docsToInsert := make([]interface{}, len(docs))
	for i, doc := range docs {
		newDoc := model.NewDocument(doc)
		err := runBeforeCreateHooks(ctx, newDoc)
		if err != nil {
			return nil, fmt.Errorf("error running before create hooks at index %d : %w", i, err)
		}
		newDocs[i] = newDoc
		docsToInsert[i] = newDoc
	}
	_, err := model.collection.InsertMany(ctx, docsToInsert, opts...)
	if err != nil {
		return nil, err
	}
	for i, doc := range newDocs {
		doc.isNew = false
		doc.collection = model.collection
		err := runAfterCreateHooks(ctx, doc)
		if err != nil {
			return nil, fmt.Errorf("error running after create hooks at index %d : %w", i, err)
		}
	}
	return newDocs, nil
}

func (model *Model[T]) DeleteOne(
	ctx context.Context,
	query bson.M,
	opts ...*options.DeleteOptions,
) error {
	// TODO: run before delete hooks, also check the Document.Delete method
	_, err := model.collection.DeleteOne(ctx, query, opts...)
	// TODO: run after delete hooks, also check the Document.Delete method
	return err
}

func (model *Model[T]) DeleteMany(
	ctx context.Context,
	query bson.M,
	opts ...*options.DeleteOptions,
) error {
	// TODO: run before delete hooks
	_, err := model.collection.DeleteMany(ctx, query, opts...)
	// TODO: run after delete hooks
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

	// TODO: run after find hooks
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
	// TODO: run after find hooks

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

	for cursor.Next(ctx) {
		doc := &Document[T]{}
		err := cursor.Decode(doc)
		if err != nil {
			return nil, err
		}
		doc.collection = model.collection
		docs = append(docs, doc)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	// TODO: run after find hooks

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

// NOTE: This does not validate the document. Use carefully
func (model *Model[T]) UpdateOne(ctx context.Context,
	query bson.M, update bson.M,
	opts ...*options.UpdateOptions,
) error {
	// TODO: run before update hooks
	if _, ok := update["$set"]; ok {
		update["$set"].(bson.M)["updatedAt"] = time.Now()
	} else {
		update["$set"] = bson.M{"updatedAt": time.Now()}
	}
	_, err := model.collection.UpdateOne(ctx, query, update, opts...)
	// TODO: run after update hooks
	return err
}

// NOTE: This does not validate the document. Use carefully
func (model *Model[T]) UpdateMany(ctx context.Context,
	query bson.M, update bson.M, opts ...*options.UpdateOptions,
) error {
	// TODO: run before update hooks
	if _, ok := update["$set"]; ok {
		update["$set"].(bson.M)["updatedAt"] = time.Now()
	} else {
		update["$set"] = bson.M{"updatedAt": time.Now()}
	}
	_, err := model.collection.UpdateMany(ctx, query, update, opts...)
	// TODO: run after update hooks
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

func GetProjections(keys string) bson.D {
	sf := strings.Split(keys, " ")
	projections := bson.D{}
	for _, v := range sf {
		ex := strings.HasPrefix(v, "-")
		if ex {
			projections = append(projections, bson.E{Key: v[1:], Value: 0})
		} else {
			projections = append(projections, bson.E{Key: v, Value: 1})
		}
	}

	return bson.D{{Key: "$project", Value: projections}}
}

func GetUnwind(path string, preserveNull bool) bson.D {
	return bson.D{
		{
			Key: "$unwind",
			Value: bson.D{
				{Key: "path", Value: "$" + path},
				{Key: "preserveNullAndEmptyArrays", Value: preserveNull},
			},
		},
	}
}
