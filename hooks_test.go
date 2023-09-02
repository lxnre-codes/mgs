package mgs_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/0x-buidl/mgs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type HookBook struct {
	mock.Mock `bson:"-"`
	Title     string `bson:"title"     json:"title"    validate:"required"`
	// ObjectID or Author object
	Authors   []interface{} `bson:"authors"   json:"authors"  validate:"required,min=1"`
	Chapters  []Chapter     `bson:"chapters"  json:"chapters" validate:"required,min=1"`
	Price     Price         `bson:"price"     json:"price"`
	Deleted   bool          `bson:"deleted"   json:"-"`
	DeletedAt *time.Time    `bson:"deletedAt" json:"-"`
}

type IHookBook = mgs.Document[HookBook, *mgs.DefaultSchema]

var calledHooks = make([]string, 0)

var (
	mu  sync.Mutex
	err error
)

var (
	errorBV = errors.New("before validation error")
	errorV  = errors.New("validation error")
	errorAV = errors.New("after validation error")
	errorBC = errors.New("before create error")
	errorAC = errors.New("after create error")
	errorBS = errors.New("before save error")
	errorAS = errors.New("after save error")
	errorBU = errors.New("before update error")
	errorAU = errors.New("after update error")
	errorBD = errors.New("before delete error")
	errorAD = errors.New("after delete error")
	errorBF = errors.New("before find error")
	errorAF = errors.New("after find error")
)

var hookErrors = make(map[string]error)

func resetHookVars() {
	calledHooks = make([]string, 0)
	hookErrors = make(map[string]error)
}

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

func queryOrDocHook(args mock.Arguments) {
	arg := args.Get(1).(*mgs.HookArg[HookBook])
	q, qok := arg.Data().(*mgs.Query[HookBook])
	d, dok := arg.Data().(*IHookBook)

	if !qok && !dok {
		err = fmt.Errorf("arg.Data() is %T, not %T or %T", arg.Data(), q, d)
	}
}

func deleteOrDocHook(args mock.Arguments) {
	arg := args.Get(1).(*mgs.HookArg[HookBook])
	d, dok := arg.Data().(*IHookBook)
	dr, drok := arg.Data().(*mongo.DeleteResult)

	if !dok && !drok {
		err = fmt.Errorf("arg.Data() is %T, not %T or %T", arg.Data(), d, dr)
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
	if err, ok := hookErrors["BeforeCreate"]; ok {
		return err
	}

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
	if err, ok := hookErrors["AfterCreate"]; ok {
		return err
	}

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
	if err, ok := hookErrors["BeforeSave"]; ok {
		return err
	}

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
	if err, ok := hookErrors["AfterSave"]; ok {
		return err
	}

	b.Called(ctx, arg)
	return err
}

func (b HookBook) BeforeDelete(ctx context.Context, arg *mgs.HookArg[HookBook]) error {
	mu.Lock()
	defer mu.Unlock()
	err = nil

	b.On("BeforeDelete", ctx, arg).Run(queryOrDocHook)
	calledHooks = append(calledHooks, "BeforeDelete")
	if err, ok := hookErrors["BeforeDelete"]; ok {
		return err
	}

	b.Called(ctx, arg)
	return err
}

func (b HookBook) AfterDelete(ctx context.Context, arg *mgs.HookArg[HookBook]) error {
	mu.Lock()
	defer mu.Unlock()

	b.On("AfterDelete", ctx, arg).Run(deleteOrDocHook)
	calledHooks = append(calledHooks, "AfterDelete")
	if err, ok := hookErrors["AfterDelete"]; ok {
		return err
	}

	b.Called(ctx, arg)
	return err
}

func (b HookBook) BeforeFind(ctx context.Context, arg *mgs.HookArg[HookBook]) error {
	mu.Lock()
	defer mu.Unlock()
	err = nil

	b.On("BeforeFind", ctx, arg).Run(queryHook)
	calledHooks = append(calledHooks, "BeforeFind")
	if err, ok := hookErrors["BeforeFind"]; ok {
		return err
	}

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
	if err, ok := hookErrors["AfterFind"]; ok {
		return err
	}

	b.Called(ctx, arg)
	return err
}

func (b HookBook) BeforeUpdate(ctx context.Context, arg *mgs.HookArg[HookBook]) error {
	mu.Lock()
	defer mu.Unlock()
	err = nil

	b.On("BeforeUpdate", ctx, arg).Run(queryHook)
	calledHooks = append(calledHooks, "BeforeUpdate")
	if err, ok := hookErrors["BeforeUpdate"]; ok {
		return err
	}

	b.Called(ctx, arg)
	return err
}

func (b HookBook) AfterUpdate(ctx context.Context, arg *mgs.HookArg[HookBook]) error {
	mu.Lock()
	defer mu.Unlock()
	err = nil

	b.On("AfterUpdate", ctx, arg).Run(updateHook)
	calledHooks = append(calledHooks, "AfterUpdate")
	if err, ok := hookErrors["AfterUpdate"]; ok {
		return err
	}

	b.Called(ctx, arg)
	return err
}

func (b HookBook) BeforeValidate(ctx context.Context, arg *mgs.HookArg[HookBook]) error {
	mu.Lock()
	defer mu.Unlock()
	err = nil

	b.On("BeforeValidate", ctx, arg).Run(singleDocHook)
	calledHooks = append(calledHooks, "BeforeValidate")
	if he, ok := hookErrors["BeforeValidate"]; ok {
		err = he
	}

	b.Called(ctx, arg)
	return err
}

func (b HookBook) Validate(ctx context.Context, arg *mgs.HookArg[HookBook]) error {
	mu.Lock()
	defer mu.Unlock()
	err = nil

	b.On("Validate", ctx, arg).Run(singleDocHook)
	calledHooks = append(calledHooks, "Validate")
	if he, ok := hookErrors["Validate"]; ok {
		err = he
	}

	b.Called(ctx, arg)
	return err
}

func (b HookBook) AfterValidate(ctx context.Context, arg *mgs.HookArg[HookBook]) error {
	mu.Lock()
	defer mu.Unlock()
	err = nil

	b.On("AfterValidate", ctx, arg).Run(singleDocHook)
	calledHooks = append(calledHooks, "AfterValidate")
	if he, ok := hookErrors["AfterValidate"]; ok {
		err = he
	}

	b.Called(ctx, arg)
	return err
}

func TestCreateHooks(t *testing.T) {
	ctx := context.Background()
	db, cleanup := getDb(ctx)
	defer cleanup(ctx)

	bookModel := mgs.NewModel[HookBook, *mgs.DefaultSchema](db.Collection("books"))

	t.Run("Should run Model.CreateOne hooks successfully", func(t *testing.T) {
		resetHookVars()

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
		resetHookVars()

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

	t.Run("Should pass Document Save Hooks", func(t *testing.T) {
		resetHookVars()

		doc := bookModel.NewDocument(HookBook{})
		err := doc.Save(ctx)
		assert.NoError(t, err)

		doc.Doc.Title = "New Title"
		err = doc.Save(ctx)
		assert.NoError(t, err)

		hooksToRun := map[string]bool{
			"BeforeValidate": true, "Validate": true, "AfterValidate": true,
			"BeforeSave": true, "AfterSave": true,
		}

		/**
				10 hooks should be called.
		    (2 books * 3 validate hooks) + 4 (save hooks)
		*/
		expectedCalls := 10

		if len(calledHooks) != expectedCalls {
			t.Errorf("Expected %d hooks to be called, got %d", expectedCalls, len(calledHooks))
		}

		for _, h := range calledHooks {
			if _, ok := hooksToRun[h]; !ok {
				t.Errorf("Unexpected hook called: %s", h)
			}
		}
	})

	t.Run("Should fail Model Validate Hooks", func(t *testing.T) {
		resetHookVars()

		hookErrors["BeforeValidate"] = errorBV

		books := []HookBook{{}, {}, {}}

		_, err := bookModel.CreateMany(ctx, books)
		assert.Error(t, err)
		assert.Equal(t, errorBV, err)

		_, err = bookModel.CreateOne(ctx, HookBook{})
		assert.Error(t, err)
		assert.Equal(t, errorBV, err)

		hookErrors["BeforeValidate"] = nil
		hookErrors["Validate"] = errorV

		_, err = bookModel.CreateMany(ctx, books)
		assert.Error(t, err)
		assert.Equal(t, errorV, err)

		_, err = bookModel.CreateOne(ctx, HookBook{})
		assert.Error(t, err)
		assert.Equal(t, errorV, err)

		hookErrors["Validate"] = nil
		hookErrors["AfterValidate"] = errorAV

		_, err = bookModel.CreateMany(ctx, books)
		assert.Error(t, err)
		assert.Equal(t, errorAV, err)

		_, err = bookModel.CreateOne(ctx, HookBook{})
		assert.Error(t, err)
		assert.Equal(t, errorAV, err)
	})

	t.Run("Should fail Model Create Hooks", func(t *testing.T) {
		resetHookVars()

		books := []HookBook{{}, {}, {}}
		hookErrors["BeforeCreate"] = errorBC
		_, err := bookModel.CreateMany(ctx, books)
		assert.Error(t, err)
		assert.Equal(t, errorBC, err)

		_, err = bookModel.CreateOne(ctx, HookBook{})
		assert.Error(t, err)
		assert.Equal(t, errorBC, err)

		hookErrors["BeforeCreate"] = nil
		hookErrors["BeforeSave"] = errorBS
		_, err = bookModel.CreateMany(ctx, books)
		assert.Error(t, err)
		assert.Equal(t, errorBS, err)

		_, err = bookModel.CreateOne(ctx, HookBook{})
		assert.Error(t, err)
		assert.Equal(t, errorBS, err)

		hookErrors["BeforeSave"] = nil
		hookErrors["AfterSave"] = errorAS

		_, err = bookModel.CreateMany(ctx, books)
		assert.Error(t, err)
		assert.Equal(t, errorAS, err)

		_, err = bookModel.CreateOne(ctx, HookBook{})
		assert.Error(t, err)
		assert.Equal(t, errorAS, err)

		hookErrors["AfterSave"] = nil
		hookErrors["AfterCreate"] = errorAC

		_, err = bookModel.CreateMany(ctx, books)
		assert.Error(t, err)
		assert.Equal(t, errorAC, err)

		_, err = bookModel.CreateOne(ctx, HookBook{})
		assert.Error(t, err)
		assert.Equal(t, errorAC, err)
	})

	t.Run("Should cancel validations for Model create many", func(t *testing.T) {
		resetHookVars()
		books := []HookBook{{}, {}, {}}
		ctx, cancel := context.WithTimeout(ctx, time.Nanosecond)
		defer cancel()
		_, err := bookModel.CreateMany(ctx, books)
		assert.Error(t, err)
	})

	t.Run("Should fail Document Validate Hooks", func(t *testing.T) {
		resetHookVars()

		doc := bookModel.NewDocument(HookBook{})

		hookErrors["BeforeValidate"] = errorBV

		err := doc.Save(ctx)
		assert.Error(t, err)
		assert.Equal(t, errorBV, err)

		hookErrors["BeforeValidate"] = nil
		hookErrors["Validate"] = errorV

		err = doc.Save(ctx)
		assert.Error(t, err)
		assert.Equal(t, errorV, err)

		hookErrors["Validate"] = nil
		hookErrors["AfterValidate"] = errorAV

		err = doc.Save(ctx)
		assert.Error(t, err)
		assert.Equal(t, errorAV, err)
	})

	t.Run("Should fail Document Save Hooks", func(t *testing.T) {
		resetHookVars()

		doc := bookModel.NewDocument(HookBook{})

		hookErrors["BeforeSave"] = errorBS

		err := doc.Save(ctx)
		assert.Error(t, err)
		assert.Equal(t, errorBS, err)

		hookErrors["BeforeSave"] = nil
		hookErrors["AfterSave"] = errorAS

		err = doc.Save(ctx)
		assert.Error(t, err)
		assert.Equal(t, errorAS, err)
	})
}

func TestUpdateHooks(t *testing.T) {
	ctx := context.Background()
	db, cleanup := getDb(ctx)
	defer cleanup(ctx)

	bookModel := mgs.NewModel[HookBook, *mgs.DefaultSchema](db.Collection("books"))

	t.Run("Should run Model.UpdateOne and Model.UpdateMany hooks", func(t *testing.T) {
		resetHookVars()

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

	t.Run("Should fail Update Hooks", func(t *testing.T) {
		resetHookVars()

		hookErrors["BeforeUpdate"] = errorBU

		query := bson.M{}
		update := bson.M{"$push": bson.M{"authors": primitive.NewObjectID()}}
		_, err := bookModel.UpdateMany(ctx, query, update)
		assert.Error(t, err)
		assert.Equal(t, errorBU, err)

		_, err = bookModel.UpdateOne(ctx, query, update)
		assert.Error(t, err)
		assert.Equal(t, errorBU, err)

		hookErrors["BeforeUpdate"] = nil
		hookErrors["AfterUpdate"] = errorAU

		_, err = bookModel.UpdateMany(ctx, query, update)
		assert.Error(t, err)
		assert.Equal(t, errorAU, err)

		_, err = bookModel.UpdateOne(ctx, query, update)
		assert.Error(t, err)
		assert.Equal(t, errorAU, err)
	})
}

func TestFindHooks(t *testing.T) {
	ctx := context.Background()
	db, cleanup := getDb(ctx)
	defer cleanup(ctx)

	bookModel := mgs.NewModel[HookBook, *mgs.DefaultSchema](db.Collection("books"))
	books := generateBooks(ctx, db)

	t.Run("Should run Model.FindOne and Model.FindMany hooks", func(t *testing.T) {
		resetHookVars()

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

	t.Run("Should fail Find Hooks", func(t *testing.T) {
		resetHookVars()

		hookErrors["BeforeFind"] = errorBF

		query := bson.M{}
		_, err := bookModel.Find(ctx, query)
		assert.Error(t, err)
		assert.Equal(t, errorBF, err)

		_, err = bookModel.FindOne(ctx, query)
		assert.Error(t, err)
		assert.Equal(t, errorBF, err)

		_, err = bookModel.FindById(ctx, books[0].GetID())
		assert.Error(t, err)
		assert.Equal(t, errorBF, err)

		hookErrors["BeforeFind"] = nil
		hookErrors["AfterFind"] = errorAF

		_, err = bookModel.Find(ctx, query)
		assert.Error(t, err)
		assert.Equal(t, errorAF, err)

		_, err = bookModel.FindOne(ctx, query)
		assert.Error(t, err)
		assert.Equal(t, errorAF, err)

		_, err = bookModel.FindById(ctx, books[0].GetID())
		assert.Error(t, err)
		assert.Equal(t, errorAF, err)
	})
}

func TestDeleteHooks(t *testing.T) {
	ctx := context.Background()
	db, cleanup := getDb(ctx)
	defer cleanup(ctx)

	bookModel := mgs.NewModel[HookBook, *mgs.DefaultSchema](db.Collection("books"))
	// generateBooks(ctx, db)

	t.Run("Should run Model.DeleteOne and Model.DeleteMany hooks", func(t *testing.T) {
		resetHookVars()

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

	t.Run("Should run Document.Delete hooks", func(t *testing.T) {
		doc := bookModel.NewDocument(HookBook{})
		err := doc.Save(ctx)
		assert.NoError(t, err)

		resetHookVars()

		err = doc.Delete(ctx)
		assert.NoError(t, err)

		hooksToRun := map[string]bool{
			"BeforeDelete": true, "AfterDelete": true,
		}

		expectedCalls := 2

		if len(calledHooks) != expectedCalls {
			t.Errorf("Expected %d hooks to be called, got %d", expectedCalls, len(calledHooks))
		}

		for _, h := range calledHooks {
			if _, ok := hooksToRun[h]; !ok {
				t.Errorf("Unexpected hook called: %s", h)
			}
		}
	})

	t.Run("Should fail Model Delete Hooks", func(t *testing.T) {
		resetHookVars()

		hookErrors["BeforeDelete"] = errorBD

		query := bson.M{}
		_, err := bookModel.DeleteOne(ctx, query)
		assert.Error(t, err)
		assert.Equal(t, errorBD, err)

		_, err = bookModel.DeleteMany(ctx, query)
		assert.Error(t, err)
		assert.Equal(t, errorBD, err)

		hookErrors["BeforeDelete"] = nil
		hookErrors["AfterDelete"] = errorAD

		_, err = bookModel.DeleteOne(ctx, query)
		assert.Error(t, err)
		assert.Equal(t, errorAD, err)

		_, err = bookModel.DeleteMany(ctx, query)
		assert.Error(t, err)
		assert.Equal(t, errorAD, err)
	})

	t.Run("Should fail Document Delete Hooks", func(t *testing.T) {
		doc := bookModel.NewDocument(HookBook{})
		err := doc.Save(ctx)
		assert.NoError(t, err)

		resetHookVars()

		hookErrors["BeforeDelete"] = errorBD

		err = doc.Delete(ctx)
		assert.Error(t, err)
		assert.Equal(t, errorBD, err)

		hookErrors["BeforeDelete"] = nil
		hookErrors["AfterDelete"] = errorAD

		err = doc.Delete(ctx)
		assert.Error(t, err)
		assert.Equal(t, errorAD, err)
	})
}
