package options

import (
	"go.mongodb.org/mongo-driver/mongo/options"
)

// InsertOneOptions represents the options for inserting a document.
type InsertOneOptions struct {
	*options.InsertOneOptions
	*QueryOptions
}

// InsertOne returns a new InsertOneOptions.
func InsertOne() *InsertOneOptions {
	return &InsertOneOptions{}
}

// MergeInsertOneOptions combines the given InsertOneOptions instances into a single InsertOneOptions in a last-one-wins fashion.
// Deprecated: Use single struct options.
func MergeInsertOneOptions(opts ...*InsertOneOptions) (*QueryOptions, *options.InsertOneOptions) {
	qopts := make([]*QueryOptions, 0, len(opts))
	iopts := make([]*options.InsertOneOptions, 0, len(opts))
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if opt.InsertOneOptions != nil {
			iopts = append(iopts, opt.InsertOneOptions)
		}
		if opt.QueryOptions != nil {
			qopts = append(qopts, opt.QueryOptions)
		}
	}
	io, qo := options.MergeInsertOneOptions(iopts...), MergeQueryOptions(qopts...)
	return qo, io
}

// InsertManyOptions represents the options for inserting many document.
type InsertManyOptions struct {
	*options.InsertManyOptions
	*QueryOptions
}

// InsertOne returns a new InsertOneOptions.
func InsertMany() *InsertManyOptions {
	return &InsertManyOptions{}
}

// MergeInsertManyOptions combines the given InsertManyOptions instances into a single InsertManyOptions in a last-one-wins fashion.
// Deprecated: Use single struct options.
func MergeInsertManyOptions(
	opts ...*InsertManyOptions,
) (*QueryOptions, *options.InsertManyOptions) {
	qopts := make([]*QueryOptions, 0, len(opts))
	iopts := make([]*options.InsertManyOptions, 0, len(opts))
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if opt.InsertManyOptions != nil {
			iopts = append(iopts, opt.InsertManyOptions)
		}
		if opt.QueryOptions != nil {
			qopts = append(qopts, opt.QueryOptions)
		}
	}
	io, qo := options.MergeInsertManyOptions(iopts...), MergeQueryOptions(qopts...)
	return qo, io
}
