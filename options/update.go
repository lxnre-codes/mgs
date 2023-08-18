package options

import "go.mongodb.org/mongo-driver/mongo/options"

type UpdateOptions struct {
	*options.UpdateOptions
	*HookOptions
}
