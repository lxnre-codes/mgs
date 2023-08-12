package mgs

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Document is a struct that represents a document in a MongoDB collection.
// Do not use this struct directly, instead use the Model.NowDocument() function.
type Document[T interface{}] struct {
	ID         primitive.ObjectID `json:"_id"       bson:"_id"`
	CreatedAt  *time.Time         `json:"createdAt" bson:"createdAt"`
	UpdatedAt  *time.Time         `json:"updatedAt" bson:"updatedAt"`
	Doc        *T                 `json:",inline"   bson:",inline"`
	doc        T
	collection *mongo.Collection
	isNew      bool
}

func (doc *Document[T]) Save(ctx context.Context) error {
	prevUpdatedAt, prevDoc := doc.UpdatedAt, doc.Doc
	err := runBeforeSaveHooks(ctx, doc)
	if err != nil {
		return err
	}
	if doc.isNew {
		_, err := doc.Collection().InsertOne(ctx, doc)
		if err != nil {
			return err
		}
	} else {

		doc.generateUpdatedAt()
		_, err := doc.Collection().ReplaceOne(ctx, bson.M{"_id": doc.ID}, doc)
		if err != nil {
			doc.UpdatedAt = prevUpdatedAt
			return err
		}

		doc, err = doc.Model().FindById(ctx, doc.ID.Hex())
		if err != nil {
			if err == mongo.ErrNoDocuments {
				doc = nil
			} else {
				doc.UpdatedAt = prevUpdatedAt
				doc.Doc = prevDoc
				// WARN: fix this. must be accounted for
				doc.Collection().ReplaceOne(ctx, bson.M{"_id": doc.ID}, doc)
				return err
			}
		}
	}
	err = runAfterSaveHooks(ctx, doc)
	if err != nil {
		doc.UpdatedAt = prevUpdatedAt
		doc.Doc = prevDoc
		// WARN: fix this. must be accounted for
		doc.Collection().ReplaceOne(ctx, bson.M{"_id": doc.ID}, doc)
		return err
	}
	return nil
}

func (doc *Document[T]) Delete(ctx context.Context) error {
	prev, err := doc.Model().FindById(ctx, doc.ID.Hex())
	if err != nil && err != mongo.ErrNoDocuments {
		return err
	}

	err = runBeforeDeleteHooks(ctx, doc)
	if err != nil {
		return err
	}

	err = doc.Model().DeleteOne(ctx, bson.M{"_id": doc.ID})
	if err != nil {
		return err
	}

	err = runAfterDeleteHooks(ctx, prev)
	if err != nil {
		doc.Collection().InsertOne(ctx, prev)
		return err
	}

	doc = nil
	return nil
}

func (doc *Document[T]) Collection() *mongo.Collection {
	return doc.collection
}

func (doc *Document[T]) Model() *Model[T] {
	return NewModel[T](doc.collection)
}

func (doc *Document[T]) MarshalJSON() ([]byte, error) {
	d := bson.M{}
	bts, err := json.Marshal(doc.Doc)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(bts, &d)
	if err != nil {
		return nil, err
	}

	if !doc.ID.IsZero() {
		d["_id"] = doc.ID
	}
	if doc.CreatedAt != nil {
		d["createdAt"] = doc.CreatedAt
	}
	if doc.UpdatedAt != nil {
		d["updatedAt"] = doc.UpdatedAt
	}

	return json.Marshal(d)
}

func (doc *Document[T]) Json() bson.M {
	bts, _ := doc.MarshalJSON()

	var d bson.M
	json.Unmarshal(bts, &d)

	return d
}

func (doc *Document[T]) IsNew() bool {
	return doc.isNew
}

func (doc *Document[T]) IsModified(field string) bool {
	return isModified(doc.doc, *doc.Doc, field)
}

func (doc *Document[T]) generateId() {
	doc.ID = primitive.NewObjectID()
}

func (doc *Document[T]) generateCreatedAt() {
	createdAt := time.Now()
	doc.CreatedAt = &createdAt
}

func (doc *Document[T]) generateUpdatedAt() {
	updatedAt := time.Now()
	doc.UpdatedAt = &updatedAt
}

func isModified(old, newV interface{}, field string) bool {
	o := reflect.ValueOf(old)
	n := reflect.ValueOf(newV)

	parts := strings.Split(field, ".")

	if len(parts) == 1 {
		of := o.FieldByName(field)
		nf := n.FieldByName(field)
		return of.IsValid() && nf.IsValid() &&
			!reflect.DeepEqual(of.Interface(), nf.Interface())
	} else {
		of := o.FieldByName(parts[0])
		nf := n.FieldByName(parts[0])

		if !of.IsValid() || !nf.IsValid() {
			return false
		}

		return isModified(of.Interface(), nf.Interface(), strings.Join(parts[1:], "."))
	}
}
