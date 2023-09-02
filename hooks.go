package mgs

import (
	"context"
	"sync"
)

// HookArg represents arguments passed to a hook method. It is advisable to only modify [HookArg.Data] and not the Hook receiver itself, doing so may cause unexpected behaviors.
// To avoid modifying the reciever, ensure it's not a pointer.
type HookArg[T Schema] struct {
	data      interface{}
	operation QueryOperation
}

func newHookArg[T Schema](data interface{}, operation QueryOperation) *HookArg[T] {
	return &HookArg[T]{data, operation}
}

// Data returns the data associated with this hook.
func (arg *HookArg[T]) Data() interface{} {
	return arg.data
}

// Operation returns the [QueryOperation] being executed.
func (arg *HookArg[T]) Operation() QueryOperation {
	return arg.operation
}

// BeforeCreateHook runs before executing [Model.Create] and [Model.CreateMany] operations.
// [BeforeSaveHook] will run after this hook runs. [HookArg.Data] will return ptr to [Document](s) being created.
type BeforeCreateHook[T Schema] interface {
	BeforeCreate(ctx context.Context, arg *HookArg[T]) error
}

// AfterCreateHook runs after documents are written to the database when executing [Model.CreateOne] and [Model.CreateMany] operations.
// [AfterSaveHook] will run before this hook runs. [HookArg.Data] will return ptr to [Document](s) created.
type AfterCreateHook[T Schema] interface {
	AfterCreate(ctx context.Context, arg *HookArg[T]) error
}

// BeforeSaveHook runs before [Document](s) are written to the database when using [Model.CreateOne], [Model.CreateMany] or [Document.Save].
// This hook doesn't run on all [Model] Update operations. [HookArg.Data] will return ptr to [Document](s) being saved.
type BeforeSaveHook[T Schema] interface {
	BeforeSave(ctx context.Context, arg *HookArg[T]) error
}

// AfterSaveHook runs after [Document](s) are written to the database when using [Model.CreateOne], [Model.CreateMany] or [Document.Save].
// This hook doesn't run on all [Model] Update operations. [HookArg.Data] will return ptr to [Document](s) being saved.
type AfterSaveHook[T Schema] interface {
	AfterSave(ctx context.Context, arg *HookArg[T]) error
}

// BeforeDeleteHook runs before [Document](s) are removed from the database.
// [HookArg.Data] will return ptr to [Query] being executed if this is called on a [Model], otherwise it will return ptr to [Document] being deleted.
type BeforeDeleteHook[T Schema] interface {
	BeforeDelete(ctx context.Context, arg *HookArg[T]) error
}

// AfterDeleteHook runs after [Document](s) are removed from the database.
// [HookArg.Data] will return ptr to deleted [Document] if this is called on a [Document],
// otherwise it will return ptr to [mongo.DeleteResult].
type AfterDeleteHook[T Schema] interface {
	AfterDelete(ctx context.Context, arg *HookArg[T]) error
}

// BeforeFindHook runs before any find [QueryOperation] is executed.
// [HookArg.Data] will return ptr to [Query] being ececuted.
type BeforeFindHook[T Schema] interface {
	BeforeFind(ctx context.Context, arg *HookArg[T]) error
}

// AfterFindHook runs after a find operation is executed.
// [HookArg.Data] will return ptr to found [Document](s).
type AfterFindHook[T Schema] interface {
	AfterFind(ctx context.Context, arg *HookArg[T]) error
}

// BeforeUpdateHook runs before [Document](s) are updated in the database.
// This hook also runs on `replace` operations.
// [HookArg.Data] will return ptr to [Document] being updated if called on a [Document],
// otherwise it will return ptr to [Query].
type BeforeUpdateHook[T Schema] interface {
	BeforeUpdate(ctx context.Context, arg *HookArg[T]) error
}

// AfterUpdateHook runs after a document is updated in the database.
// This hook also runs on `replace` operations.
// [HookArg.Data] will return ptr to updated [*Document] if called on a [Document],
// otherwise it will return ptr to [mongo.UpdateResult].
type AfterUpdateHook[T Schema] interface {
	AfterUpdate(ctx context.Context, arg *HookArg[T]) error
}

// ValidateHook is used to register validation for a schema. This hook runs before a [Document] is saved to the database.
// It only runs on [Model.CreateOne], [Model.CreateMany] and [Document.Save] operations.
// [HookArg.Data] will return ptr to [Document] being validated.
type ValidateHook[T Schema] interface {
	Validate(ctx context.Context, arg *HookArg[T]) error
}

// BeforeValidateHook runs before validate hook is called. [HookArg.Data] will return ptr to [Document] being validated.
type BeforeValidateHook[T Schema] interface {
	BeforeValidate(ctx context.Context, arg *HookArg[T]) error
}

// AfterValidateHook runs after validate hook is called. [HookArg.Data] will return ptr to validated [Document].
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
