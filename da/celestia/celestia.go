package celestia

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/tendermint/tendermint/libs/pubsub"

	openrpc "github.com/rollkit/celestia-openrpc"

	"github.com/rollkit/celestia-openrpc/types/blob"
	"github.com/rollkit/celestia-openrpc/types/share"

	"github.com/dymensionxyz/dymint/da"
	celtypes "github.com/dymensionxyz/dymint/da/celestia/types"
	"github.com/dymensionxyz/dymint/log"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
	pb "github.com/dymensionxyz/dymint/types/pb/dymint"
)

// DataAvailabilityLayerClient use celestia-node public API.
type DataAvailabilityLayerClient struct {
	rpc celtypes.CelestiaRPCClient

	pubsubServer        *pubsub.Server
	config              Config
	logger              log.Logger
	ctx                 context.Context
	cancel              context.CancelFunc
	txPollingRetryDelay time.Duration
	txPollingAttempts   int
	submitRetryDelay    time.Duration
}

var _ da.DataAvailabilityLayerClient = &DataAvailabilityLayerClient{}
var _ da.BatchRetriever = &DataAvailabilityLayerClient{}

// Blob is the data submitted/received from DA interface.
type Blob = []byte

// Commitment should contain serialized cryptographic commitment to Blob value.
type Commitment = []byte

type Proof = []byte

// WithRPCClient sets rpc client.
func WithRPCClient(rpc celtypes.CelestiaRPCClient) da.Option {
	return func(daLayerClient da.DataAvailabilityLayerClient) {
		daLayerClient.(*DataAvailabilityLayerClient).rpc = rpc
	}
}

// WithTxPollingRetryDelay sets tx polling retry delay.
func WithTxPollingRetryDelay(delay time.Duration) da.Option {
	return func(daLayerClient da.DataAvailabilityLayerClient) {
		daLayerClient.(*DataAvailabilityLayerClient).txPollingRetryDelay = delay
	}
}

// WithTxPollingAttempts sets tx polling retry delay.
func WithTxPollingAttempts(attempts int) da.Option {
	return func(daLayerClient da.DataAvailabilityLayerClient) {
		daLayerClient.(*DataAvailabilityLayerClient).txPollingAttempts = attempts
	}
}

// WithSubmitRetryDelay sets submit retry delay.
func WithSubmitRetryDelay(delay time.Duration) da.Option {
	return func(daLayerClient da.DataAvailabilityLayerClient) {
		daLayerClient.(*DataAvailabilityLayerClient).submitRetryDelay = delay
	}
}

// Init initializes DataAvailabilityLayerClient instance.
func (c *DataAvailabilityLayerClient) Init(config []byte, pubsubServer *pubsub.Server, kvStore store.KVStore, logger log.Logger, options ...da.Option) error {
	c.logger = logger

	if len(config) <= 0 {
		return errors.New("config is empty")
	}
	err := json.Unmarshal(config, &c.config)
	if err != nil {
		return err
	}
	err = (&c.config).InitNamespaceID()
	if err != nil {
		return err
	}

	if c.config.GasPrices != 0 && c.config.Fee != 0 {
		return errors.New("can't set both gas prices and fee")
	}

	if c.config.Fee == 0 && c.config.GasPrices == 0 {
		return errors.New("fee or gas prices must be set")
	}

	if c.config.GasAdjustment == 0 {
		c.config.GasAdjustment = defaultGasAdjustment
	}

	c.pubsubServer = pubsubServer
	// Set defaults
	c.txPollingRetryDelay = defaultTxPollingRetryDelay
	c.txPollingAttempts = defaultTxPollingAttempts
	c.submitRetryDelay = defaultSubmitRetryDelay

	c.ctx, c.cancel = context.WithCancel(context.Background())

	// Apply options
	for _, apply := range options {
		apply(c)
	}

	return nil
}

// Start prepares DataAvailabilityLayerClient to work.
func (c *DataAvailabilityLayerClient) Start() (err error) {
	c.logger.Info("starting Celestia Data Availability Layer Client")

	// other client has already been set
	if c.rpc != nil {
		c.logger.Debug("celestia-node client already set")
		return nil
	}

	rpc, err := openrpc.NewClient(c.ctx, c.config.BaseURL, c.config.AuthToken)
	if err != nil {
		return err
	}

	state, err := rpc.Header.SyncState(c.ctx)
	if err != nil {
		return err
	}

	if !state.Finished() {
		c.logger.Info("waiting for celestia-node to finish syncing", "height", state.Height, "target", state.ToHeight)

		done := make(chan error, 1)
		go func() {
			done <- rpc.Header.SyncWait(c.ctx)
		}()

		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case err := <-done:
				if err != nil {
					return err
				}
				return nil
			case <-ticker.C:
				c.logger.Info("celestia-node still syncing", "height", state.Height, "target", state.ToHeight)
			}
		}
	}

	c.logger.Info("celestia-node is synced", "height", state.ToHeight)

	c.rpc = NewOpenRPC(rpc)
	return nil
}

// Stop stops DataAvailabilityLayerClient.
func (c *DataAvailabilityLayerClient) Stop() error {
	c.logger.Info("stopping Celestia Data Availability Layer Client")
	err := c.pubsubServer.Stop()
	if err != nil {
		return err
	}
	c.cancel()
	return nil
}

// GetClientType returns client type.
func (c *DataAvailabilityLayerClient) GetClientType() da.Client {
	return da.Celestia
}

// SubmitBatch submits a batch to the DA layer.
/*func (c *DataAvailabilityLayerClient) SubmitBatch(batch *types.Batch) da.ResultSubmitBatch {
	data, err := batch.MarshalBinary()
	if err != nil {
		return da.ResultSubmitBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
			},
		}
	}

	blockBlob, err := blob.NewBlobV0(c.config.NamespaceID.Bytes(), data)
	if err != nil {
		return da.ResultSubmitBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
			},
		}
	}
	blobs := []*blob.Blob{blockBlob}

	estimatedGas := DefaultEstimateGas(uint32(len(data)))
	gasWanted := uint64(float64(estimatedGas) * c.config.GasAdjustment)
	fees := c.calculateFees(gasWanted)
	c.logger.Debug("Submitting to da blob with size", "size", len(blockBlob.Data), "estimatedGas", estimatedGas, "gasAdjusted", gasWanted, "fees", fees)

	for {
		select {
		case <-c.ctx.Done():
			c.logger.Debug("Context cancelled")
			return da.ResultSubmitBatch{}
		default:
			txResponse, err := c.rpc.SubmitPayForBlob(c.ctx, math.NewInt(fees), gasWanted, blobs)
			if err != nil {
				c.logger.Error("Failed to submit DA batch. Emitting health event and trying again", "error", err)
				res, err := da.SubmitBatchHealthEventHelper(c.pubsubServer, c.ctx, false, err)
				if err != nil {
					return res
				}
				time.Sleep(c.submitRetryDelay)
				continue
			}

			//double check txResponse is not nil - not supposed to happen
			if txResponse == nil {
				err := errors.New("txResponse is nil")
				c.logger.Error("Failed to submit DA batch", "error", err)
				res, err := da.SubmitBatchHealthEventHelper(c.pubsubServer, c.ctx, false, err)
				if err != nil {
					return res
				}
				time.Sleep(c.submitRetryDelay)
				continue
			}

			c.logger.Info("Successfully submitted DA batch", "txHash", txResponse.TxHash, "daHeight", txResponse.Height, "gasWanted", txResponse.GasWanted, "gasUsed", txResponse.GasUsed)
			res, err := da.SubmitBatchHealthEventHelper(c.pubsubServer, c.ctx, true, nil)
			if err != nil {
				return res
			}
			return da.ResultSubmitBatch{
				BaseResult: da.BaseResult{
					Code:     da.StatusSuccess,
					Message:  "tx hash: " + txResponse.TxHash,
					DAHeight: uint64(txResponse.Height),
				},
			}
		}
	}
}*/

func (c *DataAvailabilityLayerClient) RetrieveBatch(dataLayerHeight uint64, commitment Commitment) da.ResultRetrieveBatch {
	blob, err := c.rpc.Get(c.ctx, dataLayerHeight, c.config.NamespaceID.Bytes(), commitment)
	if err != nil {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
			},
		}
	}

	var batches []*types.Batch
	var batch pb.Batch
	err = proto.Unmarshal(blob.Data, &batch)
	if err != nil {
		c.logger.Error("failed to unmarshal block", "daHeight", dataLayerHeight, "error", err)
	}
	parsedBatch := new(types.Batch)
	err = parsedBatch.FromProto(&batch)
	if err != nil {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
			},
		}
	}
	batches = append(batches, parsedBatch)
	//}
	prof, err := c.getProof(dataLayerHeight, commitment)
	if err != nil {
		c.logger.Error("Failed to submit DA batch", "error", err)
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
			},
		}
	}
	included, err := c.validateBlob(dataLayerHeight, commitment, prof)
	if err != nil {
		c.logger.Error("Failed to submit DA batch", "error", err)
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
			},
		}
	} else if !included {
		err := errors.New("Blob not included")
		c.logger.Error("Failed to submit DA batch", "error", err)
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
			},
		}
	}
	return da.ResultRetrieveBatch{
		BaseResult: da.BaseResult{
			Code:     da.StatusSuccess,
			DAHeight: dataLayerHeight,
		},
		Batches: batches,
	}
}

// RetrieveBatches gets a batch of blocks from DA layer.
func (c *DataAvailabilityLayerClient) RetrieveBatches(dataLayerHeight uint64) da.ResultRetrieveBatch {
	blobs, err := c.rpc.GetAll(c.ctx, dataLayerHeight, []share.Namespace{c.config.NamespaceID.Bytes()})
	if err != nil {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
			},
		}
	}

	var batches []*types.Batch
	for i, blob := range blobs {
		var batch pb.Batch
		err = proto.Unmarshal(blob.Data, &batch)
		if err != nil {
			c.logger.Error("failed to unmarshal block", "daHeight", dataLayerHeight, "position", i, "error", err)
			continue
		}
		parsedBatch := new(types.Batch)
		err := parsedBatch.FromProto(&batch)
		if err != nil {
			return da.ResultRetrieveBatch{
				BaseResult: da.BaseResult{
					Code:    da.StatusError,
					Message: err.Error(),
				},
			}
		}
		batches = append(batches, parsedBatch)
	}

	return da.ResultRetrieveBatch{
		BaseResult: da.BaseResult{
			Code:     da.StatusSuccess,
			DAHeight: dataLayerHeight,
		},
		Batches: batches,
	}
}

// SubmitBatch submits a batch to the DA layer.
func (c *DataAvailabilityLayerClient) SubmitBatch(batch *types.Batch) da.ResultSubmitBatch {
	data, err := batch.MarshalBinary()
	if err != nil {
		return da.ResultSubmitBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
			},
		}
	}

	for {
		select {
		case <-c.ctx.Done():
			c.logger.Debug("Context cancelled")
			return da.ResultSubmitBatch{}
		default:
			gasPrice := float64(-1)
			if c.config.GasPrices >= 0 {
				gasPrice = c.config.GasPrices
			}
			//TODO: (srene) Split batch in multiple blobs if necessary
			height, commitments, blobs, err := c.submit([]Blob{data}, gasPrice)

			c.logger.Info("DA batch submitted", "commitments", len(commitments), "blobs", len(blobs))
			if err != nil {
				c.logger.Error("Failed to submit DA batch. Emitting health event and trying again", "error", err)
				res, err := da.SubmitBatchHealthEventHelper(c.pubsubServer, c.ctx, false, err)
				if err != nil {
					return res
				}
				time.Sleep(c.submitRetryDelay)
				continue
			}

			//double check txResponse is not nil - not supposed to happen
			if commitments == nil {
				err := errors.New("Da commitments are nil")
				c.logger.Error("Failed to submit DA batch", "error", err)
				res, err := da.SubmitBatchHealthEventHelper(c.pubsubServer, c.ctx, false, err)
				if err != nil {
					return res
				}
				time.Sleep(c.submitRetryDelay)
				continue
			}

			for _, commitment := range commitments {
				prof, err := c.getProof(height, commitment)
				if err != nil {
					err := errors.New("Error getting proofs for commitment")
					c.logger.Error("Failed to submit DA batch", "error", err)
					res, err := da.SubmitBatchHealthEventHelper(c.pubsubServer, c.ctx, false, err)
					if err != nil {
						return res
					}
					time.Sleep(c.submitRetryDelay)
					continue
				}
				included, err := c.validateBlob(height, commitment, prof)
				if err != nil {
					err := errors.New("Error in validating proof inclusion")
					c.logger.Error("Failed to submit DA batch", "error", err)
					res, err := da.SubmitBatchHealthEventHelper(c.pubsubServer, c.ctx, false, err)
					if err != nil {
						return res
					}
					time.Sleep(c.submitRetryDelay)
					continue
				} else if !included {
					err := errors.New("Blob not included")
					c.logger.Error("Failed to submit DA batch", "error", err)
					res, err := da.SubmitBatchHealthEventHelper(c.pubsubServer, c.ctx, false, err)
					if err != nil {
						return res
					}
					time.Sleep(c.submitRetryDelay)
					continue
				}
			}

			if err != nil {
				err := errors.New("da proofs are not valid")
				c.logger.Error("Failed to submit DA batch", "error", err)
				res, err := da.SubmitBatchHealthEventHelper(c.pubsubServer, c.ctx, false, err)
				if err != nil {
					return res
				}
				time.Sleep(c.submitRetryDelay)
				continue
			}

			res, err := da.SubmitBatchHealthEventHelper(c.pubsubServer, c.ctx, true, nil)
			if err != nil {
				return res
			}
			return da.ResultSubmitBatch{
				BaseResult: da.BaseResult{
					Code: da.StatusSuccess,
					//Message:  "tx hash: " + txResponse.TxHash,
					DAHeight: uint64(height),
				},
			}
		}
	}

}

// Submit submits the Blobs to Data Availability layer.
func (c *DataAvailabilityLayerClient) submit(daBlobs []Blob, gasPrice float64) (uint64, []Commitment, []*blob.Blob, error) {
	blobs, commitments, err := c.blobsAndCommitments(daBlobs)
	if err != nil {
		return 0, nil, nil, err
	}
	options := openrpc.DefaultSubmitOptions()

	if gasPrice >= 0 {
		blobSizes := make([]uint32, len(blobs))
		for i, blob := range blobs {
			blobSizes[i] = uint32(len(blob.Data))
		}

		estimatedGas := EstimateGas(blobSizes, DefaultGasPerBlobByte, DefaultTxSizeCostPerByte)
		gasWanted := uint64(float64(estimatedGas) * c.config.GasAdjustment)
		fees := c.calculateFees(gasWanted)
		options.Fee = fees
		options.GasLimit = gasWanted
	}
	height, err := c.rpc.Submit(c.ctx, blobs, options)

	if err != nil {
		return 0, nil, nil, err
	}
	c.logger.Info("Successfully submitted blobs to Celestia", "height", height, "gas", options.GasLimit, "fee", options.Fee)

	return height, commitments, blobs, nil
}

func (c *DataAvailabilityLayerClient) getProof(height uint64, commitment Commitment) (*blob.Proof, error) {

	c.logger.Info("Getting proof via RPC call")
	proof, err := c.rpc.GetProof(c.ctx, height, c.config.NamespaceID.Bytes(), commitment)
	if err != nil {
		return nil, err
	}
	c.logger.Info("Proof received", "proof length", proof.Len())

	return proof, nil

}

// blobsAndCommitments converts []da.Blob to []*blob.Blob and generates corresponding []da.Commitment
func (c *DataAvailabilityLayerClient) blobsAndCommitments(daBlobs []Blob) ([]*blob.Blob, []Commitment, error) {
	var blobs []*blob.Blob
	var commitments []Commitment
	for _, daBlob := range daBlobs {
		b, err := blob.NewBlobV0(c.config.NamespaceID.Bytes(), daBlob)
		if err != nil {
			return nil, nil, err
		}
		blobs = append(blobs, b)

		commitment, err := blob.CreateCommitment(b)
		if err != nil {
			return nil, nil, err
		}
		//commitment := []byte{0}
		commitments = append(commitments, commitment)

		shares, err := blob.SplitBlobs(*b)
		if err != nil {
			c.logger.Info("Commitment created", "num_shares", len(shares))
		}
	}
	return blobs, commitments, nil
}

func (c *DataAvailabilityLayerClient) validateBlob(height uint64, commitment Commitment, proofs *blob.Proof) (bool, error) {

	c.logger.Info("Getting inclusion validation via RPC call")
	isIncluded, error := c.rpc.Included(c.ctx, height, c.config.NamespaceID.Bytes(), proofs, commitment)
	return isIncluded, error
}
