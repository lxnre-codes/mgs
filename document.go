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

// Schema is an interface that represents the structure of a document in a MongoDB collection.
// It must be a struct.
type Schema interface{}

// Document is a struct that represents a document in a MongoDB collection.
// Do not use this struct directly, instead use the Model.NowDocument() method.
type Document[T Schema] struct {
	ID         primitive.ObjectID `json:"_id"       bson:"_id"`
	CreatedAt  *time.Time         `json:"createdAt" bson:"createdAt"`
	UpdatedAt  *time.Time         `json:"updatedAt" bson:"updatedAt"`
	Doc        *T                 `json:",inline"   bson:",inline"`
	doc        T
	collection *mongo.Collection
	isNew      bool
}

// Saves a document to the database atomically. This method creates a new document if the document is not already existing, Otherwise, it updates the existing document.
// The operation fails if any of the hooks return an error.
func (doc *Document[T]) Save(ctx context.Context) error {
	prevUpdatedAt := doc.UpdatedAt
	err := runBeforeSaveHooks(ctx, doc)
	if err != nil {
		return err
	}

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		if doc.isNew {
			_, err := doc.Collection().InsertOne(sessCtx, doc)
			if err != nil {
				return nil, err
			}
		} else {
			doc.generateUpdatedAt()
			_, err := doc.Collection().ReplaceOne(sessCtx, bson.M{"_id": doc.ID}, doc)
			if err != nil {
				doc.UpdatedAt = prevUpdatedAt
				return nil, err
			}
		}
		err = runAfterSaveHooks(sessCtx, doc)
		if err != nil {
			doc.UpdatedAt = prevUpdatedAt
		}
		return nil, err
	}

	_, err = withTransaction(ctx, doc.Collection(), callback)
	return err
}

// Deletes a document from the database atomically.
// The operation fails if any of the hooks return an error.
func (doc *Document[T]) Delete(ctx context.Context) error {
	err := runBeforeDeleteHooks(ctx, doc)
	if err != nil {
		return err
	}

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		err = doc.Model().DeleteOne(sessCtx, bson.M{"_id": doc.ID})
		if err != nil {
			return nil, err
		}

		err = runAfterDeleteHooks(sessCtx, doc)
		return nil, err
	}

	_, err = withTransaction(ctx, doc.Collection(), callback)
	if err != nil {
		return err
	}

	doc = nil
	return nil
}

// Returns the collection that the document belongs to.
func (doc *Document[T]) Collection() *mongo.Collection {
	return doc.collection
}

// Returns the model that the document belongs to.
func (doc *Document[T]) Model() *Model[T] {
	return NewModel[T](doc.collection)
}

// Returns the document as a JSON bytes.
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

func (doc *Document[T]) JSON() (bson.M, error) {
	bts, err := doc.MarshalJSON()
	if err != nil {
		return nil, err
	}

	var d bson.M
	json.Unmarshal(bts, &d)
	if err != nil {
		return nil, err
	}

	return d, nil
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
