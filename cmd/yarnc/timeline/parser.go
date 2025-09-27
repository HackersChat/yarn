package timeline

import (
	"context"

	"go.yarn.social/client"
	"go.yarn.social/types"
)

type Parser interface {
	Parse(ctx context.Context, twts types.Twts, me types.Twter) error
}

type Options struct {
	OutputJSON   bool
	OutputRAW    bool
	OutputGIT    bool
	OutputFollow bool
	Client       *client.Client
}

// GetParser Factory ...
func GetParser(opts Options) Parser {
	switch {
	case opts.OutputFollow:
		return followParser{opts.Client, defaultParser{}}
	case opts.OutputJSON:
		return jsontParser{}
	case opts.OutputRAW:
		return defaultRawParser{}
	case opts.OutputGIT:
		return gitParser{defaultPrinter{}}
	default:
		return defaultParser{}
	}
}
