package client

import (
	"context"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DAClient defines very generic interface for interaction with Data Availability layers.
type DAClient interface {
	// Get returns Blob for each given ID, or an error.
	//
	// Error should be returned if ID is not formatted properly, there is no Blob for given ID or any other client-level
	// error occurred (dropped connection, timeout, etc).
	Get(ctx context.Context, ids []ID, namespace Namespace) ([]Blob, error)

	// GetIDs returns IDs of all Blobs located in DA at given height.
	GetIDs(ctx context.Context, height uint64, namespace Namespace) (*GetIDsResult, error)

	// GetProofs returns inclusion Proofs for Blobs specified by their IDs.
	GetProofs(ctx context.Context, ids []ID, namespace Namespace) ([]Proof, error)

	// Commit creates a Commitment for each given Blob.
	Commit(ctx context.Context, blobs []Blob, namespace Namespace) ([]Commitment, error)

	// Submit submits the Blobs to Data Availability layer.
	//
	// This method is synchronous. Upon successful submission to Data Availability layer, it returns the IDs identifying blobs
	// in DA.
	Submit(ctx context.Context, blobs []Blob, gasPrice float64, namespace Namespace) ([]ID, error)

	// SubmitWithOptions submits the Blobs to Data Availability layer.
	//
	// This method is synchronous. Upon successful submission to Data Availability layer, it returns the IDs identifying blobs
	// in DA.
	SubmitWithOptions(ctx context.Context, blobs []Blob, gasPrice float64, namespace Namespace, options []byte) ([]ID, error)

	// Validate validates Commitments against the corresponding Proofs. This should be possible without retrieving the Blobs.
	Validate(ctx context.Context, ids []ID, proofs []Proof, namespace Namespace) ([]bool, error)

	// GetByHeight retrieves celestia extended header by height
	GetByHeight(ctx context.Context, height uint64) (*ExtendedHeader, error)

	// Balance returns the celestia light client account balance
	Balance(context.Context) (*Balance, error)

	// Address returns the celestia light client account address
	Address(ctx context.Context) (Address, error)

	// MaxBlobSize returns the max blob size allowed in celestia
	MaxBlobSize(ctx context.Context) (uint64, error)
}

// Namespace is an optional parameter used to set the location a blob should be
// posted to, for DA layers supporting the functionality.
type Namespace = []byte

// Blob is the data submitted/received from DA interface.
type Blob = []byte

// ID should contain serialized data required by the implementation to find blob in Data Availability layer.
type ID = []byte

// Commitment should contain serialized cryptographic commitment to Blob value.
type Commitment = []byte

// Proof should contain serialized proof of inclusion (publication) of Blob in Data Availability layer.
type Proof = []byte

// Balance is an alias to the Coin type from Cosmos-SDK.
type Balance = sdk.Coin

// Address is an alias to the Address type from Cosmos-SDK.
type Address = sdk.Address

// GetIDsResult holds the result of GetIDs call: IDs and timestamp of corresponding block.
type GetIDsResult struct {
	IDs       []ID
	Timestamp time.Time
}

// API defines the jsonrpc DA service module API
type DAAPI struct {
	Internal struct {
		MaxBlobSize       func(ctx context.Context) (uint64, error)                                     `perm:"read"`
		Get               func(ctx context.Context, ids []ID, ns Namespace) ([]Blob, error)             `perm:"read"`
		GetIDs            func(ctx context.Context, height uint64, ns Namespace) (*GetIDsResult, error) `perm:"read"`
		GetProofs         func(ctx context.Context, ids []ID, ns Namespace) ([]Proof, error)            `perm:"read"`
		Commit            func(ctx context.Context, blobs []Blob, ns Namespace) ([]Commitment, error)   `perm:"read"`
		Validate          func(context.Context, []ID, []Proof, Namespace) ([]bool, error)               `perm:"read"`
		Submit            func(context.Context, []Blob, float64, Namespace) ([]ID, error)               `perm:"write"`
		SubmitWithOptions func(context.Context, []Blob, float64, Namespace, []byte) ([]ID, error)       `perm:"write"`
	}
}

// MaxBlobSize returns the max blob size
func (api *DAAPI) MaxBlobSize(ctx context.Context) (uint64, error) {
	return api.Internal.MaxBlobSize(ctx)
}

// Get returns Blob for each given ID, or an error.
func (api *DAAPI) Get(ctx context.Context, ids []ID, ns Namespace) ([]Blob, error) {
	return api.Internal.Get(ctx, ids, ns)
}

// GetIDs returns IDs of all Blobs located in DA at given height.
func (api *DAAPI) GetIDs(ctx context.Context, height uint64, ns Namespace) (*GetIDsResult, error) {
	return api.Internal.GetIDs(ctx, height, ns)
}

// GetProofs returns inclusion Proofs for Blobs specified by their IDs.
func (api *DAAPI) GetProofs(ctx context.Context, ids []ID, ns Namespace) ([]Proof, error) {
	return api.Internal.GetProofs(ctx, ids, ns)
}

// Commit creates a Commitment for each given Blob.
func (api *DAAPI) Commit(ctx context.Context, blobs []Blob, ns Namespace) ([]Commitment, error) {
	return api.Internal.Commit(ctx, blobs, ns)
}

// Validate validates Commitments against the corresponding Proofs. This should be possible without retrieving the Blobs.
func (api *DAAPI) Validate(ctx context.Context, ids []ID, proofs []Proof, ns Namespace) ([]bool, error) {
	return api.Internal.Validate(ctx, ids, proofs, ns)
}

// Submit submits the Blobs to Data Availability layer.
func (api *DAAPI) Submit(ctx context.Context, blobs []Blob, gasPrice float64, ns Namespace) ([]ID, error) {
	return api.Internal.Submit(ctx, blobs, gasPrice, ns)
}

// SubmitWithOptions submits the Blobs to Data Availability layer.
func (api *DAAPI) SubmitWithOptions(ctx context.Context, blobs []Blob, gasPrice float64, ns Namespace, options []byte) ([]ID, error) {
	return api.Internal.SubmitWithOptions(ctx, blobs, gasPrice, ns, options)
}

// API defines the jsonrpc Header service module API
type HeadersAPI struct {
	Internal struct {
		GetByHeight func(context.Context, uint64) (*ExtendedHeader, error) `perm:"read"`
	}
}

// GetByHeight submits the Blobs to Data Availability layer.
func (api *HeadersAPI) GetByHeight(ctx context.Context, height uint64) (*ExtendedHeader, error) {
	return api.Internal.GetByHeight(ctx, height)
}

// API defines the jsonrpc service module API
type StateAPI struct {
	Internal struct {
		Balance        func(context.Context) (*Balance, error)    `perm:"read"`
		AccountAddress func(ctx context.Context) (Address, error) `perm:"read"`
	}
}

// GetByHeight submits the Blobs to Data Availability layer.
func (api *StateAPI) Balance(ctx context.Context) (*Balance, error) {
	return api.Internal.Balance(ctx)
}

func (api *StateAPI) AccountAddress(ctx context.Context) (Address, error) {
	return api.Internal.AccountAddress(ctx)
}
