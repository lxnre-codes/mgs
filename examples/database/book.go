package database

import (
	"context"
	"time"

	"github.com/0x-buidl/mgs"
	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Book struct {
	Title     string             `json:"title"  bson:"title"`
	Author    primitive.ObjectID `json:"author" bson:"author"`
	Price     float64            `json:"price"  bson:"price"`
	Deleted   bool               `json:"-"      bson:"deleted"`
	DeletedAt *time.Time         `json:"-"      bson:"deletedAt"`
}

type (
	BookModel = mgs.Model[Book, *mgs.DefaultSchema]
	BookDoc   = mgs.Document[Book, *mgs.DefaultSchema]
)

func NewBookModel(coll *mongo.Collection) *mgs.Model[Book, *mgs.DefaultSchema] {
	return mgs.NewModel[Book, *mgs.DefaultSchema](coll)
}

func (book *Book) Validate(ctx context.Context, arg *mgs.HookArg[Book]) error {
	v := validator.New()
	return v.StructCtx(ctx, book)
}

func (book *Book) BeforeValidate(ctx context.Context, arg *mgs.HookArg[Book]) error {
	return nil
}

func (book *Book) AfterValidate(ctx context.Context, arg *mgs.HookArg[Book]) error {
	return nil
}

func (book *Book) BeforeCreate(ctx context.Context, arg *mgs.HookArg[Book]) error {
	return nil
}

func (book *Book) AfterCreate(ctx context.Context, arg *mgs.HookArg[Book]) error {
	return nil
}

func (book *Book) BeforeUpdate(ctx context.Context, arg *mgs.HookArg[Book]) error {
	return nil
}

func (book *Book) AfterUpdate(ctx context.Context, arg *mgs.HookArg[Book]) error {
	return nil
}

func (book *Book) BeforeDelete(ctx context.Context, arg *mgs.HookArg[Book]) error {
	return nil
}

func (book *Book) AfterDelete(ctx context.Context, arg *mgs.HookArg[Book]) error {
	return nil
}

func (book *Book) BeforeFind(ctx context.Context, arg *mgs.HookArg[Book]) error {
	return nil
}

func (book *Book) AfterFind(ctx context.Context, arg *mgs.HookArg[Book]) error {
	return nil
}
