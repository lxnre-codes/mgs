package mgs

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"github.com/0x-buidl/mgs/internal"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Schema is an interface that represents the structure of a document in a MongoDB collection.
// It must be a struct.
type Schema interface{}

// IDefaultSchema is an interface that represents the default fields that are added to a document.
// Only structs that implement this interface can be used as a default schema.
type IDefaultSchema interface {
	// Generates a new ObjectID and sets it to the ID field.
	GenerateID()
	// Generates a new time.Time and sets it to the CreatedAt field.
	GenerateCreatedAt()
	// Generates a new time.Time and sets it to the UpdatedAt field.
	GenerateUpdatedAt()
	// GetID returns the ID field.
	GetID() primitive.ObjectID
	// GetCreatedAt returns the CreatedAt field.
	GetCreatedAt() time.Time
	// GetUpdatedAt returns the UpdatedAt field.
	GetUpdatedAt() time.Time
	// Sets the ID field to id.
	// SetID(id primitive.ObjectID)

	// Sets the CreatedAt field to t.
	// SetCreatedAt(t time.Time)

	// Sets the UpdatedAt field to t.
	SetUpdatedAt(t time.Time)
	// Returns the tag name for the UpdatedAt field. t can be either "json", "bson" or any custom tag.
	// This is useful for setting the UpdatedAt field when updating with [Model.UpdateOne] and [Model.UpdateMany].
	GetUpdatedAtTag(t string) string
}

// DefaultSchema is a struct that implements the IDefaultSchema interface.
// It contains the default fields (ID,CreatedAt,UpdatedAt) that are added to a document.
type DefaultSchema struct {
	ID        primitive.ObjectID `json:"_id,omitempty"       bson:"_id,omitempty"`
	CreatedAt time.Time          `json:"createdAt,omitempty" bson:"createdAt,omitempty"`
	UpdatedAt time.Time          `json:"updatedAt,omitempty" bson:"updatedAt,omitempty"`
}

func (s *DefaultSchema) GenerateID() {
	s.ID = primitive.NewObjectID()
}

func (s *DefaultSchema) GenerateCreatedAt() {
	s.CreatedAt = time.Now()
}

func (s *DefaultSchema) GenerateUpdatedAt() {
	s.UpdatedAt = time.Now()
}

func (s DefaultSchema) GetID() primitive.ObjectID {
	return s.ID
}

func (s DefaultSchema) GetCreatedAt() time.Time {
	return s.CreatedAt
}

func (s DefaultSchema) GetUpdatedAt() time.Time {
	return s.UpdatedAt
}

func (s DefaultSchema) GetUpdatedAtTag(t string) string {
	return "updatedAt"
}

// func (s *DefaultSchema) SetID(id primitive.ObjectID) {
// 	s.ID = id
// }

// func (s *DefaultSchema) SetCreatedAt(t time.Time) {
// 	s.CreatedAt = t
// }

func (s *DefaultSchema) SetUpdatedAt(t time.Time) {
	s.UpdatedAt = t
}

// Document is a struct that represents a document in a MongoDB collection.
// Do not use this struct directly, instead use the [Model.NewDocument] method.
type Document[T Schema, P IDefaultSchema] struct {
	IDefaultSchema `json:"-" bson:"-"`
	Doc            *T `json:",inline" bson:",inline"`
	doc            T
	collection     *mongo.Collection
	isNew          bool
}

// Saves [Documet] (doc) to the database atomically. This method creates a new document if the document is not already existing, Otherwise, it updates the existing document.
// This operation fails if any of the hooks return an error.
func (doc *Document[T, P]) Save(ctx context.Context) error {
	prevUpdatedAt := doc.GetUpdatedAt()

	query := UpdateOne
	if doc.isNew {
		query = CreateOne
	}

	arg := newHookArg[T](doc, query)
	if err := runValidateHooks(ctx, doc, arg); err != nil {
		return err
	}
	if err := runBeforeSaveHooks(ctx, doc, arg); err != nil {
		return err
	}

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		if doc.IsNew() {
			_, err := doc.Collection().InsertOne(sessCtx, doc)
			if err != nil {
				return nil, err
			}
		} else {
			doc.GenerateUpdatedAt()
			_, err := doc.Collection().ReplaceOne(sessCtx, bson.M{"_id": doc.GetID()}, doc)
			if err != nil {
				doc.SetUpdatedAt(prevUpdatedAt)
				return nil, err
			}
		}
		arg := newHookArg[T](doc, query)
		err := runAfterSaveHooks(sessCtx, doc, arg)
		if err != nil {
			doc.SetUpdatedAt(prevUpdatedAt)
		}
		doc.isNew = false
		return nil, err
	}

	_, err := withTransaction(ctx, doc.Collection(), callback)
	return err
}

// Deletes [Document] (doc) from the database atomically.
// This operation fails if any of the hooks return an error.
func (doc *Document[T, P]) Delete(ctx context.Context) error {
	arg := newHookArg[T](doc, DeleteOne)
	err := runBeforeDeleteHooks(ctx, doc, arg)
	if err != nil {
		return err
	}

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		_, err = doc.Collection().DeleteOne(sessCtx, bson.M{"_id": doc.GetID()})
		if err != nil {
			return nil, err
		}

		err = runAfterDeleteHooks(sessCtx, doc, arg)
		return nil, err
	}

	_, err = withTransaction(ctx, doc.Collection(), callback)
	if err != nil {
		return err
	}

	doc = nil
	return nil
}

// func (doc *Document[T, P]) Update(ctx context.Context) error {
// 	return nil
// }

// Returns the collection that doc belongs to.
func (doc *Document[T, P]) Collection() *mongo.Collection {
	return doc.collection
}

// Returns the model that the doc belongs to.
func (doc *Document[T, P]) Model() *Model[T, P] {
	return NewModel[T, P](doc.collection)
}

// Returns doc as JSON bytes.
func (doc *Document[T, P]) MarshalJSON() ([]byte, error) {
	d := make(map[string]any)
	if err := internal.DecodeJSON(doc.Doc, &d); err != nil {
		return nil, err
	}

	defDoc := make(map[string]any)
	if err := internal.DecodeJSON(doc.IDefaultSchema, &defDoc); err != nil {
		return nil, err
	}

	for k, v := range defDoc {
		d[k] = v
	}

	return json.Marshal(d)
}

// Returns the JSON representation of doc.
func (doc *Document[T, P]) JSON() (map[string]any, error) {
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

// Returns doc as BSON bytes.
func (doc *Document[T, P]) MarshalBSON() ([]byte, error) {
	var d bson.M
	if err := internal.DecodeBSON(doc.Doc, &d); err != nil {
		return nil, err
	}

	defDoc := bson.M{}
	if err := internal.DecodeBSON(doc.IDefaultSchema, &defDoc); err != nil {
		return nil, err
	}

	for k, v := range defDoc {
		d[k] = v
	}

	return bson.Marshal(d)
}

// Unmarshals data into the doc.
func (doc *Document[T, P]) UnmarshalBSON(data []byte) error {
	dec, err := bson.NewDecoder(bsonrw.NewBSONValueReader(bsontype.EmbeddedDocument, data))
	reg := bson.NewRegistryBuilder().
		RegisterTypeMapEntry(bsontype.EmbeddedDocument, reflect.TypeOf(bson.M{})).
		Build()
	dec.SetRegistry(reg)
	if err != nil {
		return err
	}

	var nd T
	err = dec.Decode(&nd)
	if err != nil {
		return err
	}

	defType := reflect.ValueOf(*new(P)).Type().Elem()
	defSchema := reflect.New(defType).Interface()
	err = bson.Unmarshal(data, defSchema)
	if err != nil {
		return err
	}
	doc.Doc = &nd
	doc.IDefaultSchema = defSchema.(P)
	return nil
}

// Returns BSON representation of doc.
func (doc *Document[T, P]) BSON() (bson.M, error) {
	bts, err := doc.MarshalBSON()
	if err != nil {
		return nil, err
	}

	var d bson.M
	bson.Unmarshal(bts, &d)
	if err != nil {
		return nil, err
	}

	return d, nil
}

// IsNew returns true if the document is new and has never been written to the database.
// Otherwise, it returns false
func (doc *Document[T, P]) IsNew() bool {
	return doc.isNew
}

// IsModified returns true if the field has been modified.
func (doc *Document[T, P]) IsModified(field string) bool {
	return internal.IsModified(doc.doc, *doc.Doc, field)
}
