package mgs

import (
	"context"
	"fmt"
	"sync"
)

// HookArg is the arguments passed to a hook function.
type HookArg[T Schema] struct {
	data      interface{}
	operation QueryOperation
	// single    bool
}

func newHookArg[T Schema](data interface{}, operation QueryOperation) *HookArg[T] {
	return &HookArg[T]{data, operation}
}

// Data returns the data passed to the hook.
func (arg *HookArg[T]) Data() interface{} {
	return arg.data
}

// // Single returns true if the hook is being called on a single operation.
// func (arg *HookArg[T]) Single() bool {
// 	return arg.single
// }

// Operation returns the operation being executed.
func (arg *HookArg[T]) Operation() QueryOperation {
	return arg.operation
}

// BeforeCreateHook runs before executing [Model.Create] and [Model.CreateMany] operations.
// [BeforeSaveHook] will run after this hook runs.
// [HookArg.Data] will return pointer to documents being created
type BeforeCreateHook[T Schema] interface {
	BeforeCreate(ctx context.Context, arg *HookArg[T]) error
}

// AfterCreateHook runs after executing [Model.Create] and [Model.CreateMany] operations.
// [AfterSaveHook] will run before this hook runs.
// Use [options.Hook.SetDisabledHooks] to disable [AfterSaveHook].
// [HookArg.Data] will return the documents created
type AfterCreateHook[T Schema] interface {
	AfterCreate(ctx context.Context, arg *HookArg[T]) error
}

// BeforeSaveHook runs before a document is written to the database when using
// [Model.CreateOne], [Model.CreateMany] or [Document.Save].
// This hook doesn't run on all [Model] Update operations.
// To check if the document being saved is new, use the Document.IsNew() method.
// [HookArg.Data] will return [*Document] being saved.
type BeforeSaveHook[T Schema] interface {
	BeforeSave(ctx context.Context, arg *HookArg[T]) error
}

// AfterSaveHook runs after a document is written to the database when using
// [Model.CreateOne], [Model.CreateMany] or [Document.Save].
// This hook doesn't run on all [Model] Update operations.
// Document.IsNew() will always return false in this hook.
// [HookArg.Data] will return the saved document.
type AfterSaveHook[T Schema] interface {
	AfterSave(ctx context.Context, arg *HookArg[T]) error
}

// BeforeDeleteHook runs before a document is removed from the database.
// [HookArg.Data] will return the [Query] being executed if this is called on a [Model],
// otherwise it will return the [*Document] being deleted.
type BeforeDeleteHook[T Schema] interface {
	BeforeDelete(ctx context.Context, arg *HookArg[T]) error
}

// AfterDeleteHook runs after a document is removed from the database.
// [HookArg.Data] will return [*mongo.DeleteResult] if this is called on a [Model],
// otherwise it will return the deleted [*Document].
type AfterDeleteHook[T Schema] interface {
	AfterDelete(ctx context.Context, arg *HookArg[T]) error
}

// BeforeFindHook runs before any find operation is executed.
// [HookArg.Data] will allways return the query being executed [*Query].
type BeforeFindHook[T Schema] interface {
	BeforeFind(ctx context.Context, arg *HookArg[T]) error
}

// AfterFindHook runs after a find operation is executed.
// [HookArg.Data] will return the found [*Document].
type AfterFindHook[T Schema] interface {
	AfterFind(ctx context.Context, arg *HookArg[T]) error
}

// BeforeUpdateHook runs before a document is updated in the database.
// This hook also runs on `replace` operations.
// [HookArg.Data] will return the [*Query] being executed.
type BeforeUpdateHook[T Schema] interface {
	BeforeUpdate(ctx context.Context, arg *HookArg[T]) error
}

// AfterUpdateHook runs after a document is updated in the database.
// This hook also runs on `replace` operations.
// [HookArg.Data] will return [*Document] if the operation is a `find-and-update` operation,
// otherwise it will return [*mongo.UpdateResult].
type AfterUpdateHook[T Schema] interface {
	AfterUpdate(ctx context.Context, arg *HookArg[T]) error
}

// ValidateHook is used to register a validation for a schema.
// This hook runs before a document is saved to the database.
// It only runs on [Model.CreateOne], [Model.CreateMany], [Document.Save] and [Document.Update] operations.
// [HookArg.Data] will return the [*Document] being validated.
type ValidateHook[T Schema] interface {
	Validate(ctx context.Context, arg *HookArg[T]) error
}

// BeforeValidateHook runs before validate hook is called.
// [HookArg.Data] will return the [*Document] being validated.
type BeforeValidateHook[T Schema] interface {
	BeforeValidate(ctx context.Context, arg *HookArg[T]) error
}

// AfterValidateHook runs after validate hook is called.
// [HookArg.Data] will return the validated [*Document].
type AfterValidateHook[T Schema] interface {
	AfterValidate(ctx context.Context, arg *HookArg[T]) error
}

type hook[T Schema] func(ctx context.Context, d any, arg *HookArg[T]) error

func (model *Model[T]) beforeCreateMany(
	ctx context.Context,
	docs []T,
) ([]*Document[T], []any, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	newDocs := make([]*Document[T], len(docs))
	docsToInsert := make([]interface{}, len(docs))

	var wg sync.WaitGroup
	var mu sync.Mutex
	var err error

	for i, doc := range docs {
		wg.Add(1)
		go func(i int, doc T) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					newDoc := model.NewDocument(doc)
					arg := newHookArg[T](newDoc, CreateQuery)
					hErr := runBeforeCreateHooks(ctx, newDoc.Doc, arg)
					if hErr != nil {
						cancel()
						mu.Lock()
						err = fmt.Errorf("error running before create hooks at index %d : %w",
							i, hErr)
						mu.Unlock()
						return
					}
					mu.Lock()
					newDocs[i] = newDoc
					docsToInsert[i] = newDoc
					mu.Unlock()
					return
				}
			}
		}(i, doc)
	}
	wg.Wait()
	if err != nil {
		return nil, nil, err
	}

	return newDocs, docsToInsert, nil
}

func (model *Model[T]) afterCreateMany(ctx context.Context, docs []*Document[T]) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var wg sync.WaitGroup
	var mu sync.Mutex
	var err error

	for _, doc := range docs {
		doc.isNew = false
		doc.collection = model.collection
		wg.Add(1)
		go func(doc *Document[T]) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					arg := newHookArg[T](doc, CreateQuery)
					hErr := runAfterCreateHooks(ctx, doc.Doc, arg)
					if hErr != nil {
						cancel()
						mu.Lock()
						err = hErr
						mu.Unlock()
						return
					}
					return
				}
			}
		}(doc)
	}
	wg.Wait()
	return err
}

func runBeforeCreateHooks[T Schema](ctx context.Context, d any, arg *HookArg[T]) error {
	if err := runValidateHooks(ctx, d, arg); err != nil {
		return err
	}

	if hook, ok := d.(BeforeCreateHook[T]); ok {
		if err := hook.BeforeCreate(ctx, arg); err != nil {
			return err
		}
	}

	if err := runBeforeSaveHooks(ctx, d, arg); err != nil {
		return err
	}

	return nil
}

func runAfterCreateHooks[T Schema](ctx context.Context, d any, arg *HookArg[T]) error {
	if err := runAfterSaveHooks(ctx, d, arg); err != nil {
		return err
	}

	if hook, ok := d.(AfterCreateHook[T]); ok {
		if err := hook.AfterCreate(ctx, arg); err != nil {
			return err
		}
	}

	return nil
}

func runBeforeDeleteHooks[T Schema](ctx context.Context, d any, arg *HookArg[T]) error {
	if hook, ok := d.(BeforeDeleteHook[T]); ok {
		if err := hook.BeforeDelete(ctx, arg); err != nil {
			return err
		}
	}
	return nil
}

func runAfterDeleteHooks[T Schema](ctx context.Context, d any, arg *HookArg[T]) error {
	if hook, ok := d.(AfterDeleteHook[T]); ok {
		if err := hook.AfterDelete(ctx, arg); err != nil {
			return err
		}
	}
	return nil
}

func runBeforeFindHooks[T Schema](ctx context.Context, d any, arg *HookArg[T]) error {
	if hook, ok := d.(BeforeFindHook[T]); ok {
		return hook.BeforeFind(ctx, arg)
	}
	return nil
}

func runAfterFindHooks[T Schema](ctx context.Context, d any, arg *HookArg[T]) error {
	if hook, ok := d.(AfterFindHook[T]); ok {
		if err := hook.AfterFind(ctx, arg); err != nil {
			return err
		}
	}
	return nil
}

func runBeforeSaveHooks[T Schema](ctx context.Context, d any, arg *HookArg[T]) error {
	if hook, ok := d.(BeforeSaveHook[T]); ok {
		if err := hook.BeforeSave(ctx, arg); err != nil {
			return err
		}
	}
	return nil
}

func runAfterSaveHooks[T Schema](ctx context.Context, d any, arg *HookArg[T]) error {
	if hook, ok := d.(AfterSaveHook[T]); ok {
		if err := hook.AfterSave(ctx, arg); err != nil {
			return err
		}
	}
	return nil
}

func runBeforeUpdateHooks[T Schema](ctx context.Context, d any, arg *HookArg[T]) error {
	if hook, ok := d.(BeforeUpdateHook[T]); ok {
		if err := hook.BeforeUpdate(ctx, arg); err != nil {
			return err
		}
	}

	if hook, ok := d.(BeforeSaveHook[T]); ok {
		if err := hook.BeforeSave(ctx, arg); err != nil {
			return err
		}
	}
	return nil
}

func runAfterUpdateHooks[T Schema](ctx context.Context, d any, arg *HookArg[T]) error {
	if hook, ok := d.(AfterUpdateHook[T]); ok {
		if err := hook.AfterUpdate(ctx, arg); err != nil {
			return err
		}
	}

	if hook, ok := d.(AfterSaveHook[T]); ok {
		if err := hook.AfterSave(ctx, arg); err != nil {
			return err
		}
	}
	return nil
}

func runValidateHooks[T Schema](ctx context.Context, d any, arg *HookArg[T]) error {
	if validator, ok := d.(ValidateHook[T]); ok {
		if hook, ok := d.(BeforeValidateHook[T]); ok {
			if err := hook.BeforeValidate(ctx, arg); err != nil {
				return err
			}
		}

		if err := validator.Validate(ctx, arg); err != nil {
			return err
		}

		if hook, ok := d.(AfterValidateHook[T]); ok {
			if err := hook.AfterValidate(ctx, arg); err != nil {
				return err
			}
		}
	}
	return nil
}
