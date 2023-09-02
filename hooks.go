package mgs

import (
	"context"
	"sync"
)

// HookArg is the arguments passed to a hook function.
// It is advisable to only modify [HookArg.Data] and not the Hook receiver itself, doing so may cause unexpected behavior.
// To avoid this, the hook receiver should not bo a pointer.
type HookArg[T Schema] struct {
	data      interface{}
	operation QueryOperation
}

func newHookArg[T Schema](data interface{}, operation QueryOperation) *HookArg[T] {
	return &HookArg[T]{data, operation}
}

// Data returns the data passed to the hook.
func (arg *HookArg[T]) Data() interface{} {
	return arg.data
}

// Operation returns the operation being executed.
func (arg *HookArg[T]) Operation() QueryOperation {
	return arg.operation
}

// BeforeCreateHook runs before executing [Model.Create] and [Model.CreateMany] operations.
// [BeforeSaveHook] will run after this hook runs.
// [HookArg.Data] will return pointer to document(s) being created.
type BeforeCreateHook[T Schema] interface {
	BeforeCreate(ctx context.Context, arg *HookArg[T]) error
}

// AfterCreateHook runs after executing [Model.Create] and [Model.CreateMany] operations.
// [AfterSaveHook] will run before this hook runs.
// Use [options.Hook.SetDisabledHooks] to disable [AfterSaveHook].
// [HookArg.Data] will return the document(s) created.
type AfterCreateHook[T Schema] interface {
	AfterCreate(ctx context.Context, arg *HookArg[T]) error
}

// BeforeSaveHook runs before document(s) are written to the database when using
// [Model.CreateOne], [Model.CreateMany] or [Document.Save].
// This hook doesn't run on all [Model] Update operations.
// To check if the document being saved is new, use the [Document.IsNew] method.
// [HookArg.Data] will return [*Document](s) being saved.
type BeforeSaveHook[T Schema] interface {
	BeforeSave(ctx context.Context, arg *HookArg[T]) error
}

// AfterSaveHook runs after document(s) are written to the database when using
// [Model.CreateOne], [Model.CreateMany] or [Document.Save].
// This hook doesn't run on all [Model] Update operations.
// [Document.IsNew] will always return false in this hook.
// [HookArg.Data] will return the saved document(s).
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
// [HookArg.Data] will return the found document(s) [*Document].
type AfterFindHook[T Schema] interface {
	AfterFind(ctx context.Context, arg *HookArg[T]) error
}

// BeforeUpdateHook runs before a document is updated in the database.
// This hook also runs on `replace` operations.
// [HookArg.Data] will return the [*Query] being executed if called on [*Model],
// otherwise it will return [*Document] being updated.
type BeforeUpdateHook[T Schema] interface {
	BeforeUpdate(ctx context.Context, arg *HookArg[T]) error
}

// AfterUpdateHook runs after a document is updated in the database.
// This hook also runs on `replace` operations.
// [HookArg.Data] will return [*mongo.UpdateResult] if called on [*Model],
// otherwise it will return the updated [*Document].
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

type hook[T Schema, P IDefaultSchema] func(ctx context.Context, d *Document[T, P], arg *HookArg[T]) error

func (model *Model[T, P]) beforeCreateMany(
	ctx context.Context,
	docs []T,
) ([]*Document[T, P], []any, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	newDocs := make([]*Document[T, P], len(docs))
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
					arg := newHookArg[T](newDoc, CreateMany)
					hErr := runValidateHooks(ctx, newDoc, arg)
					if hErr != nil {
						cancel()
						mu.Lock()
						err = hErr
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

	ds := model.docSample()
	err = runBeforeCreateHooks(ctx, ds, newHookArg[T](&newDocs, CreateMany))
	if err != nil {
		return nil, nil, err
	}

	return newDocs, docsToInsert, nil
}

func runBeforeCreateHooks[T Schema, P IDefaultSchema](
	ctx context.Context,
	d *Document[T, P],
	arg *HookArg[T],
) error {
	intf := any(d.Doc)
	if hook, ok := intf.(BeforeCreateHook[T]); ok {
		if err := hook.BeforeCreate(ctx, arg); err != nil {
			return err
		}
	}

	if err := runBeforeSaveHooks(ctx, d, arg); err != nil {
		return err
	}

	return nil
}

func runAfterCreateHooks[T Schema, P IDefaultSchema](
	ctx context.Context,
	d *Document[T, P],
	arg *HookArg[T],
) error {
	if err := runAfterSaveHooks(ctx, d, arg); err != nil {
		return err
	}

	intf := any(d.Doc)
	if hook, ok := intf.(AfterCreateHook[T]); ok {
		if err := hook.AfterCreate(ctx, arg); err != nil {
			return err
		}
	}

	return nil
}

func runBeforeDeleteHooks[T Schema, P IDefaultSchema](
	ctx context.Context,
	d *Document[T, P],
	arg *HookArg[T],
) error {
	intf := any(d.Doc)
	if hook, ok := intf.(BeforeDeleteHook[T]); ok {
		if err := hook.BeforeDelete(ctx, arg); err != nil {
			return err
		}
	}
	return nil
}

func runAfterDeleteHooks[T Schema, P IDefaultSchema](
	ctx context.Context,
	d *Document[T, P],
	arg *HookArg[T],
) error {
	intf := any(d.Doc)
	if hook, ok := intf.(AfterDeleteHook[T]); ok {
		if err := hook.AfterDelete(ctx, arg); err != nil {
			return err
		}
	}
	return nil
}

func runBeforeFindHooks[T Schema, P IDefaultSchema](
	ctx context.Context,
	d *Document[T, P],
	arg *HookArg[T],
) error {
	intf := any(d.Doc)
	if hook, ok := intf.(BeforeFindHook[T]); ok {
		return hook.BeforeFind(ctx, arg)
	}
	return nil
}

func runAfterFindHooks[T Schema, P IDefaultSchema](
	ctx context.Context,
	d *Document[T, P],
	arg *HookArg[T],
) error {
	intf := any(d.Doc)
	if hook, ok := intf.(AfterFindHook[T]); ok {
		if err := hook.AfterFind(ctx, arg); err != nil {
			return err
		}
	}
	return nil
}

func runBeforeSaveHooks[T Schema, P IDefaultSchema](
	ctx context.Context,
	d *Document[T, P],
	arg *HookArg[T],
) error {
	intf := any(d.Doc)
	if hook, ok := intf.(BeforeSaveHook[T]); ok {
		if err := hook.BeforeSave(ctx, arg); err != nil {
			return err
		}
	}
	return nil
}

func runAfterSaveHooks[T Schema, P IDefaultSchema](
	ctx context.Context,
	d *Document[T, P],
	arg *HookArg[T],
) error {
	intf := any(d.Doc)
	if hook, ok := intf.(AfterSaveHook[T]); ok {
		if err := hook.AfterSave(ctx, arg); err != nil {
			return err
		}
	}
	return nil
}

func runBeforeUpdateHooks[T Schema, P IDefaultSchema](
	ctx context.Context,
	d *Document[T, P],
	arg *HookArg[T],
) error {
	intf := any(d.Doc)
	if hook, ok := intf.(BeforeUpdateHook[T]); ok {
		if err := hook.BeforeUpdate(ctx, arg); err != nil {
			return err
		}
	}

	return nil
}

func runAfterUpdateHooks[T Schema, P IDefaultSchema](
	ctx context.Context,
	d *Document[T, P],
	arg *HookArg[T],
) error {
	intf := any(d.Doc)
	if hook, ok := intf.(AfterUpdateHook[T]); ok {
		if err := hook.AfterUpdate(ctx, arg); err != nil {
			return err
		}
	}

	return nil
}

func runValidateHooks[T Schema, P IDefaultSchema](
	ctx context.Context,
	d *Document[T, P],
	arg *HookArg[T],
) error {
	intf := any(d.Doc)
	if validator, ok := intf.(ValidateHook[T]); ok {
		if hook, ok := intf.(BeforeValidateHook[T]); ok {
			if err := hook.BeforeValidate(ctx, arg); err != nil {
				return err
			}
		}

		if err := validator.Validate(ctx, arg); err != nil {
			return err
		}

		if hook, ok := intf.(AfterValidateHook[T]); ok {
			if err := hook.AfterValidate(ctx, arg); err != nil {
				return err
			}
		}
	}
	return nil
}
