package mgs

import (
	"go.mongodb.org/mongo-driver/bson"
)

// Query is a struct that holds information about the current operation beign executed on a model.
type Query[T Schema] struct {
	// The document filter for this operation
	Filter *bson.M
	// Update payload if Operation is an update operation
	Update *bson.M
	// Options specific to the current operation
	Options interface{}
	// Operation being executed
	Operation QueryOperation
}

// Model Query Operation
type QueryOperation string

const (
	CreateOne        QueryOperation = "create-one"
	CreateMany       QueryOperation = "create-many"
	FindMany         QueryOperation = "find-many"
	FindOne          QueryOperation = "find-one"
	FindOneAndUpdate QueryOperation = "find-one-and-update"
	Replace          QueryOperation = "replace"
	UpdateOne        QueryOperation = "update-one"
	UpdateMany       QueryOperation = "update-many"
	DeleteOne        QueryOperation = "delete-one"
	DeleteMany       QueryOperation = "delete-many"
)

// NewQuery returns and empty [Query] struct.
func NewQuery[T Schema]() *Query[T] {
	return &Query[T]{}
}

// SetFilter sets the Query filter field.
func (q *Query[T]) SetFilter(f *bson.M) *Query[T] {
	q.Filter = f
	return q
}

// SetUpdate sets the Query Update field.
func (q *Query[T]) SetUpdate(u *bson.M) *Query[T] {
	q.Update = u
	return q
}

// SetOptions sets the Query Options field.
func (q *Query[T]) SetOptions(o interface{}) *Query[T] {
	q.Options = o
	return q
}

// SetOperation sets the Query Operation field.
func (q *Query[T]) SetOperation(o QueryOperation) *Query[T] {
	q.Operation = o
	return q
}
