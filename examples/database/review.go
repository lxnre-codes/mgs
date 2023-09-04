package database

import (
	"context"

	"github.com/0x-buidl/mgs"
	"go.mongodb.org/mongo-driver/mongo"
)

type Review struct {
	// ObjectID or Book object
	Book    interface{} `bson:"book"    json:"book"    validate:"required"`
	Comment string      `bson:"comment" json:"comment" validate:"required"`
	Rating  int         `bson:"rating"  json:"rating"  validate:"required"`
}

type (
	ReviewModel = mgs.Model[Review, *mgs.DefaultSchema]
	ReviewDoc   = mgs.Document[Review, *mgs.DefaultSchema]
)

func NewReviewModel(coll *mongo.Collection) *ReviewModel {
	return mgs.NewModel[Review, *mgs.DefaultSchema](coll)
}

func (Review) Validate(ctx context.Context, arg *mgs.HookArg[Review]) error {
	return nil
}
