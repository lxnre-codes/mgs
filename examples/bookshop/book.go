package bookshop

import (
	"time"

	mgs "github.com/0x-buidl/go-mongoose"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Book struct {
	Title     string             `json:"title"  bson:"title"`
	Author    primitive.ObjectID `json:"author" bson:"author"`
	Price     float64            `json:"price"  bson:"price"`
	Deleted   bool               `json:"-"      bson:"deleted"`
	DeletedAt *time.Time         `json:"-"      bson:"deletedAt"`
}

func NewBookModel(coll *mongo.Collection) *mgs.Model[Book] {
	return mgs.NewModel[Book](coll)
}
