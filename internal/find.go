package internal

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	mopt "github.com/0x-buidl/mgs/options"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UnionFindOpts interface {
	*mopt.FindOptions | *mopt.FindOneOptions
}

func BuildPopulatePipeline[P UnionFindOpts](d any, q bson.M, opt P) (mongo.Pipeline, *options.AggregateOptions, error) {
	pipelineOpts, aggrOpts, queryOpts := mergeFindOptsWithAggregatOpts(opt)
	pipeline := append(mongo.Pipeline{bson.D{{Key: "$match", Value: q}}}, pipelineOpts...)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var err error
	for _, pop := range *queryOpts.PopulateOption {
		wg.Add(1)
		func(pop *mopt.PopulateOptions) {
			defer wg.Done()
			mu.Lock()
			if err != nil {
				mu.Unlock()
				return
			}
			mu.Unlock()

			pipe, pErr := getPopulateStages(d, pop)

			mu.Lock()
			if pErr != nil {
				err = pErr
				mu.Unlock()
				return
			}
			pipeline = append(pipeline, pipe...)
			mu.Unlock()
		}(pop)
	}
	wg.Wait()

	if err != nil {
		return nil, nil, err
	}

	return pipeline, aggrOpts, nil
}

// TODO: custom populate errors

func getPopulateStages(doc any, opt *mopt.PopulateOptions) (mongo.Pipeline, error) {
	// set initial match stage
	lookupPipeline := mongo.Pipeline{
		bson.D{
			{Key: "$match", Value: bson.M{"$expr": bson.M{"$eq": bson.A{"$$localField", "$" + *opt.ForeignField}}}},
		},
	}

	// set custom match stage
	if opt.Match != nil {
		lookupPipeline = append(lookupPipeline, bson.D{{Key: "$match", Value: *opt.Match}})
	}

	// populated nested populations
	lookups := make([]bson.D, 0)
	if opt.Populate != nil {
		var wg sync.WaitGroup
		var mu sync.Mutex
		var err error
		for _, p := range *opt.Populate {
			wg.Add(1)
			go func(p *mopt.PopulateOptions) {
				defer wg.Done()
				mu.Lock()
				if err != nil {
					mu.Unlock()
					return
				}
				mu.Unlock()

				pipe, pErr := getPopulateStages(opt.Schema, p)

				mu.Lock()
				if pErr != nil {
					err = pErr
					mu.Unlock()
					return
				}
				lookups = append(lookups, pipe...)
				mu.Unlock()
			}(p)
		}
		wg.Wait()
	}

	// merge options into aggregate pipeline
	if opt.Options != nil {
		popt := opt.Options
		if popt.Sort != nil {
			lookupPipeline = append(lookupPipeline, bson.D{{Key: "$sort", Value: popt.Sort}})
		}
		if popt.Skip != nil {
			lookupPipeline = append(lookupPipeline, bson.D{{Key: "$skip", Value: popt.Skip}})
		}
		var limit int64 = 1
		if popt.Limit != nil && !*opt.OnlyOne {
			limit = *popt.Limit
		}
		lookupPipeline = append(lookupPipeline, bson.D{{Key: "$limit", Value: limit}})
		if len(lookups) > 0 {
			lookupPipeline = append(lookupPipeline, lookups...)
		}
		if popt.Projection != nil {
			lookupPipeline = append(
				lookupPipeline,
				bson.D{{Key: "$project", Value: popt.Projection}},
			)
		}
	} else {
		if *opt.OnlyOne {
			lookupPipeline = append(lookupPipeline, bson.D{{Key: "$limit", Value: 1}})
		}
		if len(lookups) > 0 {
			lookupPipeline = append(lookupPipeline, lookups...)
		}
	}

	lookup := bson.M{
		"from":     *opt.Collection,
		"let":      bson.M{"localField": "$" + *opt.Path},
		"pipeline": lookupPipeline,
		"as":       *opt.Path,
	}
	populatePipeline := mongo.Pipeline{}
	// declare value and type of document to populate
	v, t := reflect.ValueOf(doc), reflect.TypeOf(doc)

	// document must be a struct
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("document must be a struct to populate nested path")
	}

	paths := strings.Split(*opt.Path, ".")
	windPaths := make([]bson.D, 0)
	// traverse and unwind all slice fields
	for len(paths) > 0 {
		currPath := paths[0] // set current path
		// get all bson tags of the struct
		fields := getStructFields(t)

		// get the struct field of the path to populate
		field, ok := fields[currPath]
		if !ok {
			return nil, fmt.Errorf("field %s not found in struct", currPath)
		}

		// check if it's a pointer
		ft := field.Type
		if ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}

		switch {
		case ft.Kind() == reflect.Slice:
			// check if it's a pointer
			elemType := ft.Elem()
			if elemType.Kind() == reflect.Pointer {
				elemType = elemType.Elem()
			}

			// if path has more fields, type must be a struct
			if len(paths) > 1 && elemType.Kind() != reflect.Struct {
				return nil, fmt.Errorf("field %s must be a slice of struct", currPath)
			}
			// if the field is a slice, we need to unwind it
			populatePipeline = append(populatePipeline, bson.D{{Key: "$unwind", Value: "$" + currPath}})

			// declare the slice regroup by _id
			group := bson.E{Key: "$group", Value: bson.M{"_id": "$_id"}}

			// only group by _id if there are more paths to wind & current path is initial path.
			if len(paths) < len(strings.Split(*opt.Path, ".")) {
				group.Value.(bson.M)["_id"] = nil
			}

			for k := range fields {
				// key is equal to current path, push the value
				if k == currPath {
					group.Value.(bson.M)[k] = bson.M{"$push": "$" + k}
				} else {
					// otherwise, select the first value
					group.Value.(bson.M)[k] = bson.M{"$first": "$" + k}
				}
			}
			// wind paths in descending order
			windPaths = append([]bson.D{{group}}, windPaths...)
			t = elemType
		case ft.Kind() == reflect.Struct:
			t = ft
		default:
			if len(paths) > 1 {
				return nil, fmt.Errorf("field %s must be a struct or slice of struct", currPath)
			}
		}
		paths = paths[1:] // trim off first key

	}

	populatePipeline = append(populatePipeline, bson.D{{Key: "$lookup", Value: lookup}})
	if *opt.OnlyOne {
		populatePipeline = append(
			populatePipeline,
			bson.D{
				{Key: "$unwind", Value: bson.M{"path": "$" + *opt.Path, "preserveNullAndEmptyArrays": true}},
			},
		)
	}
	populatePipeline = append(populatePipeline, windPaths...)
	return populatePipeline, nil
}

// TODO: merge these options with aggregate options if possible
// if opt.AllowPartialResults != nil {
// }
// if opt.CursorType != nil {
// }
// if opt.Max != nil {
// }
// if opt.Min != nil {
// }
// if opt.NoCursorTimeout != nil {
// }
// if opt.OplogReplay != nil {
// }
// if opt.ReturnKey != nil {
// }
// if opt.ShowRecordID != nil {
// }
// if opt.Snapshot != nil {
// }

func mergeFindOptsWithAggregatOpts[T UnionFindOpts](opt T) (mongo.Pipeline, *options.AggregateOptions, *mopt.QueryOptions) {
	aggOpts, pipelineOpts, queryOpts := options.Aggregate(), mongo.Pipeline{}, mopt.Query()
	switch opt := any(opt).(type) {
	case *mopt.FindOptions:
		if opt.AllowDiskUse != nil {
			aggOpts.SetAllowDiskUse(*opt.AllowDiskUse)
		}
		if opt.BatchSize != nil {
			aggOpts.SetBatchSize(*opt.BatchSize)
		}
		if opt.Collation != nil {
			aggOpts.SetCollation(opt.Collation)
		}
		if opt.Comment != nil {
			aggOpts.SetComment(*opt.Comment)
		}
		if opt.Hint != nil {
			aggOpts.SetHint(opt.Hint)
		}
		if opt.MaxAwaitTime != nil {
			aggOpts.SetMaxAwaitTime(*opt.MaxAwaitTime)
		}
		if opt.MaxTime != nil {
			aggOpts.SetMaxTime(*opt.MaxTime)
		}
		if opt.Let != nil {
			aggOpts.SetLet(opt.Let)
		}
		if opt.Sort != nil {
			pipelineOpts = append(pipelineOpts, bson.D{{Key: "$sort", Value: opt.Sort}})
		}
		if opt.Limit != nil {
			pipelineOpts = append(pipelineOpts, bson.D{{Key: "$limit", Value: *opt.Limit}})
		}
		if opt.Skip != nil {
			pipelineOpts = append(pipelineOpts, bson.D{{Key: "$skip", Value: *opt.Skip}})
		}
		if opt.Projection != nil {
			pipelineOpts = append(pipelineOpts, bson.D{{Key: "$project", Value: opt.Projection}})
		}
		queryOpts = opt.QueryOptions
	case *mopt.FindOneOptions:
		if opt.BatchSize != nil {
			aggOpts.SetBatchSize(*opt.BatchSize)
		}
		if opt.Collation != nil {
			aggOpts.SetCollation(opt.Collation)
		}
		if opt.Comment != nil {
			aggOpts.SetComment(*opt.Comment)
		}
		if opt.Hint != nil {
			aggOpts.SetHint(opt.Hint)
		}
		if opt.MaxAwaitTime != nil {
			aggOpts.SetMaxAwaitTime(*opt.MaxAwaitTime)
		}
		if opt.MaxTime != nil {
			aggOpts.SetMaxTime(*opt.MaxTime)
		}
		if opt.Sort != nil {
			pipelineOpts = append(pipelineOpts, bson.D{{Key: "$sort", Value: opt.Sort}})
		}
		if opt.Skip != nil {
			pipelineOpts = append(pipelineOpts, bson.D{{Key: "$skip", Value: *opt.Skip}})
		}
		if opt.Projection != nil {
			pipelineOpts = append(pipelineOpts, bson.D{{Key: "$project", Value: opt.Projection}})
		}
		queryOpts = opt.QueryOptions
	}
	return pipelineOpts, aggOpts, queryOpts
}

func getStructFields(t reflect.Type) map[string]reflect.StructField {
	fields := make(map[string]reflect.StructField)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tags := strings.Split(field.Tag.Get("bson"), ",")
		// skip fields with no bson tag
		if len(tags) == 0 {
			continue
		}

		// skip fields with "-" tag
		if tags[0] == "-" {
			continue
		}

		// check if field is inlined
		isInlined := false
		for _, tag := range tags {
			if tag == "inline" {
				isInlined = true
				break
			}
		}

		// if field is inlined, get fields from embedded struct
		if isInlined {
			// check if it's a pointer
			typ := field.Type
			if typ.Kind() == reflect.Pointer {
				typ = typ.Elem()
			}

			// check that inlined field is a struct
			if typ.Kind() != reflect.Struct {
				continue
			}

			embeddedFields := getStructFields(typ)
			for key, embeddedField := range embeddedFields {
				fields[key] = embeddedField
			}
		} else if tags[0] != "" {
			// if field is not inlined, add it to fields
			fields[tags[0]] = field
		}

	}
	return fields
}
