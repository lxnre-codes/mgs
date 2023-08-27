package options

import "go.mongodb.org/mongo-driver/mongo/options"

// FindOneOptions represents options that can be used to configure a FindOne operation.
type FindOneOptions struct {
	*options.FindOneOptions
	*QueryOptions
	*HookOptions
}

// FindOne returns a new FindOneOptions.
func FindOne() *FindOneOptions {
	return &FindOneOptions{options.FindOne(), Query(), Hook()}
}

// MergeFindOneOptions combines the given FindOneOptions instances into a single FindOneOptions in a last-one-wins fashion.
// Deprecated: Use single struct options.
func MergeFindOneOptions(opts ...*FindOneOptions) (*QueryOptions, *options.FindOneOptions) {
	fopts := make([]*options.FindOneOptions, 0, len(opts))
	qopts := make([]*QueryOptions, 0, len(opts))
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if opt.FindOneOptions != nil {
			fopts = append(fopts, opt.FindOneOptions)
		}
		if opt.QueryOptions != nil {
			qopts = append(qopts, opt.QueryOptions)
		}
	}
	fo, qo := options.MergeFindOneOptions(fopts...), MergeQueryOptions(qopts...)
	return qo, fo
}

// FindOptions represents options that can be used to configure a Find operation.
type FindOptions struct {
	*options.FindOptions
	*QueryOptions
	*HookOptions
}

// Find returns a new FindOptions.
func Find() *FindOptions {
	return &FindOptions{options.Find(), Query(), Hook()}
}

// MergeFindOptions combines the given FindOptions instances into a single FindOptions in a last-one-wins fashion.
// Deprecated: Use single struct options.
func MergeFindOptions(opts ...*FindOptions) (*QueryOptions, *options.FindOptions) {
	fopts := make([]*options.FindOptions, 0, len(opts))
	qopts := make([]*QueryOptions, 0, len(opts))
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if opt.FindOptions != nil {
			fopts = append(fopts, opt.FindOptions)
		}
		if opt.QueryOptions != nil {
			qopts = append(qopts, opt.QueryOptions)
		}
	}

	fo, qo := options.MergeFindOptions(fopts...), MergeQueryOptions(qopts...)
	return qo, fo
}
