package celestia

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/celestiaorg/nmt"
	"github.com/gogo/protobuf/proto"
	"github.com/tendermint/tendermint/libs/pubsub"

	openrpc "github.com/rollkit/celestia-openrpc"

	"github.com/rollkit/celestia-openrpc/types/blob"

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

// TODO(srene): export from github.com/celestiaorg/celestia-app/pkg/appconsts or get them from lightnode
var DefaultGovMaxSquareSize = 64
var ContinuationSparseShareContentSize = 512 - 29 - 1
var DefaultMaxBytes = DefaultGovMaxSquareSize * DefaultGovMaxSquareSize * ContinuationSparseShareContentSize

//var DefaultMaxBytes = 5000

// Blob is the data submitted/received from DA interface.
type Blob = []byte

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
/*func (c *DataAvailabilityLayerClient) SubmitBatchPayForBlob(batch *types.Batch) da.ResultSubmitBatch {
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
			Code: da.StatusSuccess,
			MetaData: &da.DAMetaData{
				Height: dataLayerHeight,
			},
		},
		Batches: batches,
	}
}*/

// SubmitBatch submits a batch to the DA layer.
func (c *DataAvailabilityLayerClient) SubmitBatch(batch *types.Batch) da.ResultSubmitBatch {

	var blobs [][]byte

	data, err := batch.MarshalBinary()
	blobs = [][]byte{data}
	if err != nil {
		return da.ResultSubmitBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
			},
		}
	}
	if len(data) > DefaultMaxBytes {
		return da.ResultSubmitBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: "Batch size bigger than maximum blob size",
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
			//TODO(srene):  Split batch in multiple blobs if necessary if supported
			height, commitments, blobs, err := c.submit(blobs, gasPrice)

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
			var numShares []int
			var indexes []int
			for _, commitment := range commitments {
				proof, err := c.getProof(height, commitment)
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

				nmtProofs := []*nmt.Proof(*proof)
				shares := 0
				for i, proof := range nmtProofs {
					if i == 0 {
						indexes = append(indexes, proof.Start())
					}
					shares += proof.End() - proof.Start()
				}
				numShares = append(numShares, shares)

				included, err := c.validateProof(height, commitment, proof)
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
					Code:    da.StatusSuccess,
					Message: "Submission successful",
					MetaData: &da.DAMetaData{
						Height:      uint64(height),
						Commitments: commitments,
						Indexes:     indexes,
						Lengths:     numShares},
				},
			}
		}
	}

}

func (c *DataAvailabilityLayerClient) RetrieveBatches(daMetaData *da.DAMetaData) da.ResultRetrieveBatch {

	var batches []*types.Batch
	for i, commitment := range daMetaData.Commitments {
		blob, err := c.rpc.Get(c.ctx, daMetaData.Height, c.config.NamespaceID.Bytes(), commitment)
		if err != nil {
			return da.ResultRetrieveBatch{
				BaseResult: da.BaseResult{
					Code:    da.StatusBlobNotFound,
					Message: err.Error(),
				},
			}
		}

		var batch pb.Batch
		err = proto.Unmarshal(blob.Data, &batch)
		if err != nil {
			c.logger.Error("failed to unmarshal block", "daHeight", daMetaData.Height, "error", err)
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

		proof, err := c.getProof(daMetaData.Height, commitment)
		if err != nil {
			c.logger.Error("Failed to submit DA batch", "error", err)
			return da.ResultRetrieveBatch{
				BaseResult: da.BaseResult{
					Code:    da.StatusError,
					Message: err.Error(),
				},
			}
		}
		nmtProofs := []*nmt.Proof(*proof)
		shares := 0
		index := 0
		for i, proof := range nmtProofs {
			if i == 0 {
				index = proof.Start()
			}
			shares += proof.End() - proof.Start()
		}

		if index != daMetaData.Indexes[i] || shares != daMetaData.Lengths[i] {
			return da.ResultRetrieveBatch{
				BaseResult: da.BaseResult{
					Code:    da.StatusProofNotMatching,
					Message: err.Error(),
				},
			}
		}
		included, err := c.validateProof(daMetaData.Height, commitment, proof)
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
					Code:    da.StatusBlobNotIncluded,
					Message: err.Error(),
				},
			}
		}
	}
	return da.ResultRetrieveBatch{
		BaseResult: da.BaseResult{
			Code:    da.StatusSuccess,
			Message: "Batch retrieval successful",
			MetaData: &da.DAMetaData{
				Height: daMetaData.Height,
			},
		},
		Batches: batches,
	}
}

// Submit submits the Blobs to Data Availability layer.
func (c *DataAvailabilityLayerClient) submit(daBlobs []Blob, gasPrice float64) (uint64, []da.Commitment, []*blob.Blob, error) {
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

func (c *DataAvailabilityLayerClient) getProof(height uint64, commitment da.Commitment) (*blob.Proof, error) {

	c.logger.Info("Getting proof via RPC call")
	proof, err := c.rpc.GetProof(c.ctx, height, c.config.NamespaceID.Bytes(), commitment)
	if err != nil {
		return nil, err
	}
	c.logger.Info("Proof received", "proof length", proof.Len())

	return proof, nil

}

func (c *DataAvailabilityLayerClient) blobsAndCommitments(daBlobs []Blob) ([]*blob.Blob, []da.Commitment, error) {
	var blobs []*blob.Blob
	var commitments []da.Commitment
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

		commitments = append(commitments, commitment)
	}
	return blobs, commitments, nil
}

func (c *DataAvailabilityLayerClient) validateProof(height uint64, commitment da.Commitment, proof *blob.Proof) (bool, error) {

	c.logger.Info("Getting inclusion validation via RPC call")
	isIncluded, error := c.rpc.Included(c.ctx, height, c.config.NamespaceID.Bytes(), proof, commitment)
	return isIncluded, error
}

/*func splitData(batch *types.Batch, maxBytes int) ([][]byte, error) {

	size := 0
	var batches []*types.Batch
	var blobs [][]byte
	var blocks []*types.Block
	var commits []*types.Commit

	tmpBatch := &types.Batch{
		StartHeight: batch.StartHeight,
		EndHeight:   batch.EndHeight,
	}
	submitted := 1
	for i, block := range batch.Blocks {
		size += block.ToProto().Size()
		if size < maxBytes {
			blocks = append(blocks, block)
			if len(batch.Commits) > i {
				commits = append(commits, batch.Commits[i])
			}
		} else {
			tmpBatch.Blocks = blocks
			tmpBatch.Commits = commits
			batches = append(batches, tmpBatch)
			tmpBatch = &types.Batch{
				StartHeight: batch.StartHeight,
				EndHeight:   batch.EndHeight,
			}
			submitted += 1
			size = block.ToProto().Size()
		}
	}

	tmpBatch.Blocks = blocks
	tmpBatch.Commits = commits
	batches = append(batches, tmpBatch)

	for _, batch := range batches {
		batch.Submitted = submitted
		blob, err := batch.MarshalBinary()
		if err != nil {
			return nil, err
		}
		blobs = append(blobs, blob)
	}

	return blobs, nil
}*/
