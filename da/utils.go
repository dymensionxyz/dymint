package da

import (
	"context"

	"github.com/tendermint/tendermint/libs/pubsub"
)

func SubmitBatchHealthEventHelper(pubsubServer *pubsub.Server, ctx context.Context, err error) (ResultSubmitBatch, error) {
	err = pubsubServer.PublishWithEvents(
		ctx,
		&EventDataHealth{Error: err},
		HealthStatus,
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
