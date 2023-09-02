package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/0x-buidl/mgs/examples/database"
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
}
