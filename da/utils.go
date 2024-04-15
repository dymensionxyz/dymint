package da

import (
	"context"

	"github.com/tendermint/tendermint/libs/pubsub"
)

// TODO: I need to check callsites of this, and help it match up with what I did with settlement, and make sure the healthy=noerror thing is continued
func SubmitBatchHealthEventHelper(pubsubServer *pubsub.Server, ctx context.Context, err error) (ResultSubmitBatch, error) {
	err = pubsubServer.PublishWithEvents(
		ctx,
		&EventDataHealth{Error: err},
		map[string][]string{EventTypeKey: {EventDAHealthStatus}},
	)
	if err != nil {
		return ResultSubmitBatch{
			BaseResult: BaseResult{
				Code:    StatusError,
				Message: err.Error(),
				Error:   err,
			},
		}, err
	}
	return ResultSubmitBatch{}, nil
}
