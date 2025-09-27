package timeline

import (
	"context"
	"os"
	"sort"
	"time"

	log "github.com/sirupsen/logrus"

	"go.yarn.social/client"
	"go.yarn.social/types"
)

type followParser struct {
	client *client.Client
	parser Parser
}

func (f followParser) Parse(ctx context.Context, twts types.Twts, me types.Twter) error {
	f.parser.Parse(ctx, twts, me)

	ticker := time.NewTicker(5 * time.Second)
	go func(ctx context.Context, client *client.Client, parser Parser, lastHash string) {
		for {
			select {
			case <-ticker.C:
				twts := checkNewTwts(client, lastHash)
				if twts != nil {
					lastHash = twts[len(twts)-1].Hash()
					parser.Parse(ctx, twts, me)
				}
			case <-ctx.Done():
				return
			}
		}
	}(ctx, f.client, f.parser, twts[len(twts)-1].Hash())

	<-ctx.Done()
	return nil
}

func checkNewTwts(client *client.Client, lastHash string) types.Twts {
	res, err := client.Timeline(0)
	if err != nil {
		log.WithError(err).Error("error retrieving timeline")
		os.Exit(1)
	}

	twts := res.Twts[:]

	for i, t := range twts {
		if t.Hash() == lastHash {
			if i == 0 {
				return nil
			}
			newTwts := twts[:i]
			sort.Sort(sort.Reverse(newTwts))
			return newTwts
		}
	}
	return nil
}
