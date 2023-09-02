# Go Mongoose driver

Mgs is a mongoose-like go mongodb odm. If you've used mongoose before or you're looking for a dev friendly mongodb odm for golang, then this is for you.

## Features

- Perform left join (lookup) on find operations without having to define aggregation pipelines for each query.
- Register hooks on predefined collection schemas for CRUD operations.
- Type safe schema & model validations.
- Atomic WRITE operations (documents are not written to database if one or more hooks return errors).

## Requirements

- Go >=1.18 (for generics)
- MongoDB >=4.2 (for atomic transactions)

## Installation

```bash
go get github.com/0x-buidl/mgs@latest
```

## License

This package is licensed under the [Apache License](LICENSE).
