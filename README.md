# Mgs

![CodeQL](https://github.com/0x-buidl/mgs/actions/workflows/codeql.yml/badge.svg)
![Build & Tests](https://github.com/0x-buidl/mgs/actions/workflows/build.yml/badge.svg)
[![GitHub release (with filter)](https://img.shields.io/github/v/release/0x-buidl/mgs)](https://github.com/0x-buidl/mgs/releases)
[![Go MongoDB](https://img.shields.io/badge/mongodb-driver-g?logo=mongodb)](https://pkg.go.dev/go.mongodb.org/mongo-driver/mongo)
[![Go Report Card](https://goreportcard.com/badge/github.com/0x-buidl/mgs)](https://goreportcard.com/report/github.com/0x-buidl/mgs)
[![Go Reference](https://pkg.go.dev/badge/github.com/0x-buidl/mgs.svg)](https://pkg.go.dev/github.com/0x-buidl/mgs)
[![Coverage Status](https://coveralls.io/repos/github/0x-buidl/mgs/badge.svg)](https://coveralls.io/github/0x-buidl/mgs)

Mgs is a mongoose-like [go mongodb](https://github.com/mongodb/mongo-go-driver) odm. If you've used [mongoose.js](https://mongoosejs.com), or you're looking for a dev friendly [mongodb](https://www.mongodb.com) odm for golang, then this is for you.

---

## Features

- Perform left join (lookup) on find operations without having to define aggregation pipelines for each query.
- Register hooks on predefined collection schemas for CRUD operations.
- Type safe schema & model validations.
- Atomic WRITE operations (documents are not written to database if one or more hooks return errors).

---

## Requirements

- Go >=1.18 (for generics)
- MongoDB >=4.4 (for atomic transactions)

---

## Installation

```bash
go get github.com/0x-buidl/mgs@latest
```

---

## Usage

To get started, establish a connection to mongodb client like normal.

```go
import (
    "context"
    "time"

    "github.com/0x-buidl/mgs"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    "go.mongodb.org/mongo-driver/mongo/readpref"
)

ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
if err != nil {
	log.Fatal(err)
}

defer func() {
    if err = client.Disconnect(ctx); err != nil {
        panic(err)
    }
}()

err = client.Ping(ctx, nil)
if err != nil {
	log.Fatal(err)
}

db := client.Database("test")

coll := db.Collection("test_coll")
```

Define your collection schema and hooks.

**NOTE**: Do not modify hook receivers, doing so may lead to unexpected behaviours. To avoid errors, ensure hook receivers are not pointers.

```go
type Book struct {
	Title  string      `json:"title"  bson:"title"`
	Author interface{} `json:"author" bson:"author"`
}

// For simplicity. You can also define your custom default schema that implements mgs.IDefaultSchema
type BookModel = mgs.Model[Book,*mgs.DefaultSchema]
type BookDoc = mgs.Document[Book]

func NewBookModel(coll *mongo.Collection) *BookModel {
    return mgs.NewModel[Book,*mgs.DefaultSchema](coll)
}

func (book Book) Validate(ctx context.Context, arg *mgs.HookArg[Book]) error {
	return nil
}

func (book Book) BeforeValidate(ctx context.Context, arg *mgs.HookArg[Book]) error {
	return nil
}

func (book Book) AfterValidate(ctx context.Context, arg *mgs.HookArg[Book]) error {
	return nil
}

func (book Book) BeforeCreate(ctx context.Context, arg *mgs.HookArg[Book]) error {
	return nil
}

func (book Book) AfterCreate(ctx context.Context, arg *mgs.HookArg[Book]) error {
	return nil
}

func (book Book) BeforeUpdate(ctx context.Context, arg *mgs.HookArg[Book]) error {
	return nil
}

func (book Book) AfterUpdate(ctx context.Context, arg *mgs.HookArg[Book]) error {
	return nil
}

func (book Book) BeforeDelete(ctx context.Context, arg *mgs.HookArg[Book]) error {
	return nil
}

func (book Book) AfterDelete(ctx context.Context, arg *mgs.HookArg[Book]) error {
	return nil
}

func (book Book) BeforeFind(ctx context.Context, arg *mgs.HookArg[Book]) error {
	return nil
}

func (book Book) AfterFind(ctx context.Context, arg *mgs.HookArg[Book]) error {
	return nil
}

```

Additional examples and usage guides can be found under the [examples](examples) directory and [mgs go docs](https://pkg.go.dev/github.com/0x-buidl/mgs).

## License

This package is licensed under the [Apache License](LICENSE).
