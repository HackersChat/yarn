package timeline

import (
	"context"
	"fmt"
	"time"

	"go.yarn.social/types"
)

type defaultParser struct {
}

func (d defaultParser) Parse(ctx context.Context, twts types.Twts, me types.Twter) error {
	for _, twt := range twts {
		PrintTwt(twt, time.Now(), me)
		fmt.Println()
	}
	return nil
}

type defaultRawParser struct {
}

func (d defaultRawParser) Parse(ctx context.Context, twts types.Twts, me types.Twter) error {
	for _, twt := range twts {
		PrintTwtRaw(twt)
	}

	return nil
}
