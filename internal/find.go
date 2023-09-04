package internal

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	mopt "github.com/0x-buidl/mgs/options"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// TODO: custom populate errors
func GetPopulateStages(doc any, opt *mopt.PopulateOptions) (mongo.Pipeline, error) {
	lookupPipeline := mongo.Pipeline{
		bson.D{
			{
				Key: "$match",
				Value: bson.M{
					"$expr": bson.M{"$eq": bson.A{"$$localField", "$" + *opt.ForeignField}},
				},
			},
		},
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

				pipe, pErr := GetPopulateStages(opt.Schema, p)

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
			elemType := ft.Elem()
			if elemType.Kind() == reflect.Pointer {
				elemType = elemType.Elem()
			}
			// check if it's slice of struct
			if len(paths) > 1 && elemType.Kind() != reflect.Struct {
				return nil, fmt.Errorf("field %s must be a slice of struct", currPath)
			}
			// if the field is a slice, we need to unwind it
			populatePipeline = append(
				populatePipeline,
				bson.D{{Key: "$unwind", Value: "$" + currPath}},
			)
			group := bson.E{
				Key:   "$group",
				Value: bson.M{"_id": "$_id"},
			}

			if len(paths) < len(strings.Split(*opt.Path, ".")) {
				group.Value.(bson.M)["_id"] = nil
			}

			// wind paths in descending order
			for k := range fields {
				if k == currPath {
					group.Value.(bson.M)[k] = bson.M{"$push": "$" + k}
				} else {
					group.Value.(bson.M)[k] = bson.M{"$first": "$" + k}
				}
			}
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
				{
					Key:   "$unwind",
					Value: bson.M{"path": "$" + *opt.Path, "preserveNullAndEmptyArrays": true},
				},
			},
		)
	}
	populatePipeline = append(populatePipeline, windPaths...)
	return populatePipeline, nil
}

func getStructFields(t reflect.Type) map[string]reflect.StructField {
	fields := make(map[string]reflect.StructField)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("bson")
		if tag != "" && tag != "-" {
			fields[tag] = field
		}
	}
	return fields
}
