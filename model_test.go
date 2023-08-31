package mgs_test

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"reflect"
	"testing"
	"time"

	mgs "github.com/0x-buidl/go-mongoose"
	mopt "github.com/0x-buidl/go-mongoose/options"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Book struct {
	Title string `bson:"title"     json:"title"    validate:"required"`
	// ObjectID or Author object
	Authors   []interface{} `bson:"authors"   json:"authors"  validate:"required,min=1"`
	Chapters  []Chapter     `bson:"chapters"  json:"chapters" validate:"required,min=1"`
	Price     float64       `bson:"price"     json:"price"`
	Deleted   bool          `bson:"deleted"   json:"-"`
	DeletedAt *time.Time    `bson:"deletedAt" json:"-"`
}

func (b *Book) Validate(ctx context.Context, arg *mgs.HookArg[Book]) error {
	var err error
	for _, author := range b.Authors {
		if _, ok := author.(primitive.ObjectID); !ok {
			err = fmt.Errorf("author must be ObjectID")
			break
		}
	}
	if err == nil {
		for _, chapter := range b.Chapters {
			if _, ok := chapter.Author.(primitive.ObjectID); !ok {
				err = fmt.Errorf("chapter.author must be ObjectID")
				break
			}
		}
	}
	return err
}

type Author struct {
	Name      string     `bson:"name"      json:"name" validate:"required"`
	Deleted   bool       `bson:"deleted"   json:"-"`
	DeletedAt *time.Time `bson:"deletedAt" json:"-"`
}

type Chapter struct {
	ID    primitive.ObjectID `bson:"_id"    json:"_id"    validate:"required"`
	Title string             `bson:"title"  json:"title"  validate:"required"`
	Pages int                `bson:"pages"  json:"pages"`
	// ObjectID or Author object
	Author interface{} `bson:"author" json:"author" validate:"required"`
}

func setup() (*mongo.Database, func(context.Context)) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "mongodb://localhost:27017"
	}

	// force mongoDB to use bson.M for embedded documents instead of bson.D
	reg := bson.NewRegistryBuilder().
		RegisterTypeMapEntry(bsontype.EmbeddedDocument, reflect.TypeOf(bson.M{})).
		Build()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := mongo.Connect(
		ctx,
		options.Client().ApplyURI(url).SetRegistry(reg),
	)
	if err != nil {
		log.Fatal(err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("db connected to ", url)

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
		Title:   "The Lord of the Rings",
		Authors: []any{primitive.NewObjectID()},
		Price:   10.99,
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

	bookModel := mgs.NewModel[Book](db.Collection("books"))

	fopt := options.Find()
	fopt.SetProjection(bson.D{{Key: "name", Value: 1}, {Key: "_id", Value: 0}})
	popOpts := mopt.PopulateOption{
		mopt.Populate().SetPath("chapters.author").SetCollection("authors").SetOptions(fopt),
		mopt.Populate().SetPath("authors").SetCollection("author").SetOptions(fopt),
	}

	opts := mopt.Find()
	opts.SetPopulate(popOpts...)

	generateBooks(ctx, db)

	start := time.Now()
	fmt.Println("populating docs with find", start)
	books, err := bookModel.Find(ctx, bson.M{}, opts)
	if err != nil {
		panic(err)
	}
	fmt.Println("docs populated in ", time.Since(start))

	for _, book := range books {
		for _, author := range book.Doc.Authors {
			assert.NotEmpty(t, author.(bson.M)["name"], "author should not be empty")
		}
		chapters := book.Doc.Chapters
		for _, chapter := range chapters {
			author := chapter.Author.(bson.M)
			assert.NotEmpty(t, author["name"], "author should not be empty")
		}
	}
}

func generateBooks(ctx context.Context, db *mongo.Database) []*mgs.Document[Book] {
	authorModel := mgs.NewModel[Author](db.Collection("authors"))
	bookModel := mgs.NewModel[Book](db.Collection("books"))
	newAuthors := make([]Author, 0)
	for i := 1; i <= 15; i++ {
		newAuthors = append(newAuthors, Author{Name: fmt.Sprintf("Author %d", i)})
	}
	authors, err := authorModel.CreateMany(ctx, newAuthors)
	if err != nil {
		panic(err)
	}

	docs := make([]Book, 0)
	for i := 1; i <= 100; i++ {
		chapters := make([]Chapter, 0)
		bookAuthors := make([]any, 0)
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
			chapter := Chapter{
				ID:     primitive.NewObjectID(),
				Title:  fmt.Sprintf("Chapter %d", i),
				Pages:  i * 10,
				Author: author.ID,
			}
			chapters = append(chapters, chapter)
		}

		doc := Book{
			Title:    fmt.Sprintf("Book %d", i),
			Authors:  bookAuthors,
			Price:    10.99,
			Chapters: chapters,
		}
		docs = append(docs, doc)
	}

	books, err := bookModel.CreateMany(ctx, docs)
	if err != nil {
		panic(err)
	}
	return books
}
