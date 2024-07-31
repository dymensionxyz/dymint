package dymension

import (
	"github.com/avast/retry-go/v4"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/settlement"
	rollapptypes "github.com/dymensionxyz/dymint/third_party/dymension/rollapp/types"
)

// RunWithRetry runs the given operation with retry, doing a number of attempts, and taking the last
// error only. It uses the context of the HubClient.
func (c *Client) RunWithRetry(operation func() error) error {
	return retry.Do(operation,
		retry.Context(c.ctx),
		retry.LastErrorOnly(true),
		retry.Delay(c.retryMinDelay),
		retry.Attempts(c.retryAttempts),
		retry.MaxDelay(c.retryMaxDelay),
	)
}

// RunWithRetryInfinitely runs the given operation with retry, doing a number of attempts, and taking the last
// error only. It uses the context of the HubClient.
func (c *Client) RunWithRetryInfinitely(operation func() error) error {
	return retry.Do(operation,
		retry.Context(c.ctx),
		retry.LastErrorOnly(true),
		retry.Delay(c.retryMinDelay),
		retry.Attempts(0),
		retry.MaxDelay(c.retryMaxDelay),
	)
}

func convertStateInfoToResultRetrieveBatch(stateInfo *rollapptypes.StateInfo) (*settlement.ResultRetrieveBatch, error) {
	daMetaData := &da.DASubmitMetaData{}
	daMetaData, err := daMetaData.FromPath(stateInfo.DAPath)
	if err != nil {
		return nil, err
	}
	batchResult := &settlement.Batch{
		StartHeight: stateInfo.StartHeight,
		EndHeight:   stateInfo.StartHeight + stateInfo.NumBlocks - 1,
		MetaData: &settlement.BatchMetaData{
			DA: daMetaData,
		},
	}
	return &settlement.ResultRetrieveBatch{
		ResultBase: settlement.ResultBase{Code: settlement.StatusSuccess, StateIndex: stateInfo.StateInfoIndex.Index},
		Batch:      batchResult,
	}, nil
}
