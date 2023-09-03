package database

import (
	"context"
	"time"

	"github.com/0x-buidl/mgs"
	"go.mongodb.org/mongo-driver/mongo"
)

type Author struct {
	Name      string     `json:"name" bson:"name"`
	Deleted   bool       `json:"-"    bson:"deleted"`
	DeletedAt *time.Time `json:"-"    bson:"deletedAt"`
}

type (
	AuthorModel = mgs.Model[Author, *mgs.DefaultSchema]
	AuthorDoc   = mgs.Document[Author, *mgs.DefaultSchema]
)

func NewAuthorModel(coll *mongo.Collection) *AuthorModel {
	return mgs.NewModel[Author, *mgs.DefaultSchema](coll)
}

func (a Author) Validate(ctx context.Context, arg *mgs.HookArg[Author]) error {
	return nil
}
