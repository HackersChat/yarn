package timeline

import (
	"context"
	"encoding/json"
	"fmt"

	"go.yarn.social/types"
)

type jsontParser struct {
}

func (d jsontParser) Parse(ctx context.Context, twts types.Twts, me types.Twter) error {
	data, err := json.Marshal(twts)
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
