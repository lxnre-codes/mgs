package mgs_test

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/0x-buidl/mgs"
	mopt "github.com/0x-buidl/mgs/options"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DefaultSchema bson.M

func (s *DefaultSchema) GenerateID() {
}

func (s *DefaultSchema) GenerateCreatedAt() {
}

func (s *DefaultSchema) GenerateUpdatedAt() {
}

func (s DefaultSchema) GetID() primitive.ObjectID {
	return primitive.ObjectID{}
}

func (s DefaultSchema) GetCreatedAt() time.Time {
	return time.Time{}
}

func (s DefaultSchema) GetUpdatedAt() time.Time {
	return time.Time{}
}

func (s DefaultSchema) GetUpdatedAtTag(t string) string {
	return ""
}

func (s *DefaultSchema) SetID(id primitive.ObjectID) {
}

func (s *DefaultSchema) SetCreatedAt(t time.Time) {
}

func (s *DefaultSchema) SetUpdatedAt(t time.Time) {
}

type Book struct {
	Title string `bson:"title"     json:"title"    validate:"required"`
	// ObjectID or Author object
	Authors   []interface{} `bson:"authors"   json:"authors"  validate:"required,min=1"`
	Chapters  []Chapter     `bson:"chapters"  json:"chapters" validate:"required,min=1"`
	Price     Price         `bson:"price"     json:"price"`
	Deleted   bool          `bson:"deleted"   json:"-"`
	DeletedAt *time.Time    `bson:"deletedAt" json:"-"`
}

type Price struct {
	Amount   int64  `bson:"amount"   json:"amount"`
	Currency string `bson:"currency" json:"currency"`
	Decimals int64  `bson:"decimals" json:"decimals"`
}

type BookDoc = mgs.Document[Book, *mgs.DefaultSchema]

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

func TestNewModel(t *testing.T) {
	t.Run("Should panic if schema is is not a struct", func(t *testing.T) {
		assert.Panics(t, func() {
			mgs.NewModel[bson.M, *mgs.DefaultSchema](nil)
		}, "NewModel should panic if schema is not a struct")

		assert.Panics(t, func() {
			mgs.NewModel[Book, *DefaultSchema](nil)
		}, "NewModel should panic if defaultSchema is not a struct")
	})
}

func TestModel_NewDocument(t *testing.T) {
	ctx := context.Background()

	db, cleanup := getDb(ctx)
	defer cleanup(ctx)

	bookModel := mgs.NewModel[Book, *mgs.DefaultSchema](db.Collection("books"))

	nb := Book{
		Title:   "The Lord of the Rings",
		Authors: []any{primitive.NewObjectID()},
		Price:   Price{Amount: 100, Currency: "USD", Decimals: 99},
	}
	doc := bookModel.NewDocument(nb)
	assert.NotZero(t, doc.GetID(), "doc.ID should not be zero")
	assert.NotZero(t, doc.GetCreatedAt(), "doc.CreatedAt should not be zero")
	assert.NotZero(t, doc.GetUpdatedAt(), "doc.UpdatedAt should not be zero")
	assert.Equal(t, bookModel, doc.Model(), "doc.Model should be equal to bookModel")
	assert.Equal(t, nb, *doc.Doc, "doc should be equal to nb")
	assert.True(t, doc.IsNew(), "doc should be new")
	assert.False(t, doc.IsModified("Title"), "doc.Title should not be modified")
	assert.False(t, doc.IsModified("Price.Currency"), "doc.Price.Currency should not be modified")
	doc.Doc.Title = "The Lord of the Rings 2"
	doc.Doc.Price.Currency = "EUR"
	assert.True(t, doc.IsModified("Title"), "doc.Title should be modified")
	assert.True(t, doc.IsModified("Price.Currency"), "doc.Price.Currency should be modified")

	json, err := doc.JSON()
	assert.NoError(t, err, "doc.JSON() should not return error")

	// default fields
	assert.Contains(t, json, "_id", "json should have _id field")
	assert.Contains(t, json, "createdAt", "json should have createdAt field")
	assert.Contains(t, json, "updatedAt", "json should have updatedAt field")
	// custom bson fields for internal use
	assert.NotContains(t, json, "deleted", "json should not have deleted field")
	assert.NotContains(t, json, "deletedAt", "json should not have deletedAt field")

	bson, err := doc.BSON()
	assert.NoError(t, err, "doc.BSON() should not return error")

	// default fields
	assert.Contains(t, bson, "_id", "bson should have _id field")
	assert.Contains(t, bson, "createdAt", "bson should have createdAt field")
	assert.Contains(t, bson, "updatedAt", "bson should have updatedAt field")
	// custom bson fields for internal use
	assert.Contains(t, bson, "deleted", "bson should have deleted field")
	assert.Contains(t, bson, "deletedAt", "bson should have deletedAt field")
}

func TestModel_Find(t *testing.T) {
	ctx := context.Background()
	db, cleanup := getDb(ctx)
	defer cleanup(ctx)

	bookModel := mgs.NewModel[Book, *mgs.DefaultSchema](db.Collection("books"))
	genBooks := generateBooks(ctx, db)

	t.Run("Should return error when FindById receives invalid id", func(t *testing.T) {
		_, err := bookModel.FindById(ctx, "invalid")
		assert.Equal(
			t, err, primitive.ErrInvalidHex,
			"FindByID should return error if id is not valid",
		)

		_, err = bookModel.FindById(ctx, 12345)
		assert.Equal(
			t, err, primitive.ErrInvalidHex,
			"FindByID should return error if id is not valid",
		)
	})

	t.Run("Should pass if FindById receives valid id", func(t *testing.T) {
		_, err := bookModel.FindById(ctx, genBooks[0].GetID().Hex())
		assert.NoError(t, err, "FindByID should pass if id is valid ObjectID hex")

		_, err = bookModel.FindById(ctx, genBooks[0].GetID())
		assert.NoError(t, err, "FindByID should pass if id is valid ObjectID")
	})
}

func TestModel_Populate(t *testing.T) {
	ctx := context.Background()
	db, cleanup := getDb(ctx)
	defer cleanup(ctx)

	bookModel := mgs.NewModel[Book, *mgs.DefaultSchema](db.Collection("books"))
	genBooks := generateBooks(ctx, db)

	fopt := options.Find()
	fopt.SetProjection(bson.D{{Key: "name", Value: 1}, {Key: "_id", Value: 0}})
	popOpts := mopt.PopulateOption{
		mopt.Populate().SetPath("chapters.author").SetCollection("authors").SetOptions(fopt),
		mopt.Populate().SetPath("authors").SetCollection("authors").SetOptions(fopt),
	}

	t.Run("Should populate docs with FindOne", func(t *testing.T) {
		opts := mopt.FindOne()
		opts.SetPopulate(popOpts...)

		// start := time.Now()
		// fmt.Println("---------- finding & populating single doc with FindOne ----------")
		book, err := bookModel.FindOne(ctx, bson.M{}, opts)

		// fmt.Printf("---------- FindOne executed in %v ---------- \n", time.Since(start))

		assert.NoError(t, err, "Find should not return error")
		assert.NotNil(t, book, "book should not be nil")

		for _, author := range book.Doc.Authors {
			assert.NotEmpty(t, author.(bson.M)["name"], "author should not be empty")
		}

		chapters := book.Doc.Chapters
		for _, chapter := range chapters {
			author := chapter.Author.(bson.M)
			assert.NotEmpty(t, author["name"], "author should not be empty")
		}

		// fmt.Printf(
		// 	"---------- FindOne populated %d paths  ---------- \n",
		// 	len(chapters)+len(book.Doc.Authors),
		// )
	})

	t.Run("Should populate docs with FindById", func(t *testing.T) {
		opts := mopt.FindOne()
		opts.SetPopulate(popOpts...)

		// start := time.Now()
		// fmt.Println("---------- finding & populating single doc with FindById ----------")
		book, err := bookModel.FindById(ctx, genBooks[0].GetID(), opts)

		// fmt.Printf("---------- FindById executed in %v ---------- \n", time.Since(start))

		assert.NoError(t, err, "Find should not return error")
		assert.NotNil(t, book, "book should not be nil")

		for _, author := range book.Doc.Authors {
			assert.NotEmpty(t, author.(bson.M)["name"], "author should not be empty")
		}

		chapters := book.Doc.Chapters
		for _, chapter := range chapters {
			author := chapter.Author.(bson.M)
			assert.NotEmpty(t, author["name"], "author should not be empty")
		}

		// fmt.Printf(
		// 	"---------- FindById populated %d paths  ---------- \n",
		// 	len(chapters)+len(book.Doc.Authors),
		// )
	})

	t.Run("Should populate docs with Find", func(t *testing.T) {
		opts := mopt.Find()
		opts.SetPopulate(popOpts...)

		// start := time.Now()
		// fmt.Println("---------- finding & populating docs ----------")

		books, err := bookModel.Find(ctx, bson.M{}, opts)
		assert.NoError(t, err, "Find should not return error")

		// fmt.Printf("---------- executed in %v ---------- \n", time.Since(start))

		pathsPopulated := 0
		for _, book := range books {
			for _, author := range book.Doc.Authors {
				assert.NotEmpty(t, author.(bson.M)["name"], "author should not be empty")
			}

			chapters := book.Doc.Chapters
			for _, chapter := range chapters {
				author := chapter.Author.(bson.M)
				assert.NotEmpty(t, author["name"], "author should not be empty")
			}
			pathsPopulated += len(chapters) + len(book.Doc.Authors)
		}

		// fmt.Printf("---------- %d paths populated ---------- \n", pathsPopulated)
	})
}

func generateBooks(ctx context.Context, db *mongo.Database) []*BookDoc {
	authorModel := mgs.NewModel[Author, *mgs.DefaultSchema](db.Collection("authors"))
	bookModel := mgs.NewModel[Book, *mgs.DefaultSchema](db.Collection("books"))

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
				if a == author.GetID() {
					include = false
				}
			}
			if include {
				bookAuthors = append(bookAuthors, author.GetID())
			}
			chapter := Chapter{
				ID:     primitive.NewObjectID(),
				Title:  fmt.Sprintf("Chapter %d", i),
				Pages:  i * 10,
				Author: author.GetID(),
			}
			chapters = append(chapters, chapter)
		}

		doc := Book{
			Title:    fmt.Sprintf("Book %d", i),
			Authors:  bookAuthors,
			Price:    Price{Currency: "USD", Amount: 10, Decimals: 99},
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
