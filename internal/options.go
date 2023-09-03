package internal

import (
	mopt "github.com/0x-buidl/mgs/options"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UnionFindOpts interface {
	*mopt.FindOptions | *mopt.FindOneOptions
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

func MergeFindOptsWithAggregatOpts[T UnionFindOpts](
	opt T,
) (mongo.Pipeline, *options.AggregateOptions, *mopt.QueryOptions) {
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
