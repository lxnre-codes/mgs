package database

import (
	"context"
	"fmt"
	"time"

	"github.com/0x-buidl/mgs"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Book struct {
	Title string `json:"title"    bson:"title"`
	// ObjectID or Author Object
	Author    interface{}   `json:"author"   bson:"author"`
	Reviews   []interface{} `json:"chapters" bson:"chapters"`
	Deleted   bool          `json:"-"        bson:"deleted"`
	DeletedAt *time.Time    `json:"-"        bson:"deletedAt"`
}

type (
	BookModel = mgs.Model[Book, *mgs.DefaultSchema]
	BookDoc   = mgs.Document[Book, *mgs.DefaultSchema]
)

func NewBookModel(coll *mongo.Collection) *BookModel {
	return mgs.NewModel[Book, *mgs.DefaultSchema](coll)
}

func (Book) Validate(ctx context.Context, arg *mgs.HookArg[Book]) error {
	var err error

	book := arg.Data().(*BookDoc)
	if _, ok := book.Doc.Author.(primitive.ObjectID); !ok {
		err = fmt.Errorf("author must be ObjectID")
	}

	return err
}

func (Book) BeforeValidate(ctx context.Context, arg *mgs.HookArg[Book]) error {
	return nil
}

func (Book) AfterValidate(ctx context.Context, arg *mgs.HookArg[Book]) error {
	return nil
}

func (Book) BeforeCreate(ctx context.Context, arg *mgs.HookArg[Book]) error {
	return nil
}

func (book Book) AfterCreate(ctx context.Context, arg *mgs.HookArg[Book]) error {
	return nil
}

func (book Book) BeforeUpdate(ctx context.Context, arg *mgs.HookArg[Book]) error {
	return nil
}

func (book Book) AfterUpdate(ctx context.Context, arg *mgs.HookArg[Book]) error {
	return nil
}

func (book Book) BeforeDelete(ctx context.Context, arg *mgs.HookArg[Book]) error {
	return nil
}

func (book Book) AfterDelete(ctx context.Context, arg *mgs.HookArg[Book]) error {
	return nil
}

func (book Book) BeforeFind(ctx context.Context, arg *mgs.HookArg[Book]) error {
	q := *arg.Data().(*mgs.Query[Book]).Filter
	q["deleted"] = false
	return nil
}

func (book Book) AfterFind(ctx context.Context, arg *mgs.HookArg[Book]) error {
	return nil
}
