package mgs

import (
	"context"
	"fmt"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
)

type BeforeCreateHook[T Schema] interface {
	BeforeCreate(ctx context.Context, doc *Document[T]) error
}

type AfterCreateHook[T Schema] interface {
	AfterCreate(ctx context.Context, doc *Document[T]) error
}

type BeforeDeleteHook[T Schema] interface {
	BeforeDelete(ctx context.Context, doc *Document[T]) error
}

type AfterDeleteHook interface {
	AfterDelete(ctx context.Context, doc interface{}) error
}

type BeforeFindHook interface {
	BeforeFind(ctx context.Context, query bson.M) error
}

type AfterFindHook interface {
	AfterFind(ctx context.Context, doc interface{}) error
}

type BeforeSaveHook interface {
	BeforeSave(ctx context.Context, doc interface{}) error
}

type AfterSaveHook interface {
	AfterSave(ctx context.Context, doc interface{}) error
}

type BeforeUpdateHook interface {
	BeforeUpdate(ctx context.Context, doc interface{}) error
}

type AfterUpdateHook interface {
	AfterUpdate(ctx context.Context, doc interface{}) error
}

type hook[T Schema] func(ctx context.Context, doc *Document[T]) error

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
					hErr := runBeforeCreateHooks(ctx, newDoc)
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

func (model *Model[T]) bulkHookRun(ctx context.Context, docs []*Document[T], fn hook[T]) error {
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
					hErr := fn(ctx, doc)
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

func runBeforeCreateHooks[T Schema](ctx context.Context, doc *Document[T]) error {
	var d interface{} = doc.Doc
	if hook, ok := d.(BeforeCreateHook[T]); ok {
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

func runAfterCreateHooks[T Schema](ctx context.Context, doc *Document[T]) error {
	var d interface{} = doc.Doc
	if hook, ok := d.(AfterCreateHook[T]); ok {
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

func runBeforeDeleteHooks[T Schema](ctx context.Context, doc *Document[T]) error {
	var d interface{} = doc.Doc
	if hook, ok := d.(BeforeDeleteHook[T]); ok {
		if err := hook.BeforeDelete(ctx, doc); err != nil {
			return err
		}
	}
	return nil
}

func runAfterDeleteHooks[T Schema](ctx context.Context, doc *Document[T]) error {
	var d interface{} = doc.Doc
	if hook, ok := d.(AfterDeleteHook); ok {
		if err := hook.AfterDelete(ctx, doc); err != nil {
			return err
		}
	}
	return nil
}

func runBeforeFindHooks[T Schema](ctx context.Context, doc *Document[T], query bson.M) error {
	var d interface{} = doc.Doc
	if hook, ok := d.(BeforeFindHook); ok {
		return hook.BeforeFind(ctx, query)
	}
	return nil
}

func runAfterFindHooks[T Schema](ctx context.Context, doc *Document[T]) error {
	var d interface{} = doc.Doc
	if hook, ok := d.(AfterFindHook); ok {
		if err := hook.AfterFind(ctx, doc); err != nil {
			return err
		}
	}
	return nil
}

func runBeforeSaveHooks[T Schema](ctx context.Context, doc *Document[T]) error {
	var d interface{} = doc.Doc
	if hook, ok := d.(BeforeSaveHook); ok {
		if err := hook.BeforeSave(ctx, doc); err != nil {
			return err
		}
	}
	return nil
}

func runAfterSaveHooks[T Schema](ctx context.Context, doc *Document[T]) error {
	var d interface{} = doc.Doc
	if hook, ok := d.(AfterSaveHook); ok {
		if err := hook.AfterSave(ctx, doc); err != nil {
			return err
		}
	}
	return nil
}

func runBeforeUpdateHooks[T Schema](ctx context.Context, doc *Document[T]) error {
	var d interface{} = doc.Doc
	if hook, ok := d.(BeforeUpdateHook); ok {
		if err := hook.BeforeUpdate(ctx, doc); err != nil {
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

func runAfterUpdateHooks[T Schema](ctx context.Context, doc *Document[T]) error {
	var d interface{} = doc.Doc
	if hook, ok := d.(AfterUpdateHook); ok {
		if err := hook.AfterUpdate(ctx, doc); err != nil {
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
