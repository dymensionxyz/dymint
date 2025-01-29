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

// heightLen is a length (in bytes) of serialized height. used in daclient.ID to translate ID to height+commitment
const heightLen = 8

// DataAvailabilityLayerClient use celestia-node public API.
type DataAvailabilityLayerClient struct {
	client       daclient.DAClient
	pubsubServer *pubsub.Server
	config       Config
	logger       types.Logger
	ctx          context.Context
	cancel       context.CancelFunc
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
		return nil
	}

	client, err := daclient.NewClient(c.config.BaseURL, c.config.AuthToken)
	if err != nil {
		return fmt.Errorf("error while establishing connection to DA layer: %w", err)
	}
	c.client = client

	return
}

// Stop stops DataAvailabilityLayerClient.
func (c *DataAvailabilityLayerClient) Stop() error {
	c.logger.Info("Stopping Celestia Data Availability Layer Client.")
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

	backoff := c.config.Backoff.Backoff()

	for {
		select {
		case <-c.ctx.Done():
			c.logger.Debug("Context cancelled.")
			return da.ResultSubmitBatch{}
		default:

			daMetaData, err := c.submit(data)
			if err != nil {
				c.logger.Error("Submit blob.", "error", err)
				metrics.RollappConsecutiveFailedDASubmission.Inc()
				backoff.Sleep()
				continue
			}

			c.logger.Debug("Submitted blob to DA successfully.")

			checkMetadata, err := c.checkBatchAvailability(daMetaData)
			if err != nil {
				c.logger.Error("Check batch availability: submitted batch but did not get availability success status.", "error", err)
				metrics.RollappConsecutiveFailedDASubmission.Inc()
				backoff.Sleep()
				continue
			}
			daMetaData.Root = checkMetadata.Root
			daMetaData.Index = checkMetadata.Index
			daMetaData.Length = checkMetadata.Length

			c.logger.Debug("Blob availability check passed successfully.")

			metrics.RollappConsecutiveFailedDASubmission.Set(0)
			return da.ResultSubmitBatch{
				BaseResult: da.BaseResult{
					Code:    da.StatusSuccess,
					Message: "Submission successful",
				},
				SubmitMetaData: &da.DASubmitMetaData{
					Client: da.Celestia,
					DAPath: daMetaData.ToPath(),
				},
			}
		}
	}
}

// RetrieveBatches downloads a batch from celestia, defined by daMetadata, and retries RetryAttempts in case of failure
func (c *DataAvailabilityLayerClient) RetrieveBatches(daPath string) da.ResultRetrieveBatch {
	submitMetadata := &SubmitMetaData{}
	daMetaData, err := submitMetadata.FromPath(string(daPath))
	if err != nil {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: "Unable to get DA metadata",
			},
		}
	}
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

// CheckBatchAvailability retrieves availability proofs from celestia for the blob defined in daMetaData, and validates its inclusion
func (c *DataAvailabilityLayerClient) CheckBatchAvailability(daPath string) da.ResultCheckBatch {

	submitMetadata := &SubmitMetaData{}
	daMetaData, err := submitMetadata.FromPath(string(daPath))
	if err != nil {
		return da.ResultCheckBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: "Unable to get DA metadata",
			},
		}
	}
	for {
		select {
		case <-c.ctx.Done():
			c.logger.Debug("Context cancelled")
			return da.ResultCheckBatch{
				BaseResult: da.BaseResult{
					Code:    da.StatusError,
					Message: "context cancelled",
				},
			}
		default:
			err := retry.Do(
				func() error {
					_, err = c.checkBatchAvailability(daMetaData)
					return err
				},
				retry.Attempts(uint(*c.config.RetryAttempts)), //nolint:gosec // RetryAttempts should be always positive
				retry.DelayType(retry.FixedDelay),
				retry.Delay(c.config.RetryDelay),
			)
			if err != nil {
				c.logger.Error("CheckBatchAvailability batch", "height", daMetaData.Height, "commitment", hex.EncodeToString(daMetaData.Commitment), "error", err)
				return da.ResultCheckBatch{
					BaseResult: da.BaseResult{
						Code:    da.StatusError,
						Message: "Blob not available",
					},
				}
			}

			return da.ResultCheckBatch{
				BaseResult: da.BaseResult{
					Code:    da.StatusSuccess,
					Message: "Blob available",
				},
			}
		}
	}
}

// GetMaxBlobSizeBytes returns the maximum allowed blob size in the DA, used to check the max batch size configured
func (c *DataAvailabilityLayerClient) GetMaxBlobSizeBytes() uint64 {
	return maxBlobSizeBytes
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

// submit submits a blob to celestia, including data bytes.
func (c *DataAvailabilityLayerClient) submit(data []byte) (*SubmitMetaData, error) {
	// TODO(srene):  Split batch in multiple blobs if necessary when supported
	ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
	defer cancel()
	ids, err := c.client.Submit(ctx, []da.Blob{data}, c.config.GasPrices, c.config.NamespaceID.Bytes())
	if err != nil {
		return &SubmitMetaData{}, err
	}
	height, commitment := splitID(ids[0])

	return &SubmitMetaData{
		Height:     height,
		Commitment: commitment,
		Namespace:  c.config.NamespaceID.Bytes(),
	}, nil
}

// checkBatchAvailability gets da availability data from celestia and validates blob inclusion with it
func (c *DataAvailabilityLayerClient) checkBatchAvailability(daMetaData *SubmitMetaData) (*CheckMetaData, error) {
	daCheckMetaData, err := c.getDAAvailabilityMetaData(daMetaData)
	if err != nil {
		return &CheckMetaData{}, err
	}

	err = c.validateInclusion(daCheckMetaData)
	if err != nil {
		return &CheckMetaData{}, err
	}

	return daCheckMetaData, nil
}

// retrieveBatches downloads a blob from celestia and returns the batch included
func (c *DataAvailabilityLayerClient) retrieveBatches(daMetaData *SubmitMetaData) da.ResultRetrieveBatch {
	ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
	defer cancel()

	c.logger.Debug("Getting blob from DA.", "height", daMetaData.Height, "namespace", hex.EncodeToString(daMetaData.Namespace), "commitment", hex.EncodeToString(daMetaData.Commitment))
	var batches []*types.Batch

	id := makeID(daMetaData.Height, daMetaData.Commitment)
	blob, err := c.client.Get(ctx, []daclient.ID{id}, c.config.NamespaceID.Bytes())
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
				Message: "blob retrieved is nil",
				Error:   da.ErrBlobNotFound,
			},
		}
	}

	var batch pb.Batch
	err = proto.Unmarshal(blob[0], &batch)
	if err != nil {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("Error unmarshaling batch: %s", err),
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

// getDataRoot returns celestia data root included in the extended headers
func (c *DataAvailabilityLayerClient) getDataRoot(daMetaData *SubmitMetaData) ([]byte, error) {
	ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
	defer cancel()
	dah, err := c.client.GetByHeight(ctx, daMetaData.Height)
	if err != nil {
		return nil, err
	}
	return dah.DAH.Hash(), nil
}

// getDAAvailabilityMetaData returns the da metadata (span + proofs) necessary to check blob inclusion
func (c *DataAvailabilityLayerClient) getDAAvailabilityMetaData(daMetaData *SubmitMetaData) (*CheckMetaData, error) {
	ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
	defer cancel()

	ids := []daclient.ID{makeID(daMetaData.Height, daMetaData.Commitment)}
	daProofs, err := c.client.GetProofs(ctx, ids, c.config.NamespaceID.Bytes())
	if err != nil || daProofs[0] == nil {
		// TODO (srene): Not getting proof means there is no existing data for the namespace and commitment (the commitment is not valid).
		// Therefore we need to prove whether the commitment is wrong or the span does not exists.
		// In case the span is valid (within the size of the matrix) it is necessary to return the data for the span and the proofs to the data root, so we can prove the data
		// is the data for the span, and reproducing the commitment will generate a different one.
		return &CheckMetaData{}, err
	}

	var nmtProofs [][]*nmt.Proof
	for _, daProof := range daProofs {
		blobProof := &[]nmt.Proof{}
		err := json.Unmarshal(daProof, blobProof)
		if err != nil {
			return &CheckMetaData{}, err
		}
		var nmtProof []*nmt.Proof
		for _, prf := range *blobProof {
			nmtProof = append(nmtProof, &prf)
		}
		nmtProofs = append(nmtProofs, nmtProof)

	}
	sharesNum := 0
	index := 0
	for j, proof := range nmtProofs[0] {
		if j == 0 {
			index = proof.Start()
		}
		sharesNum += proof.End() - proof.Start()
	}

	if daMetaData.Index > 0 && daMetaData.Length > 0 {
		if index != daMetaData.Index || sharesNum != daMetaData.Length {
			// In this case the commitment is valid but does not match the span.
			// TODO (srene): In this case the commitment is correct but does not match the span.
			// If the span is valid we have to repeat the previous step (sending data + proof of data as inclusion proof)
			// In case the span is not valid (out of the size of the matrix) we need to send unavailable proof (the span does not exist) by sending proof of any row root to data root
			return &CheckMetaData{}, fmt.Errorf("span does not match blob commitment")
		}
	}

	root, err := c.getDataRoot(daMetaData)
	if err != nil {
		return &CheckMetaData{}, err
	}

	return &CheckMetaData{
		Height:     daMetaData.Height,
		Commitment: daMetaData.Commitment,
		Namespace:  daMetaData.Namespace,
		Root:       root,
		Index:      index,
		Length:     sharesNum,
		Proofs:     daProofs,
	}, nil
}

func (c *DataAvailabilityLayerClient) validateInclusion(daMetaData *CheckMetaData) error {
	ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
	defer cancel()

	ids := []daclient.ID{makeID(daMetaData.Height, daMetaData.Commitment)}
	included, err := c.client.Validate(ctx, ids, daMetaData.Proofs, c.config.NamespaceID.Bytes())
	if err != nil {
		return err
	}
	for _, incl := range included {
		if !incl {
			return fmt.Errorf("blob commitment not included")
		}
	}
	return nil
}

// makeID generates a blob ID, to be used with the light client, defined by height + commitment
func makeID(height uint64, commitment da.Commitment) daclient.ID {
	id := make([]byte, heightLen+len(commitment))
	binary.LittleEndian.PutUint64(id, height)
	copy(id[heightLen:], commitment)
	return id
}

// splitID returns height + commitment from the daclient ID
func splitID(id daclient.ID) (uint64, da.Commitment) {
	if len(id) <= heightLen {
		return 0, nil
	}
	commitment := id[heightLen:]
	return binary.LittleEndian.Uint64(id[:heightLen]), commitment
}
