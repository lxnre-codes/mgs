package mgs_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	client *mongo.Client
	dbName = "mgs_test"
)

func setup() func(context.Context) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "mongodb://localhost:27017"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var err error
	client, err = mongo.Connect(ctx, options.Client().ApplyURI(url))
	if err != nil {
		log.Fatal(err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	host := "localhost db"
	if !strings.Contains(url, "localhost") {
		host = "remote db"
	}
	fmt.Printf("db client connected to %s successfully \n", host)

	return func(ctx context.Context) {
		if err := client.Database(dbName).Drop(ctx); err != nil {
			panic(err)
		}
		if err := client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}
}

func getDb(ctx context.Context) (*mongo.Database, func(context.Context)) {
	db := client.Database(dbName)

	return db, func(ctx context.Context) {
		if err := db.Drop(ctx); err != nil {
			panic(err)
		}
	}
}

func TestMain(m *testing.M) {
	teardown := setup()
	defer teardown(context.Background())

	m.Run()
}
