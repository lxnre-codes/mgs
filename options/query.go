package options

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// PopulateOptions represents the options for populating a path on a document.
type PopulateOptions struct {
	// Path to populate.
	Path *string
	// Optional query conditions to match.
	Match *bson.M
	// Model to use for population.
	Collection *string
	// Optional query options. Only sort, skip, limit and projections are used.
	Options *options.FindOptions
	// Paths to populate on the populated doc.
	Populate *[]*PopulateOptions
	// Schema the populated doc is associated with, default is bson.M.
	Schema interface{}
	// Override the schema-level `LocalField` option for this individual `populate()` call.
	LocalField *string
	// Override the schema-level `ForeignField` option for this individual `populate()` call.
	// Default: "_id"
	ForeignField *string
	// If true `path` is to a document, or `nil` if no document was found.
	// If false `path` is set to an array, which will be empty if no documents are found.
	// Default: true
	OnlyOne *bool
	// StrictPopulate *bool
}

// Populate returns a new PopulateOptions with `OnlyOne` set to `true`.
func Populate() *PopulateOptions {
	onlyOne := true
	schema := bson.M{}
	ff := "_id"
	return &PopulateOptions{OnlyOne: &onlyOne, ForeignField: &ff, Schema: schema}
}

// SetPath sets the `Path` option.
func (pop *PopulateOptions) SetPath(path string) *PopulateOptions {
	pop.Path = &path
	return pop
}

// SetMatch sets the `Match` option.
func (pop *PopulateOptions) SetMatch(match bson.M) *PopulateOptions {
	pop.Match = &match
	return pop
}

// SetCollection sets the `Collection` option.
func (pop *PopulateOptions) SetCollection(coll string) *PopulateOptions {
	pop.Collection = &coll
	return pop
}

// SetOptions sets the `Options` option.
func (pop *PopulateOptions) SetOptions(options *options.FindOptions) *PopulateOptions {
	pop.Options = options
	return pop
}

// SetPopulate sets the `Populate` option.
func (pop *PopulateOptions) SetPopulate(populate ...*PopulateOptions) *PopulateOptions {
	pop.Populate = &populate
	return pop
}

// SetSchema sets the `Schema` option.
func (pop *PopulateOptions) SetSchema(schema interface{}) *PopulateOptions {
	pop.Schema = schema
	return pop
}

// SetLocalField sets the `LocalField` option.
func (pop *PopulateOptions) SetLocalField(localField string) *PopulateOptions {
	pop.LocalField = &localField
	return pop
}

// SetForeignField sets the `ForeignField` option.
func (pop *PopulateOptions) SetForeignField(foreignField string) *PopulateOptions {
	pop.ForeignField = &foreignField
	return pop
}

// SetOnlyOne sets the `OnlyOne` option.
func (pop *PopulateOptions) SetOnlyOne(onlyOne bool) *PopulateOptions {
	pop.OnlyOne = &onlyOne
	return pop
}

type PopulateOption = []*PopulateOptions

// QueryOptions represents the options for querying documents.
type QueryOptions struct {
	*PopulateOption
}

// Query returns a new QueryOptions.
func Query() *QueryOptions {
	return &QueryOptions{}
}

// SetPopulate sets the `Populate` option.
func (qo *QueryOptions) SetPopulate(populate ...*PopulateOptions) *QueryOptions {
	qo.PopulateOption = &populate
	return qo
}

func MergeQueryOptions(opts ...*QueryOptions) *QueryOptions {
	qo := Query()
	for _, opt := range opts {
		qo.PopulateOption = opt.PopulateOption
	}
	return qo
}
