package celestia

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	errorsmod "cosmossdk.io/errors"
	"github.com/avast/retry-go/v4"
	"github.com/celestiaorg/nmt"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	"github.com/gogo/protobuf/proto"
	goDA "github.com/rollkit/go-da"

	"github.com/dymensionxyz/dymint/da/celestia/client"
	daclient "github.com/dymensionxyz/dymint/da/celestia/client"
	pb "github.com/dymensionxyz/dymint/types/pb/dymint"
	"github.com/tendermint/tendermint/libs/pubsub"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
	"github.com/dymensionxyz/dymint/types/metrics"
	uretry "github.com/dymensionxyz/dymint/utils/retry"
)

// heightLen is a length (in bytes) of serialized height.
//
// This is 8 as uint64 consist of 8 bytes.
const heightLen = 8

// DataAvailabilityLayerClient use celestia-node public API.
type DataAvailabilityLayerClient struct {
	client       daclient.DAClient
	pubsubServer *pubsub.Server
	config       Config
	logger       types.Logger
	ctx          context.Context
	cancel       context.CancelFunc
	synced       chan struct{}
}

var (
	_ da.DataAvailabilityLayerClient = &DataAvailabilityLayerClient{}
	_ da.BatchRetriever              = &DataAvailabilityLayerClient{}
)

// WithRPCClient sets rpc client.
func WithRPCClient(client client.DAClient) da.Option {
	return func(daLayerClient da.DataAvailabilityLayerClient) {
		daLayerClient.(*DataAvailabilityLayerClient).client = client
	}
}

// WithRPCRetryDelay sets failed rpc calls retry delay.
func WithRPCRetryDelay(delay time.Duration) da.Option {
	return func(daLayerClient da.DataAvailabilityLayerClient) {
		daLayerClient.(*DataAvailabilityLayerClient).config.RetryDelay = delay
	}
}

// WithRPCAttempts sets failed rpc calls retry attempts.
func WithRPCAttempts(attempts int) da.Option {
	return func(daLayerClient da.DataAvailabilityLayerClient) {
		daLayerClient.(*DataAvailabilityLayerClient).config.RetryAttempts = &attempts
	}
}

// WithSubmitBackoff sets submit retry delay config.
func WithSubmitBackoff(c uretry.BackoffConfig) da.Option {
	return func(daLayerClient da.DataAvailabilityLayerClient) {
		daLayerClient.(*DataAvailabilityLayerClient).config.Backoff = c
	}
}

// Init initializes DataAvailabilityLayerClient instance.
func (c *DataAvailabilityLayerClient) Init(config []byte, pubsubServer *pubsub.Server, _ store.KV, logger types.Logger, options ...da.Option) error {
	c.logger = logger
	c.synced = make(chan struct{}, 1)
	var err error
	c.config, err = createConfig(config)
	if err != nil {
		return fmt.Errorf("create config: %w: %w", err, gerrc.ErrInvalidArgument)
	}

	c.ctx, c.cancel = context.WithCancel(context.Background())

	c.pubsubServer = pubsubServer

	// Apply options
	for _, apply := range options {
		apply(c)
	}

	metrics.RollappConsecutiveFailedDASubmission.Set(0)

	return nil
}

func (c DataAvailabilityLayerClient) DAPath() string {
	return c.config.NamespaceIDStr
}

func createConfig(bz []byte) (c Config, err error) {
	if len(bz) <= 0 {
		return c, errors.New("supplied config is empty")
	}
	err = json.Unmarshal(bz, &c)
	if err != nil {
		return c, fmt.Errorf("json unmarshal: %w", err)
	}
	err = c.InitNamespaceID()
	if err != nil {
		return c, fmt.Errorf("init namespace id: %w", err)
	}

	if c.GasPrices == 0 {
		return c, errors.New("gas prices must be set")
	}

	// NOTE: 0 is valid value for RetryAttempts

	if c.RetryDelay == 0 {
		c.RetryDelay = defaultRpcRetryDelay
	}
	if c.Backoff == (uretry.BackoffConfig{}) {
		c.Backoff = defaultSubmitBackoff
	}
	if c.RetryAttempts == nil {
		attempts := defaultRpcRetryAttempts
		c.RetryAttempts = &attempts
	}
	return c, nil
}

// Start prepares DataAvailabilityLayerClient to work.
func (c *DataAvailabilityLayerClient) Start() (err error) {
	c.logger.Info("Starting Celestia Data Availability Layer Client.")

	// other client has already been set
	if c.client != nil {
		c.logger.Info("Celestia-node client already set.")
		c.synced <- struct{}{}
		return nil
	}

	client, err := daclient.NewClient(c.config.BaseURL, c.config.AuthToken)
	if err != nil {
		return fmt.Errorf("error while establishing connection to DA layer: %w", err)
	}
	c.client = client
	c.synced <- struct{}{}

	return
}

// Stop stops DataAvailabilityLayerClient.
func (c *DataAvailabilityLayerClient) Stop() error {
	c.logger.Info("Stopping Celestia Data Availability Layer Client.")
	err := c.pubsubServer.Stop()
	if err != nil {
		return err
	}
	c.cancel()
	close(c.synced)
	return nil
}

// WaitForSyncing is used to check when the DA light client finished syncing
func (m *DataAvailabilityLayerClient) WaitForSyncing() {
	<-m.synced
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

	backoff := c.config.Backoff.Backoff()

	for {
		select {
		case <-c.ctx.Done():
			c.logger.Debug("Context cancelled.")
			return da.ResultSubmitBatch{}
		default:

			// TODO(srene):  Split batch in multiple blobs if necessary if supported
			ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
			defer cancel()
			ids, err := c.client.Submit(ctx, []da.Blob{data}, c.config.GasPrices, c.config.NamespaceID.Bytes())
			if err != nil {
				c.logger.Error("Submit blob.", "error", err)
				metrics.RollappConsecutiveFailedDASubmission.Inc()
				backoff.Sleep()
				continue
			}
			height, commitment := splitID(ids[0])

			daMetaData := &da.DASubmitMetaData{
				Client:     da.Celestia,
				Height:     height,
				Commitment: commitment,
				Namespace:  c.config.NamespaceID.Bytes(),
			}

			c.logger.Debug("Submitted blob to DA successfully.")

			result := c.CheckBatchAvailability(daMetaData)
			if result.Code != da.StatusSuccess {
				c.logger.Error("Check batch availability: submitted batch but did not get availability success status.", "error", err)
				metrics.RollappConsecutiveFailedDASubmission.Inc()
				backoff.Sleep()
				continue
			}
			daMetaData.Root = result.CheckMetaData.Root
			daMetaData.Index = result.CheckMetaData.Index
			daMetaData.Length = result.CheckMetaData.Length

			c.logger.Debug("Blob availability check passed successfully.")

			metrics.RollappConsecutiveFailedDASubmission.Set(0)
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
	for {
		select {
		case <-c.ctx.Done():
			c.logger.Debug("Context cancelled.")
			return da.ResultRetrieveBatch{}
		default:
			var resultRetrieveBatch da.ResultRetrieveBatch

			err := retry.Do(
				func() error {
					resultRetrieveBatch = c.retrieveBatches(daMetaData)
					return resultRetrieveBatch.Error
				},
				retry.Attempts(uint(*c.config.RetryAttempts)), //nolint:gosec // RetryAttempts should be always positive
				retry.DelayType(retry.FixedDelay),
				retry.Delay(c.config.RetryDelay),
			)
			if err != nil {
				c.logger.Error("Retrieve batch", "height", daMetaData.Height, "commitment", hex.EncodeToString(daMetaData.Commitment), "error", err)
			}
			return resultRetrieveBatch

		}
	}
}

func (c *DataAvailabilityLayerClient) CheckBatchAvailability(daMetaData *da.DASubmitMetaData) da.ResultCheckBatch {
	var availabilityResult da.ResultCheckBatch
	for {
		select {
		case <-c.ctx.Done():
			c.logger.Debug("Context cancelled")
			return da.ResultCheckBatch{}
		default:
			err := retry.Do(
				func() error {
					result := c.checkBatchAvailability(daMetaData)
					availabilityResult = result

					if result.Code != da.StatusSuccess {
						c.logger.Error("Blob submitted not found in DA. Retrying availability check.", "error", result.Message)
						return da.ErrBlobNotFound
					}

					return nil
				},
				retry.Attempts(uint(*c.config.RetryAttempts)), //nolint:gosec // RetryAttempts should be always positive
				retry.DelayType(retry.FixedDelay),
				retry.Delay(c.config.RetryDelay),
			)
			if err != nil {
				c.logger.Error("CheckAvailability process failed.", "error", err)
			}
			return availabilityResult
		}
	}
}

// GetMaxBlobSizeBytes returns the maximum allowed blob size in the DA, used to check the max batch size configured
func (c *DataAvailabilityLayerClient) GetMaxBlobSizeBytes() uint64 {
	return maxBlobSizeBytes
}

// getMaxBlobSizeBytes returns the maximum allowed blob size from celestia rpc
func (c *DataAvailabilityLayerClient) getMaxBlobSizeBytes() (uint64, error) {
	return c.client.MaxBlobSize(c.ctx)
}

// GetSignerBalance returns the balance for a specific address
func (c *DataAvailabilityLayerClient) GetSignerBalance() (da.Balance, error) {
	ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
	defer cancel()

	balance, err := c.client.Balance(ctx)
	if err != nil {
		return da.Balance{}, fmt.Errorf("get balance: %w", err)
	}

	daBalance := da.Balance{
		Amount: balance.Amount,
		Denom:  balance.Denom,
	}

	return daBalance, nil
}

func makeID(height uint64, commitment da.Commitment) goDA.ID {
	id := make([]byte, heightLen+len(commitment))
	binary.LittleEndian.PutUint64(id, height)
	copy(id[heightLen:], commitment)
	return id
}

func splitID(id goDA.ID) (uint64, da.Commitment) {
	if len(id) <= heightLen {
		return 0, nil
	}
	commitment := id[heightLen:]
	return binary.LittleEndian.Uint64(id[:heightLen]), commitment
}

func (c *DataAvailabilityLayerClient) checkBatchAvailability(daMetaData *da.DASubmitMetaData) da.ResultCheckBatch {
	ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
	defer cancel()
	DACheckMetaData := &da.DACheckMetaData{
		Client:     daMetaData.Client,
		Height:     daMetaData.Height,
		Commitment: daMetaData.Commitment,
		Namespace:  daMetaData.Namespace,
	}

	dah, err := c.client.GetByHeight(ctx, daMetaData.Height)
	if err != nil {
		// Returning Data Availability header Data Root for dispute validation
		return da.ResultCheckBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("Error getting DAHeader: %s", err),
				Error:   da.ErrUnableToGetProof,
			},
			CheckMetaData: DACheckMetaData,
		}
	}

	DACheckMetaData.Root = dah.DAH.Hash()

	ids := []goDA.ID{makeID(daMetaData.Height, daMetaData.Commitment)}
	daProofs, err := c.client.GetProofs(ctx, ids, c.config.NamespaceID.Bytes())
	if err != nil || daProofs[0] == nil {
		// TODO (srene): Not getting proof means there is no existing data for the namespace and the commitment (the commitment is wrong).
		// Therefore we need to prove whether the commitment is wrong or the span does not exists.
		// In case the span is correct it is necessary to return the data for the span and the proofs to the data root, so we can prove the data
		// is the data for the span, and reproducing the commitment will generate a different one.
		return da.ResultCheckBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("Error getting NMT proof: %s", err),
				Error:   da.ErrUnableToGetProof,
			},
			CheckMetaData: DACheckMetaData,
		}
	}

	var nmtProofs [][]*nmt.Proof
	for _, daProof := range daProofs {
		blobProof := &[]nmt.Proof{}
		err := json.Unmarshal(daProof, blobProof)
		if err != nil {
			return da.ResultCheckBatch{
				BaseResult: da.BaseResult{
					Code:    da.StatusError,
					Message: fmt.Sprintf("Error parsing NMT proof: %s", err),
					Error:   da.ErrUnableToGetProof,
				},
				CheckMetaData: DACheckMetaData,
			}
		}
		var nmtProof []*nmt.Proof
		for _, prf := range *blobProof {
			nmtProof = append(nmtProof, &prf)
		}
		nmtProofs = append(nmtProofs, nmtProof)

	}
	shares := 0
	index := 0
	for j, proof := range nmtProofs[0] {
		if j == 0 {
			index = proof.Start()
		}
		shares += proof.End() - proof.Start()
	}

	if daMetaData.Index > 0 && daMetaData.Length > 0 {
		if index != daMetaData.Index || shares != daMetaData.Length {
			// TODO (srene): In this case the commitment is correct but does not match the span.
			// If the span is correct we have to repeat the previous step (sending data + proof of data)
			// In case the span is not correct we need to send unavailable proof by sending proof of any row root to data root
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
	included, err := c.client.Validate(ctx, ids, daProofs, c.config.NamespaceID.Bytes())
	// included, err = c.validateProof(daMetaData, proof)
	// The both cases below (there is an error validating the proof or the proof is wrong) should not happen
	// if we consider correct functioning of the celestia light node.
	// This will only happen in case the previous step the celestia light node returned wrong proofs..
	if err != nil {
		return da.ResultCheckBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: "Error validating proof",
				Error:   err,
			},
			CheckMetaData: DACheckMetaData,
		}
	} else if !included[0] {
		return da.ResultCheckBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: "Blob not included",
				Error:   da.ErrBlobNotIncluded,
			},
			CheckMetaData: DACheckMetaData,
		}
	}

	DACheckMetaData.Index = index
	DACheckMetaData.Length = shares
	DACheckMetaData.Proofs = nmtProofs
	return da.ResultCheckBatch{
		BaseResult: da.BaseResult{
			Code:    da.StatusSuccess,
			Message: "Blob available",
		},
		CheckMetaData: DACheckMetaData,
	}
}

func (c *DataAvailabilityLayerClient) retrieveBatches(daMetaData *da.DASubmitMetaData) da.ResultRetrieveBatch {
	ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
	defer cancel()

	c.logger.Debug("Getting blob from DA.", "height", daMetaData.Height, "namespace", hex.EncodeToString(daMetaData.Namespace), "commitment", hex.EncodeToString(daMetaData.Commitment))
	var batches []*types.Batch

	id := makeID(daMetaData.Height, daMetaData.Commitment)
	blob, err := c.client.Get(ctx, []goDA.ID{id}, c.config.NamespaceID.Bytes())
	if err != nil {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
				Error:   errorsmod.Wrap(da.ErrRetrieval, err.Error()),
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
	err = proto.Unmarshal(blob[0], &batch)
	if err != nil {
		c.logger.Error("Unmarshal blob.", "daHeight", daMetaData.Height, "error", err)
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
				Error:   da.ErrBlobNotParsed,
			},
		}
	}

	c.logger.Debug("Blob retrieved successfully from DA.", "DA height", daMetaData.Height, "lastBlockHeight", batch.EndHeight)

	parsedBatch := new(types.Batch)
	err = parsedBatch.FromProto(&batch)
	if err != nil {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
				Error:   da.ErrBlobNotParsed,
			},
		}
	}
	batches = append(batches, parsedBatch)
	return da.ResultRetrieveBatch{
		BaseResult: da.BaseResult{
			Code:    da.StatusSuccess,
			Message: "Batch retrieval successful",
		},
		Batches: batches,
	}
}
