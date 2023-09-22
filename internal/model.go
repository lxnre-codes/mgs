package internal

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func WithTransaction[T interface {
	*mongo.Database | *mongo.Collection | *mongo.SessionContext | *mongo.Client | *mongo.Session
}](ctx context.Context, sess T, fn func(ctx mongo.SessionContext) (any, error), opts ...*options.TransactionOptions,
) (any, error) {
	var session mongo.Session
	var err error
	switch sess := any(sess).(type) {
	case *mongo.SessionContext:
		return (*sess).WithTransaction(ctx, fn, opts...)
	case *mongo.Session:
		return (*sess).WithTransaction(ctx, fn, opts...)
	case *mongo.Client:
		session, err = sess.StartSession()
	case *mongo.Database:
		session, err = sess.Client().StartSession()
	case *mongo.Collection:
		session, err = sess.Database().Client().StartSession()
	}

	if err != nil {
		return nil, err
	}
	defer session.EndSession(ctx)

	return session.WithTransaction(ctx, fn, opts...)
}
