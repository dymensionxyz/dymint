package celestia

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/celestiaorg/nmt"
	"github.com/gogo/protobuf/proto"
	"github.com/tendermint/tendermint/libs/pubsub"

	openrpc "github.com/rollkit/celestia-openrpc"

	"github.com/rollkit/celestia-openrpc/types/blob"
	"github.com/rollkit/celestia-openrpc/types/header"
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

	pubsubServer                *pubsub.Server
	config                      Config
	logger                      log.Logger
	ctx                         context.Context
	cancel                      context.CancelFunc
	availabilityCheckRetryDelay time.Duration
	availabilityCheckAttempts   int
	submitRetryDelay            time.Duration
}

var _ da.DataAvailabilityLayerClient = &DataAvailabilityLayerClient{}
var _ da.BatchRetriever = &DataAvailabilityLayerClient{}

// WithRPCClient sets rpc client.
func WithRPCClient(rpc celtypes.CelestiaRPCClient) da.Option {
	return func(daLayerClient da.DataAvailabilityLayerClient) {
		daLayerClient.(*DataAvailabilityLayerClient).rpc = rpc
	}
}

// WithTxPollingRetryDelay sets tx polling retry delay.
func WithAvailabilityCheckRetryDelay(delay time.Duration) da.Option {
	return func(daLayerClient da.DataAvailabilityLayerClient) {
		daLayerClient.(*DataAvailabilityLayerClient).availabilityCheckRetryDelay = delay
	}
}

// WithTxPollingAttempts sets tx polling retry delay.
func WithAvailabilityCheckAttempts(attempts int) da.Option {
	return func(daLayerClient da.DataAvailabilityLayerClient) {
		daLayerClient.(*DataAvailabilityLayerClient).availabilityCheckAttempts = attempts
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
	c.availabilityCheckRetryDelay = defaultAvailabilityCheckRetryDelay
	c.availabilityCheckAttempts = defaultAvailabilityCheckAttempts
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
func (c *DataAvailabilityLayerClient) SubmitBatch(batch *types.Batch) da.ResultSubmitBatch {

	data, err := batch.MarshalBinary()
	if err != nil {
		return da.ResultSubmitBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
				Error:   err,
			},
		}
	}

	if len(data) > celtypes.DefaultMaxBytes {
		return da.ResultSubmitBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("size bigger than maximum blob size of %d bytes", celtypes.DefaultMaxBytes),
				Error:   errors.New("blob size too big"),
			},
		}
	}

	for {
		select {
		case <-c.ctx.Done():
			c.logger.Debug("Context cancelled")
			return da.ResultSubmitBatch{}
		default:

			c.logger.Info("Submitting DA batch")
			//TODO(srene):  Split batch in multiple blobs if necessary if supported
			height, commitment, err := c.submit(data)

			if err != nil {
				c.logger.Error("Failed to submit DA batch. Emitting health event and trying again", "error", err)
				res, err := da.SubmitBatchHealthEventHelper(c.pubsubServer, c.ctx, false, err)
				if err != nil {
					return res
				}
				time.Sleep(c.submitRetryDelay)
				continue
			}

			daMetaData := &da.DASubmitMetaData{
				Client:     da.Celestia,
				Height:     height,
				Commitment: commitment,
				Namespace:  c.config.NamespaceID.Bytes(),
			}
			var availabilityResult da.ResultCheckBatch
			err = retry.Do(func() error {
				result := c.CheckBatchAvailability(daMetaData)
				if result.Code != da.StatusSuccess {
					c.logger.Error("Blob submitted not found in DA. Retrying availability check")
					return errors.New("blob not found")
				}
				availabilityResult = result

				return nil
			}, retry.Attempts(uint(c.availabilityCheckAttempts)), retry.DelayType(retry.FixedDelay), retry.Delay(c.availabilityCheckRetryDelay))

			if availabilityResult.Code != da.StatusSuccess {
				c.logger.Error("Unable to confirm submitted blob availability. Retrying")
				res, err := da.SubmitBatchHealthEventHelper(c.pubsubServer, c.ctx, false, err)
				if err != nil {
					return res
				}
				time.Sleep(c.submitRetryDelay)
				continue
			}
			daMetaData.Root = availabilityResult.CheckMetaData.Root
			daMetaData.Index = availabilityResult.CheckMetaData.Index
			daMetaData.Length = availabilityResult.CheckMetaData.Length

			res, err := da.SubmitBatchHealthEventHelper(c.pubsubServer, c.ctx, true, nil)
			if err != nil {
				return res
			}
			return da.ResultSubmitBatch{
				BaseResult: da.BaseResult{
					Code:    da.StatusSuccess,
					Message: "Submission successful",
				},
				SubmitMetaData: daMetaData,
			}
		}
	}

}

func (c *DataAvailabilityLayerClient) RetrieveBatches(daMetaData *da.DASubmitMetaData) da.ResultRetrieveBatch {

	ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
	defer cancel()
	//Just for backward compatibility, in case no commitments are sent from the Hub, batch can be retrieved using previous implementation.
	if daMetaData.Commitment == nil {
		return c.retrieveBatches(ctx, daMetaData.Height)
	}

	for {
		select {
		case <-c.ctx.Done():
			c.logger.Debug("Context cancelled")
			return da.ResultRetrieveBatch{}
		default:
			var batches []*types.Batch
			blob, err := c.rpc.Get(ctx, daMetaData.Height, c.config.NamespaceID.Bytes(), daMetaData.Commitment)
			if err != nil {
				return da.ResultRetrieveBatch{
					BaseResult: da.BaseResult{
						Code:    da.StatusError,
						Message: err.Error(),
						Error:   da.ErrBlobNotFound,
					},
				}
			}
			if blob == nil {
				return da.ResultRetrieveBatch{
					BaseResult: da.BaseResult{
						Code:    da.StatusError,
						Message: "Blob not found",
						Error:   da.ErrBlobNotFound,
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
						Error:   err,
					},
				}
			}
			batches = append(batches, parsedBatch)
			//}
			return da.ResultRetrieveBatch{
				BaseResult: da.BaseResult{
					Code:    da.StatusSuccess,
					Message: "Batch retrieval successful",
				},
				Batches: batches,
			}
		}
	}
}

// RetrieveBatches gets a batch of blocks from DA layer.
func (c *DataAvailabilityLayerClient) retrieveBatches(ctx context.Context, dataLayerHeight uint64) da.ResultRetrieveBatch {
	blobs, err := c.rpc.GetAll(ctx, dataLayerHeight, []share.Namespace{c.config.NamespaceID.Bytes()})
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
			Code: da.StatusSuccess},
		Batches: batches,
	}
}

func (c *DataAvailabilityLayerClient) CheckBatchAvailability(daMetaData *da.DASubmitMetaData) da.ResultCheckBatch {

	var proofs []*blob.Proof

	DACheckMetaData := &da.DACheckMetaData{}
	DACheckMetaData.Height = daMetaData.Height
	DACheckMetaData.Client = daMetaData.Client
	DACheckMetaData.Commitment = daMetaData.Commitment

	dah, err := c.getDataAvailabilityHeaders(daMetaData.Height)
	if err != nil {
		//Returning Data Availability header Data Root for dispute validation
		return da.ResultCheckBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("Error getting row to data root proofs: %s", err),
				Error:   da.ErrUnableToGetProof,
			},
			CheckMetaData: DACheckMetaData,
		}
	}
	DACheckMetaData.Root = dah.Hash()
	included := false

	proof, err := c.getProof(daMetaData.Height, daMetaData.Commitment)
	if err != nil || proof == nil {

		//TODO (srene): Not getting proof means there is no existing data for the namespace and the commitment (the commitment is wrong).
		//Therefore we need to prove whether the commitment is wrong or the span does not exists.
		//In case the span is correct it is necessary to return the data for the span and the proofs to the data root, so we can prove the data
		//is the data for the span, and reproducing the commitment will generate a different one.
		return da.ResultCheckBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("Error getting NMT proof: %s", err),
				Error:   da.ErrUnableToGetProof,
			},
			CheckMetaData: DACheckMetaData,
		}
	}

	nmtProofs := []*nmt.Proof(*proof)
	shares := 0
	index := 0
	for j, proof := range nmtProofs {
		if j == 0 {
			index = proof.Start()
		}
		shares += proof.End() - proof.Start()
	}

	if daMetaData.Index > 0 && daMetaData.Length > 0 {
		if index != daMetaData.Index || shares != daMetaData.Length {

			//TODO (srene): In this case the commitment is correct but does not match the span.
			//If the span is correct we have to repeat the previous step (sending data + proof of data)
			//In case the span is not correct we need to send unavailable proof by sending proof of any row root to data root
			return da.ResultCheckBatch{
				CheckMetaData: DACheckMetaData,
				BaseResult: da.BaseResult{
					Code: da.StatusError,
					Message: fmt.Sprintf("Proof index not matching: %d != %d or length not matching: %d != %d",
						index, daMetaData.Index, shares, daMetaData.Length),
					Error: da.ErrProofNotMatching,
				},
			}
		}
	}

	included, err = c.validateProof(daMetaData.Height, daMetaData.Commitment, proof)
	//The both cases below (there is an error validating the proof or the proof is wrong) should not happen
	//if we consider correct functioning of the celestia light node.
	//This will only happen in case the previous step the celestia light node returned wrong proofs..
	if err != nil {
		return da.ResultCheckBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: "Error validating proof",
				Error:   err,
			},
			CheckMetaData: DACheckMetaData,
		}
	} else if !included {
		return da.ResultCheckBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: "Blob not included",
				Error:   da.ErrBlobNotIncluded,
			},
			CheckMetaData: DACheckMetaData,
		}
	}
	proofs = append(proofs, proof)

	DACheckMetaData.Index = index
	DACheckMetaData.Length = shares
	DACheckMetaData.Proofs = proofs
	DACheckMetaData.Namespace = c.config.NamespaceID.Bytes()
	return da.ResultCheckBatch{
		BaseResult: da.BaseResult{
			Code:    da.StatusSuccess,
			Message: "Blob available",
		},
		CheckMetaData: DACheckMetaData,
	}
}

// Submit submits the Blobs to Data Availability layer.
func (c *DataAvailabilityLayerClient) submit(daBlob da.Blob) (uint64, da.Commitment, error) {
	blobs, commitments, err := c.blobsAndCommitments(daBlob)
	if err != nil {
		return 0, nil, err
	}

	options := openrpc.DefaultSubmitOptions()

	blobSizes := make([]uint32, len(blobs))
	for i, blob := range blobs {
		blobSizes[i] = uint32(len(blob.Data))
	}

	estimatedGas := EstimateGas(blobSizes, DefaultGasPerBlobByte, DefaultTxSizeCostPerByte)
	gasWanted := uint64(float64(estimatedGas) * c.config.GasAdjustment)
	fees := c.calculateFees(gasWanted)
	options.Fee = fees
	options.GasLimit = gasWanted
	ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
	defer cancel()

	height, err := c.rpc.Submit(ctx, blobs, options)

	if err != nil {
		return 0, nil, err
	}
	c.logger.Info("Successfully submitted blobs to Celestia", "height", height, "gas", options.GasLimit, "fee", options.Fee)

	return height, commitments[0], nil
}

func (c *DataAvailabilityLayerClient) getProof(height uint64, commitment da.Commitment) (*blob.Proof, error) {

	c.logger.Info("Getting proof via RPC call")
	ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
	defer cancel()

	proof, err := c.rpc.GetProof(ctx, height, c.config.NamespaceID.Bytes(), commitment)
	if err != nil {
		return nil, err
	}

	return proof, nil

}

func (c *DataAvailabilityLayerClient) blobsAndCommitments(daBlob da.Blob) ([]*blob.Blob, []da.Commitment, error) {
	var blobs []*blob.Blob
	var commitments []da.Commitment
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
	return blobs, commitments, nil
}

func (c *DataAvailabilityLayerClient) validateProof(height uint64, commitment da.Commitment, proof *blob.Proof) (bool, error) {

	c.logger.Info("Getting inclusion validation via RPC call")
	ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
	defer cancel()
	isIncluded, error := c.rpc.Included(ctx, height, c.config.NamespaceID.Bytes(), proof, commitment)
	return isIncluded, error
}

func (c *DataAvailabilityLayerClient) getDataAvailabilityHeaders(height uint64) (*header.DataAvailabilityHeader, error) {

	c.logger.Info("Getting Celestia extended headers via RPC call")
	ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
	defer cancel()
	headers, error := c.rpc.GetHeaders(ctx, height)

	if error != nil {
		return nil, error
	}
	return headers.DAH, error

}
