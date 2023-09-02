package mgs_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/0x-buidl/mgs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
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

type IHookBook = mgs.Document[HookBook, *mgs.DefaultSchema]

var calledHooks = make([]string, 0)

var (
	mu  sync.Mutex
	err error
)

func singleDocHook(args mock.Arguments) {
	arg := args.Get(1).(*mgs.HookArg[HookBook])
	v, ok := arg.Data().(*IHookBook)
	if !ok {
		err = fmt.Errorf("arg.Data() is %T, not %T", arg.Data(), v)
	}
}

func multiDocHook(args mock.Arguments) {
	arg := args.Get(1).(*mgs.HookArg[HookBook])
	v, ok := arg.Data().(*[]*IHookBook)
	if !ok {
		err = fmt.Errorf("arg.Data() is %T, not %T", arg.Data(), v)
	}
}

func queryHook(args mock.Arguments) {
	arg := args.Get(1).(*mgs.HookArg[HookBook])
	v, ok := arg.Data().(*mgs.Query[HookBook])
	if !ok {
		err = fmt.Errorf("arg.Data() is %T, not %T", arg.Data(), v)
	}
}

func updateHook(args mock.Arguments) {
	arg := args.Get(1).(*mgs.HookArg[HookBook])
	v, ok := arg.Data().(*mongo.UpdateResult)
	if !ok {
		err = fmt.Errorf("arg.Data() is %T, not %T", arg.Data(), v)
	}
}

func deleteHook(args mock.Arguments) {
	arg := args.Get(1).(*mgs.HookArg[HookBook])
	v, ok := arg.Data().(*mongo.DeleteResult)
	if !ok {
		err = fmt.Errorf("arg.Data() is %T, not %T", arg.Data(), v)
	}
}

func (b HookBook) BeforeCreate(ctx context.Context, arg *mgs.HookArg[HookBook]) error {
	mu.Lock()
	defer mu.Unlock()
	err = nil

	fn := singleDocHook
	if arg.Operation() == mgs.CreateMany {
		fn = multiDocHook
	}

	b.On("BeforeCreate", ctx, arg).Run(fn)
	calledHooks = append(calledHooks, "BeforeCreate")

	b.Called(ctx, arg)
	return err
}

func (b HookBook) AfterCreate(ctx context.Context, arg *mgs.HookArg[HookBook]) error {
	mu.Lock()
	defer mu.Unlock()
	err = nil

	fn := singleDocHook
	if arg.Operation() == mgs.CreateMany {
		fn = multiDocHook
	}

	b.On("AfterCreate", ctx, arg).Run(fn)
	calledHooks = append(calledHooks, "AfterCreate")

	b.Called(ctx, arg)
	return err
}

func (b HookBook) BeforeSave(ctx context.Context, arg *mgs.HookArg[HookBook]) error {
	mu.Lock()
	defer mu.Unlock()
	err = nil

	fn := singleDocHook
	if arg.Operation() == mgs.CreateMany {
		fn = multiDocHook
	}

	b.On("BeforeSave", ctx, arg).Run(fn)
	calledHooks = append(calledHooks, "BeforeSave")

	b.Called(ctx, arg)
	return err
}

func (b HookBook) AfterSave(ctx context.Context, arg *mgs.HookArg[HookBook]) error {
	mu.Lock()
	defer mu.Unlock()
	err = nil

	fn := singleDocHook
	if arg.Operation() == mgs.CreateMany {
		fn = multiDocHook
	}

	b.On("AfterSave", ctx, arg).Run(fn)
	calledHooks = append(calledHooks, "AfterSave")

	b.Called(ctx, arg)
	return err
}

func (b HookBook) BeforeDelete(ctx context.Context, arg *mgs.HookArg[HookBook]) error {
	mu.Lock()
	defer mu.Unlock()
	err = nil

	b.On("BeforeDelete", ctx, arg).Run(queryHook)
	calledHooks = append(calledHooks, "BeforeDelete")

	b.Called(ctx, arg)
	return err
}

func (b HookBook) AfterDelete(ctx context.Context, arg *mgs.HookArg[HookBook]) error {
	mu.Lock()
	defer mu.Unlock()

	b.On("AfterDelete", ctx, arg).Run(deleteHook)
	calledHooks = append(calledHooks, "AfterDelete")

	b.Called(ctx, arg)
	return err
}

func (b HookBook) BeforeFind(ctx context.Context, arg *mgs.HookArg[HookBook]) error {
	mu.Lock()
	defer mu.Unlock()
	err = nil

	b.On("BeforeFind", ctx, arg).Run(queryHook)
	calledHooks = append(calledHooks, "BeforeFind")

	b.Called(ctx, arg)
	return err
}

func (b HookBook) AfterFind(ctx context.Context, arg *mgs.HookArg[HookBook]) error {
	mu.Lock()
	defer mu.Unlock()
	err = nil

	fn := singleDocHook
	if arg.Operation() == mgs.FindMany {
		fn = multiDocHook
	}

	b.On("AfterFind", ctx, arg).Run(fn)
	calledHooks = append(calledHooks, "AfterFind")

	b.Called(ctx, arg)
	return err
}

func (b HookBook) BeforeUpdate(ctx context.Context, arg *mgs.HookArg[HookBook]) error {
	mu.Lock()
	defer mu.Unlock()
	err = nil

	b.On("BeforeUpdate", ctx, arg).Run(queryHook)
	calledHooks = append(calledHooks, "BeforeUpdate")

	b.Called(ctx, arg)
	return err
}

func (b HookBook) AfterUpdate(ctx context.Context, arg *mgs.HookArg[HookBook]) error {
	mu.Lock()
	defer mu.Unlock()
	err = nil

	b.On("AfterUpdate", ctx, arg).Run(updateHook)
	calledHooks = append(calledHooks, "AfterUpdate")

	b.Called(ctx, arg)
	return err
}

func (b HookBook) BeforeValidate(ctx context.Context, arg *mgs.HookArg[HookBook]) error {
	mu.Lock()
	defer mu.Unlock()
	err = nil

	b.On("BeforeValidate", ctx, arg).Run(singleDocHook)
	calledHooks = append(calledHooks, "BeforeValidate")

	b.Called(ctx, arg)
	return err
}

func (b HookBook) Validate(ctx context.Context, arg *mgs.HookArg[HookBook]) error {
	mu.Lock()
	defer mu.Unlock()
	err = nil

	b.On("Validate", ctx, arg).Run(singleDocHook)
	calledHooks = append(calledHooks, "Validate")

	b.Called(ctx, arg)
	return err
}

func (b HookBook) AfterValidate(ctx context.Context, arg *mgs.HookArg[HookBook]) error {
	mu.Lock()
	defer mu.Unlock()
	err = nil

	b.On("AfterValidate", ctx, arg).Run(singleDocHook)
	calledHooks = append(calledHooks, "AfterValidate")

	b.Called(ctx, arg)
	return err
}

func TestCreateHooks(t *testing.T) {
	ctx := context.Background()
	db, cleanup := getDb(ctx)
	defer cleanup(ctx)

	bookModel := mgs.NewModel[HookBook, *mgs.DefaultSchema](db.Collection("books"))

	t.Run("Should run Model.CreateOne hooks", func(t *testing.T) {
		calledHooks = make([]string, 0)

		b := HookBook{}
		_, err := bookModel.CreateOne(ctx, b)

		assert.NoError(t, err)

		hooksToRun := map[string]bool{
			"BeforeValidate": true, "Validate": true, "AfterValidate": true,
			"BeforeCreate": true, "BeforeSave": true, "AfterSave": true, "AfterCreate": true,
		}

		// 7 hooks should be called
		expectedCalls := 7

		if len(calledHooks) != expectedCalls {
			t.Errorf("Expected %d hooks to be called, got %d", expectedCalls, len(calledHooks))
		}

		for _, h := range calledHooks {
			if _, ok := hooksToRun[h]; !ok {
				t.Errorf("Unexpected hook called: %s", h)
			}
		}
	})

	t.Run("Should run Model.CreateMany hooks", func(t *testing.T) {
		calledHooks = make([]string, 0)

		books := []HookBook{{}, {}, {}}
		_, err := bookModel.CreateMany(ctx, books)

		assert.NoError(t, err)

		hooksToRun := map[string]bool{
			"BeforeValidate": true, "Validate": true, "AfterValidate": true,
			"BeforeCreate": true, "BeforeSave": true, "AfterSave": true, "AfterCreate": true,
		}

		/**
				10 hooks should be called.
		    (3 books * 3 validate hooks) + 4 (create & save hooks)
		*/
		expectedCalls := 13

		if len(calledHooks) != expectedCalls {
			t.Errorf("Expected %d hooks to be called, got %d", expectedCalls, len(calledHooks))
		}

		for _, h := range calledHooks {
			if _, ok := hooksToRun[h]; !ok {
				t.Errorf("Unexpected hook called: %s", h)
			}
		}
	})
}

func TestUpdateHooks(t *testing.T) {
	ctx := context.Background()
	db, cleanup := getDb(ctx)
	defer cleanup(ctx)

	bookModel := mgs.NewModel[HookBook, *mgs.DefaultSchema](db.Collection("books"))

	t.Run("Should run Model.UpdateOne and Model.UpdateMany hooks", func(t *testing.T) {
		calledHooks = make([]string, 0)

		query := bson.M{}
		update := bson.M{"$set": bson.M{"title": "test2"}}
		_, err := bookModel.UpdateMany(ctx, query, update)
		assert.NoError(t, err)

		_, err = bookModel.UpdateOne(ctx, query, update)
		assert.NoError(t, err)

		hooksToRun := map[string]bool{
			"BeforeUpdate": true, "AfterUpdate": true,
		}

		expectedCalls := 4
		if len(calledHooks) != expectedCalls {
			t.Errorf("Expected %d hooks to be called, got %d", expectedCalls, len(calledHooks))
		}

		for _, h := range calledHooks {
			if _, ok := hooksToRun[h]; !ok {
				t.Errorf("Unexpected hook called: %s", h)
			}
		}
	})
}

func TestFindHooks(t *testing.T) {
	ctx := context.Background()
	db, cleanup := getDb(ctx)
	defer cleanup(ctx)

	bookModel := mgs.NewModel[HookBook, *mgs.DefaultSchema](db.Collection("books"))
	books := generateBooks(ctx, db)

	t.Run("Should run Model.FindOne and Model.FindMany hooks", func(t *testing.T) {
		calledHooks = make([]string, 0)
		query := bson.M{}
		_, err := bookModel.Find(ctx, query)
		assert.NoError(t, err)

		_, err = bookModel.FindOne(ctx, query)
		assert.NoError(t, err)

		_, err = bookModel.FindById(ctx, books[0].GetID())
		assert.NoError(t, err)

		hooksToRun := map[string]bool{
			"BeforeFind": true, "AfterFind": true,
		}

		expectedCalls := 6
		if len(calledHooks) != expectedCalls {
			t.Errorf("Expected %d hooks to be called, got %d", expectedCalls, len(calledHooks))
		}

		for _, h := range calledHooks {
			if _, ok := hooksToRun[h]; !ok {
				t.Errorf("Unexpected hook called: %s", h)
			}
		}
	})
}

func TestDeleteHooks(t *testing.T) {
	ctx := context.Background()
	db, cleanup := getDb(ctx)
	defer cleanup(ctx)

	bookModel := mgs.NewModel[HookBook, *mgs.DefaultSchema](db.Collection("books"))
	generateBooks(ctx, db)

	t.Run("Should run Model.DeleteOne and Model.DeleteMany hooks", func(t *testing.T) {
		calledHooks = make([]string, 0)

		query := bson.M{}
		_, err = bookModel.DeleteOne(ctx, query)
		assert.NoError(t, err)

		_, err := bookModel.DeleteMany(ctx, query)
		assert.NoError(t, err)

		hooksToRun := map[string]bool{
			"BeforeDelete": true, "AfterDelete": true,
		}

		expectedCalls := 4
		if len(calledHooks) != expectedCalls {
			t.Errorf("Expected %d hooks to be called, got %d", expectedCalls, len(calledHooks))
		}

		for _, h := range calledHooks {
			if _, ok := hooksToRun[h]; !ok {
				t.Errorf("Unexpected hook called: %s", h)
			}
		}
	})
}
