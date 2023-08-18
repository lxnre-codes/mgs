package mgs_test

import (
	"context"
	"testing"

	mgs "github.com/0x-buidl/go-mongoose"
	"github.com/stretchr/testify/assert"
)

func (b *Book) BeforeCreate(ctx context.Context, arg *mgs.HookArg[Book]) error {
	args := b.Called()
	return args.Error(0)
}

func (b *Book) AfterCreate(ctx context.Context, arg *mgs.HookArg[Book]) error {
	args := b.Called()
	return args.Error(0)
}

func (b *Book) BeforeSave(ctx context.Context, arg *mgs.HookArg[Book]) error {
	args := b.Called()
	return args.Error(0)
}

func (b *Book) AfterSave(ctx context.Context, arg *mgs.HookArg[Book]) error {
	args := b.Called()
	return args.Error(0)
}

func (b *Book) BeforeDelete(ctx context.Context, arg *mgs.HookArg[Book]) error {
	args := b.Called()
	return args.Error(0)
}

func (b *Book) AfterDelete(ctx context.Context, arg *mgs.HookArg[Book]) error {
	args := b.Called()
	return args.Error(0)
}

func (b *Book) BeforeFind(ctx context.Context, arg *mgs.HookArg[Book]) error {
	args := b.Called()
	return args.Error(0)
}

func (b *Book) AfterFind(ctx context.Context, arg *mgs.HookArg[Book]) error {
	args := b.Called(ctx, arg)
	return args.Error(0)
}

func (b *Book) BeforeUpdate(ctx context.Context, arg *mgs.HookArg[Book]) error {
	args := b.Called()
	return args.Error(0)
}

func (b *Book) AfterUpdate(ctx context.Context, arg *mgs.HookArg[Book]) error {
	args := b.Called()
	return args.Error(0)
}

func (b *Book) BeforeValidate(ctx context.Context, arg *mgs.HookArg[Book]) error {
	args := b.Called()
	return args.Error(0)
}

func (b *Book) Validate(ctx context.Context, arg *mgs.HookArg[Book]) error {
	args := b.Called()
	return args.Error(0)
}

func (b *Book) AfterValidate(ctx context.Context, arg *mgs.HookArg[Book]) error {
	args := b.Called()
	return args.Error(0)
}

func TestCreateHooks(t *testing.T) {
	ctx := context.Background()

	t.Run("should run before and after create many hooks", func(t *testing.T) {
		db, teardown := setup()
		defer teardown(ctx)

		bookModel := mgs.NewModel[Book](db.Collection("books"))

		b := Book{Title: "test1"}

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
