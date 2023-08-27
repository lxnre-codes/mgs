package mgs_test

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"testing"
	"time"

	mgs "github.com/0x-buidl/go-mongoose"
	mopt "github.com/0x-buidl/go-mongoose/options"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson"
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

type Author struct {
	mock.Mock `bson:"-"`
	Name      string     `bson:"name"      json:"name" validate:"required"`
	Deleted   bool       `bson:"deleted"   json:"-"`
	DeletedAt *time.Time `bson:"deletedAt" json:"-"`
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

func TestModel_Populate(t *testing.T) {
	ctx := context.Background()
	db, teardown := setup()
	defer teardown(ctx)

	aColl := db.Collection("authors")
	fopt := mopt.Find()
	fopt.SetProjection(bson.M{"name": 1, "_id": 0})
	popOpts := mopt.PopulateOption{
		mopt.Populate().SetPath("chapters.author").SetCollection(aColl).SetOptions(fopt),
		mopt.Populate().SetPath("authors").SetCollection(aColl).SetOptions(fopt),
	}

	docs := generateBooks(ctx, db)

	var wg sync.WaitGroup
	start := time.Now()
	fmt.Println("populating docs", start, len(docs))
	for _, doc := range docs {
		for _, opt := range popOpts {
			wg.Add(1)
			go func(d bson.M, opt *mopt.PopulateOptions) {
				defer wg.Done()
				err := mgs.Populate(ctx, d, opt)
				assert.NoError(t, err)
			}(doc, opt)
		}
	}
	wg.Wait()

	fmt.Println("docs populated in ", time.Since(start))

	for _, doc := range docs {
		authors := doc["authors"].([]bson.M)
		for _, author := range authors {
			assert.NotEmpty(t, author["name"], "author should not be empty")
		}
		chapters := doc["chapters"].([]bson.M)
		for _, chapter := range chapters {
			author := chapter["author"].(bson.M)
			assert.NotEmpty(t, author["name"], "author should not be empty")
		}
	}
}

func generateBooks(ctx context.Context, db *mongo.Database) []bson.M {
	authorModel := mgs.NewModel[Author](db.Collection("authors"))
	newAuthors := make([]Author, 0)
	for i := 1; i <= 15; i++ {
		newAuthors = append(newAuthors, Author{Name: fmt.Sprintf("Author %d", i)})
	}
	authors, err := authorModel.CreateMany(ctx, newAuthors)
	if err != nil {
		panic(err)
	}

	docs := make([]bson.M, 0)
	for i := 1; i <= 100; i++ {
		chapters := make([]bson.M, 0)
		bookAuthors := make([]primitive.ObjectID, 0)
		for i := 1; i <= 10; i++ {
			author := authors[rand.Intn(len(authors))]
			include := true
			for _, a := range bookAuthors {
				if a == author.ID {
					include = false
				}
			}
			if include {
				bookAuthors = append(bookAuthors, author.ID)
			}
			chapter := bson.M{
				"_id":    primitive.NewObjectID(),
				"title":  fmt.Sprintf("Chapter %d", i),
				"pages":  i * 10,
				"author": author.ID,
			}
			chapters = append(chapters, chapter)
		}

		doc := bson.M{
			"_id":      primitive.NewObjectID(),
			"title":    fmt.Sprintf("Book %d", i),
			"authors":  bookAuthors,
			"price":    10.99,
			"chapters": chapters,
		}
		docs = append(docs, doc)
	}
	return docs
}
