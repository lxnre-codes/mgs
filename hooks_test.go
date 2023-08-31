package mgs_test

import (
	"context"
	"testing"
	"time"

	mgs "github.com/0x-buidl/go-mongoose"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type HookBook struct {
	mock.Mock `bson:"-"`
	Title     string `bson:"title"     json:"title"    validate:"required"`
	// ObjectID or Author object
	Authors   []interface{} `bson:"authors"   json:"authors"  validate:"required,min=1"`
	Chapters  []Chapter     `bson:"chapters"  json:"chapters" validate:"required,min=1"`
	Price     float64       `bson:"price"     json:"price"`
	Deleted   bool          `bson:"deleted"   json:"-"`
	DeletedAt *time.Time    `bson:"deletedAt" json:"-"`
}

func (b *HookBook) BeforeCreate(ctx context.Context, arg *mgs.HookArg[HookBook]) error {
	args := b.Called()
	return args.Error(0)
}

func (b *HookBook) AfterCreate(ctx context.Context, arg *mgs.HookArg[HookBook]) error {
	args := b.Called()
	return args.Error(0)
}

func (b *HookBook) BeforeSave(ctx context.Context, arg *mgs.HookArg[HookBook]) error {
	args := b.Called()
	return args.Error(0)
}

func (b *HookBook) AfterSave(ctx context.Context, arg *mgs.HookArg[HookBook]) error {
	args := b.Called()
	return args.Error(0)
}

func (b *HookBook) BeforeDelete(ctx context.Context, arg *mgs.HookArg[HookBook]) error {
	args := b.Called()
	return args.Error(0)
}

func (b *HookBook) AfterDelete(ctx context.Context, arg *mgs.HookArg[HookBook]) error {
	args := b.Called()
	return args.Error(0)
}

func (b *HookBook) BeforeFind(ctx context.Context, arg *mgs.HookArg[HookBook]) error {
	args := b.Called()
	return args.Error(0)
}

func (b *HookBook) AfterFind(ctx context.Context, arg *mgs.HookArg[HookBook]) error {
	args := b.Called(ctx, arg)
	return args.Error(0)
}

func (b *HookBook) BeforeUpdate(ctx context.Context, arg *mgs.HookArg[HookBook]) error {
	args := b.Called()
	return args.Error(0)
}

func (b *HookBook) AfterUpdate(ctx context.Context, arg *mgs.HookArg[HookBook]) error {
	args := b.Called()
	return args.Error(0)
}

func (b *HookBook) BeforeValidate(ctx context.Context, arg *mgs.HookArg[HookBook]) error {
	args := b.Called()
	return args.Error(0)
}

func (b *HookBook) Validate(ctx context.Context, arg *mgs.HookArg[HookBook]) error {
	args := b.Called()
	return args.Error(0)
}

func (b *HookBook) AfterValidate(ctx context.Context, arg *mgs.HookArg[HookBook]) error {
	args := b.Called()
	return args.Error(0)
}

func TestCreateHooks(t *testing.T) {
	ctx := context.Background()

	t.Run("should run before  after create many ", func(t *testing.T) {
		db, teardown := setup()
		defer teardown(ctx)

		bookModel := mgs.NewModel[HookBook](db.Collection("books"))

		b := HookBook{Title: "test1"}

		b.On("BeforeValidate").Return(nil)
		b.On("Validate").Return(nil)
		b.On("AfterValidate").Return(nil)
		b.On("BeforeCreate").Return(nil)
		b.On("BeforeSave").Return(nil)
		b.On("AfterSave").Return(nil)
		b.On("AfterCreate").Return(nil)

		_, err := bookModel.CreateOne(ctx, b)
		assert.NoError(t, err)

		b.AssertExpectations(t)
	})
}
