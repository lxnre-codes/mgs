package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/0x-buidl/mgs/examples/database"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	db        *mongo.Database
	bookModel *database.BookModel
)

func setup() func(context.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	db = client.Database("mgs_test")
	bookModel = database.NewBookModel(db.Collection("books"))

	fmt.Println("database setup done")

	return func(ctx context.Context) {
		if err := db.Drop(ctx); err != nil {
			panic(err)
		}
		if err := client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}
}

func main() {
	ctx := context.Background()
	defer setup()(ctx)

	books, err := bookModel.CreateMany(
		ctx,
		[]database.Book{
			{Author: primitive.NewObjectID()},
			{Author: primitive.NewObjectID(), Deleted: true},
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	book, err := bookModel.FindOne(ctx, bson.M{"_id": books[0].GetID()})
	if err != nil {
		log.Fatal(err)
	}
	if book == nil {
		log.Fatal("book should not be nil")
	}

	book, err = bookModel.FindOne(ctx, bson.M{"_id": books[1].GetID()})
	if err != mongo.ErrNoDocuments {
		log.Fatal("should be mongo.ErrNoDocuments")
	}
	if book != nil {
		log.Fatal("book should be nil")
	}

	book, err = bookModel.FindById(ctx, books[1].GetID())
	if err != mongo.ErrNoDocuments {
		log.Fatal("should be mongo.ErrNoDocuments")
	}
	if book != nil {
		log.Fatal("book should be nil")
	}
}
