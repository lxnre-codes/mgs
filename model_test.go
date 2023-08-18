package mgs_test

import (
	"context"
	"log"
	"testing"
	"time"

	mgs "github.com/0x-buidl/go-mongoose"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Book struct {
	mock.Mock `bson:"-"`
	Title     string             `bson:"title"     json:"title"  validate:"required"`
	Author    primitive.ObjectID `bson:"author"    json:"author" validate:"required"`
	Price     float64            `bson:"price"     json:"price"`
	Deleted   bool               `bson:"deleted"   json:"-"`
	DeletedAt *time.Time         `bson:"deletedAt" json:"-"`
}

func setup() (*mongo.Database, func(context.Context)) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}
	db := client.Database("mgs_test")

	return db, func(ctx context.Context) {
		if err := db.Drop(ctx); err != nil {
			panic(err)
		}
	}
}

func TestModel_NewDocument(t *testing.T) {
	ctx := context.Background()

	db, teardown := setup()
	defer teardown(ctx)

	bookModel := mgs.NewModel[Book](db.Collection("books"))

	nb := Book{
		Title:  "The Lord of the Rings",
		Author: primitive.NewObjectID(),
		Price:  10.99,
	}
	doc := bookModel.NewDocument(nb)
	assert.NotZero(t, doc.ID, "doc.ID should not be zero")
	assert.NotZero(t, doc.CreatedAt, "doc.CreatedAt should not be zero")
	assert.NotZero(t, doc.UpdatedAt, "doc.UpdatedAt should not be zero")
	assert.Equal(t, nb, *doc.Doc, "doc should be equal to nb")
}
