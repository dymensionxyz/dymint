package da

import (
	"context"

	"github.com/tendermint/tendermint/libs/pubsub"
)

func SubmitBatchHealthEventHelper(pubsubServer *pubsub.Server, ctx context.Context, healthy bool, err error) (ResultSubmitBatch, error) {
	err = pubsubServer.PublishWithEvents(ctx, &EventDataDAHealthStatus{Healthy: healthy, Error: err},
		map[string][]string{EventTypeKey: {EventDAHealthStatus}})
	if err != nil {
		return ResultSubmitBatch{
			BaseResult: BaseResult{
				Code:    StatusError,
				Message: err.Error(),
			},
		}, err
	} else {
		return ResultSubmitBatch{}, nil
	}

}
