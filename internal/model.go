package internal

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SessFn func(sessCtx mongo.SessionContext) (interface{}, error)

func WithTransaction(ctx context.Context, coll *mongo.Collection, fn SessFn, opts ...*options.TransactionOptions) (interface{}, error) {
	if ctx, ok := ctx.(mongo.SessionContext); ok {
		return ctx.WithTransaction(ctx, fn, opts...)
	}
	session, err := coll.Database().Client().StartSession()
	if err != nil {
		return nil, err
	}
	defer session.EndSession(ctx)

	return session.WithTransaction(ctx, fn, opts...)
}
