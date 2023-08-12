package mgs

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
)

type BeforeCreateHook interface {
	// ctx is the context passed to the Create method. doc is the Document[T] .
	BeforeCreate(ctx context.Context, doc interface{}) error
}

type AfterCreateHook interface {
	AfterCreate(ctx context.Context, doc interface{}) error
}

type BeforeDeleteHook interface {
	BeforeDelete(ctx context.Context) error
}

type AfterDeleteHook interface {
	AfterDelete(ctx context.Context) error
}

type BeforeFindHook interface {
	// ctx is the context passed to the Find method. query is the query passed to the Find method.
	BeforeFind(ctx context.Context, query bson.M) error
}

type AfterFindHook interface {
	AfterFind(ctx context.Context) error
}

type BeforeSaveHook interface {
	// ctx is the context passed to the Create method. doc is the Document[T].
	BeforeSave(ctx context.Context, doc interface{}) error
}

type AfterSaveHook interface {
	AfterSave(ctx context.Context, doc interface{}) error
}

type BeforeUpdateHook interface {
	BeforeUpdate(ctx context.Context) error
}

type AfterUpdateHook interface {
	AfterUpdate(ctx context.Context) error
}

func runBeforeCreateHooks[T any](ctx context.Context, doc *Document[T]) error {
	var d interface{} = doc.Doc
	if hook, ok := d.(BeforeCreateHook); ok {
		if err := hook.BeforeCreate(ctx, doc); err != nil {
			return err
		}
	}

	if hook, ok := d.(BeforeSaveHook); ok {
		if err := hook.BeforeSave(ctx, doc); err != nil {
			return err
		}
	}
	return nil
}

func runAfterCreateHooks[T any](ctx context.Context, doc *Document[T]) error {
	var d interface{} = doc.Doc
	if hook, ok := d.(AfterCreateHook); ok {
		if err := hook.AfterCreate(ctx, doc); err != nil {
			return err
		}
	}

	if hook, ok := d.(AfterSaveHook); ok {
		if err := hook.AfterSave(ctx, doc); err != nil {
			return err
		}
	}
	return nil
}

func runBeforeDeleteHooks[T any](ctx context.Context, doc *Document[T]) error {
	var d interface{} = doc.Doc
	if hook, ok := d.(BeforeDeleteHook); ok {
		if err := hook.BeforeDelete(ctx); err != nil {
			return err
		}
	}
	return nil
}

func runAfterDeleteHooks[T any](ctx context.Context, doc *Document[T]) error {
	var d interface{} = doc.Doc
	if hook, ok := d.(AfterDeleteHook); ok {
		if err := hook.AfterDelete(ctx); err != nil {
			return err
		}
	}
	return nil
}

func runBeforeFindHooks[T any](ctx context.Context, doc *Document[T], query bson.M) error {
	var d interface{} = doc.Doc
	if hook, ok := d.(BeforeFindHook); ok {
		return hook.BeforeFind(ctx, query)
	}
	return nil
}

func runAfterFindHooks[T any](ctx context.Context, doc *Document[T]) error {
	var d interface{} = doc.Doc
	if hook, ok := d.(AfterFindHook); ok {
		if err := hook.AfterFind(ctx); err != nil {
			return err
		}
	}
	return nil
}

func runBeforeSaveHooks[T any](ctx context.Context, doc *Document[T]) error {
	var d interface{} = doc.Doc
	if hook, ok := d.(BeforeSaveHook); ok {
		if err := hook.BeforeSave(ctx, doc); err != nil {
			return err
		}
	}
	return nil
}

func runAfterSaveHooks[T any](ctx context.Context, doc *Document[T]) error {
	var d interface{} = doc.Doc
	if hook, ok := d.(AfterSaveHook); ok {
		if err := hook.AfterSave(ctx, doc); err != nil {
			return err
		}
	}
	return nil
}

func runBeforeUpdateHooks[T any](ctx context.Context, doc *Document[T]) error {
	var d interface{} = doc.Doc
	if hook, ok := d.(BeforeSaveHook); ok {
		if err := hook.BeforeSave(ctx, doc); err != nil {
			return err
		}
	}

	if hook, ok := d.(BeforeUpdateHook); ok {
		if err := hook.BeforeUpdate(ctx); err != nil {
			return err
		}
	}
	return nil
}

func runAfterUpdateHooks[T any](ctx context.Context, doc *Document[T]) error {
	var d interface{} = doc.Doc
	if hook, ok := d.(AfterUpdateHook); ok {
		if err := hook.AfterUpdate(ctx); err != nil {
			return err
		}
	}

	if hook, ok := d.(AfterSaveHook); ok {
		if err := hook.AfterSave(ctx, doc); err != nil {
			return err
		}
	}
	return nil
}
