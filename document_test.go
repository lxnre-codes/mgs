package mgs_test

import (
	"context"
	"testing"

	"github.com/0x-buidl/mgs"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
)

func TestSaveDocument(t *testing.T) {
	ctx := context.Background()
	db, cleanup := getDb(ctx)
	defer cleanup(ctx)

	bookModel := mgs.NewModel[Book, *mgs.DefaultSchema](db.Collection("books"))
	generateBooks(ctx, db)

	t.Run("Should save on new document", func(t *testing.T) {
		doc := bookModel.NewDocument(Book{Title: "The Lord of the Rings"})
		err := doc.Save(ctx)
		assert.NoError(t, err)
	})

	t.Run("Should save on existing document", func(t *testing.T) {
		doc, err := bookModel.FindOne(ctx, bson.M{})
		assert.NoError(t, err)
		assert.NotNil(t, doc)

		doc.Doc.Title = "The Lord of the Rings: The Fellowship of the Ring"
		err = doc.Save(ctx)
		assert.NoError(t, err)

		doc, err = bookModel.FindById(ctx, doc.GetID())
		assert.NoError(t, err)
		assert.Equal(t, "The Lord of the Rings: The Fellowship of the Ring", doc.Doc.Title)
	})
}
