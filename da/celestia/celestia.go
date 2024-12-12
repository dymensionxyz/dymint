package celestia

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/celestiaorg/celestia-openrpc/types/blob"
	"github.com/celestiaorg/celestia-openrpc/types/header"
	"github.com/celestiaorg/nmt"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	"github.com/gogo/protobuf/proto"
	"github.com/tendermint/tendermint/libs/pubsub"

	openrpc "github.com/celestiaorg/celestia-openrpc"

	"github.com/dymensionxyz/dymint/da"
	celtypes "github.com/dymensionxyz/dymint/da/celestia/types"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
	pb "github.com/dymensionxyz/dymint/types/pb/dymint"
	uretry "github.com/dymensionxyz/dymint/utils/retry"
)


type DataAvailabilityLayerClient struct {
	rpc celtypes.CelestiaRPCClient

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


func WithRPCClient(rpc celtypes.CelestiaRPCClient) da.Option {
	return func(daLayerClient da.DataAvailabilityLayerClient) {
		daLayerClient.(*DataAvailabilityLayerClient).rpc = rpc
	}
}


func WithRPCRetryDelay(delay time.Duration) da.Option {
	return func(daLayerClient da.DataAvailabilityLayerClient) {
		daLayerClient.(*DataAvailabilityLayerClient).config.RetryDelay = delay
	}
}


func WithRPCAttempts(attempts int) da.Option {
	return func(daLayerClient da.DataAvailabilityLayerClient) {
		daLayerClient.(*DataAvailabilityLayerClient).config.RetryAttempts = &attempts
	}
}


func WithSubmitBackoff(c uretry.BackoffConfig) da.Option {
	return func(daLayerClient da.DataAvailabilityLayerClient) {
		daLayerClient.(*DataAvailabilityLayerClient).config.Backoff = c
	}
}


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

	
	for _, apply := range options {
		apply(c)
	}

	types.RollappConsecutiveFailedDASubmission.Set(0)

	return nil
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


func (c *DataAvailabilityLayerClient) Start() (err error) {
	c.logger.Info("Starting Celestia Data Availability Layer Client.")

	
	if c.rpc != nil {
		c.logger.Info("Celestia-node client already set.")
		return nil
	}

	var rpc *openrpc.Client
	rpc, err = openrpc.NewClient(c.ctx, c.config.BaseURL, c.config.AuthToken)
	if err != nil {
		return err
	}
	c.rpc = NewOpenRPC(rpc)

	go c.sync(rpc)

	return
}


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


func (m *DataAvailabilityLayerClient) WaitForSyncing() {
	<-m.synced
}


func (c *DataAvailabilityLayerClient) GetClientType() da.Client {
	return da.Celestia
}


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
				Message: fmt.Sprintf("size bigger than maximum blob size: max n bytes: %d", celtypes.DefaultMaxBytes),
				Error:   errors.New("blob size too big"),
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

			
			height, commitment, err := c.submit(data)
			if errors.Is(err, gerrc.ErrInternal) {
				
				err = fmt.Errorf("submit: %w", err)
				return da.ResultSubmitBatch{
					BaseResult: da.BaseResult{
						Code:    da.StatusError,
						Message: err.Error(),
						Error:   err,
					},
				}
			}

			if err != nil {
				c.logger.Error("Submit blob.", "error", err)
				types.RollappConsecutiveFailedDASubmission.Inc()
				backoff.Sleep()
				continue
			}

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
				types.RollappConsecutiveFailedDASubmission.Inc()
				backoff.Sleep()
				continue
			}
			daMetaData.Root = result.CheckMetaData.Root
			daMetaData.Index = result.CheckMetaData.Index
			daMetaData.Length = result.CheckMetaData.Length

			c.logger.Debug("Blob availability check passed successfully.")

			types.RollappConsecutiveFailedDASubmission.Set(0)
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
				retry.Attempts(uint(*c.config.RetryAttempts)), 
				retry.DelayType(retry.FixedDelay),
				retry.Delay(c.config.RetryDelay),
			)
			if err != nil {
				c.logger.Error("Retrieve batch", "height", daMetaData.Height, "commitment", hex.EncodeToString(daMetaData.Commitment))
			}
			return resultRetrieveBatch

		}
	}
}

func (c *DataAvailabilityLayerClient) retrieveBatches(daMetaData *da.DASubmitMetaData) da.ResultRetrieveBatch {
	ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
	defer cancel()

	c.logger.Debug("Getting blob from DA.", "height", daMetaData.Height, "namespace", hex.EncodeToString(daMetaData.Namespace), "commitment", hex.EncodeToString(daMetaData.Commitment))
	var batches []*types.Batch
	blob, err := c.rpc.Get(ctx, daMetaData.Height, daMetaData.Namespace, daMetaData.Commitment)
	if err != nil {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
				Error:   da.ErrRetrieval,
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
						c.logger.Error("Blob submitted not found in DA. Retrying availability check.")
						return da.ErrBlobNotFound
					}

					return nil
				},
				retry.Attempts(uint(*c.config.RetryAttempts)), 
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

func (c *DataAvailabilityLayerClient) checkBatchAvailability(daMetaData *da.DASubmitMetaData) da.ResultCheckBatch {
	var proofs []*blob.Proof

	DACheckMetaData := &da.DACheckMetaData{
		Client:     daMetaData.Client,
		Height:     daMetaData.Height,
		Commitment: daMetaData.Commitment,
		Namespace:  daMetaData.Namespace,
	}

	dah, err := c.getDataAvailabilityHeaders(daMetaData.Height)
	if err != nil {
		
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

	proof, err := c.getProof(daMetaData)
	if err != nil || proof == nil {
		
		
		
		
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

	included, err = c.validateProof(daMetaData, proof)
	
	
	
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
	return da.ResultCheckBatch{
		BaseResult: da.BaseResult{
			Code:    da.StatusSuccess,
			Message: "Blob available",
		},
		CheckMetaData: DACheckMetaData,
	}
}


func (c *DataAvailabilityLayerClient) submit(daBlob da.Blob) (uint64, da.Commitment, error) {
	blobs, commitments, err := c.blobsAndCommitments(daBlob)
	if err != nil {
		return 0, nil, fmt.Errorf("blobs and commitments: %w: %w", err, gerrc.ErrInternal)
	}

	if len(commitments) == 0 {
		return 0, nil, fmt.Errorf("zero commitments: %w: %w", gerrc.ErrNotFound, gerrc.ErrInternal)
	}

	ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
	defer cancel()

	height, err := c.rpc.Submit(ctx, blobs, blob.NewSubmitOptions(blob.WithGasPrice(c.config.GasPrices)))
	if err != nil {
		return 0, nil, fmt.Errorf("do rpc submit: %w", err)
	}

	return height, commitments[0], nil
}

func (c *DataAvailabilityLayerClient) getProof(daMetaData *da.DASubmitMetaData) (*blob.Proof, error) {
	c.logger.Debug("Getting proof via RPC call.", "height", daMetaData.Height, "namespace", daMetaData.Namespace, "commitment", daMetaData.Commitment)
	ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
	defer cancel()

	proof, err := c.rpc.GetProof(ctx, daMetaData.Height, daMetaData.Namespace, daMetaData.Commitment)
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

	commitments = append(commitments, b.Commitment)
	return blobs, commitments, nil
}

func (c *DataAvailabilityLayerClient) validateProof(daMetaData *da.DASubmitMetaData, proof *blob.Proof) (bool, error) {
	c.logger.Debug("Validating proof via RPC call.", "height", daMetaData.Height, "namespace", daMetaData.Namespace, "commitment", daMetaData.Commitment)
	ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
	defer cancel()

	return c.rpc.Included(ctx, daMetaData.Height, daMetaData.Namespace, proof, daMetaData.Commitment)
}

func (c *DataAvailabilityLayerClient) getDataAvailabilityHeaders(height uint64) (*header.DataAvailabilityHeader, error) {
	c.logger.Debug("Getting extended headers via RPC call.", "height", height)
	ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
	defer cancel()

	headers, err := c.rpc.GetByHeight(ctx, height)
	if err != nil {
		return nil, err
	}

	return headers.DAH, nil
}


func (c *DataAvailabilityLayerClient) sync(rpc *openrpc.Client) {
	sync := func() error {
		done := make(chan error, 1)
		go func() {
			done <- rpc.Header.SyncWait(c.ctx)
		}()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case err := <-done:
				return err
			case <-ticker.C:
				state, err := rpc.Header.SyncState(c.ctx)
				if err != nil {
					return err
				}
				c.logger.Info("Celestia-node syncing", "height", state.Height, "target", state.ToHeight)

			}
		}
	}

	err := retry.Do(sync,
		retry.Attempts(0), 
		retry.Delay(10*time.Second),
		retry.LastErrorOnly(true),
		retry.DelayType(retry.FixedDelay),
		retry.OnRetry(func(n uint, err error) {
			c.logger.Error("Failed to sync Celestia DA", "attempt", n, "error", err)
		}),
	)

	c.logger.Info("Celestia-node is synced.")
	c.synced <- struct{}{}

	if err != nil {
		c.logger.Error("Waiting for Celestia data availability client to sync", "err", err)
	}
}


func (d *DataAvailabilityLayerClient) GetMaxBlobSizeBytes() uint32 {
	return maxBlobSizeBytes
}


func (d *DataAvailabilityLayerClient) GetSignerBalance() (da.Balance, error) {
	ctx, cancel := context.WithTimeout(d.ctx, d.config.Timeout)
	defer cancel()

	balance, err := d.rpc.GetSignerBalance(ctx)
	if err != nil {
		return da.Balance{}, fmt.Errorf("get balance: %w", err)
	}

	daBalance := da.Balance{
		Amount: balance.Amount,
		Denom:  balance.Denom,
	}

	return daBalance, nil
}
