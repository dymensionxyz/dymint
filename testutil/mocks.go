package testutil

import (
	"sync/atomic"

	"errors"

	"github.com/dymensionxyz/dymint/mocks"
	"github.com/dymensionxyz/dymint/types"
	"github.com/stretchr/testify/mock"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/proxy"

	"github.com/dymensionxyz/dymint/da"
	mockda "github.com/dymensionxyz/dymint/da/mock"
	"github.com/dymensionxyz/dymint/settlement"
	slmock "github.com/dymensionxyz/dymint/settlement/mock"
	"github.com/dymensionxyz/dymint/store"
)

// ABCIMethod is a string representing an ABCI method
type ABCIMethod string

const (
	// InitChain is the string representation of the InitChain ABCI method
	InitChain ABCIMethod = "InitChain"
	// CheckTx is the string representation of the CheckTx ABCI method
	CheckTx ABCIMethod = "CheckTx"
	// BeginBlock is the string representation of the BeginBlockMethod ABCI method
	BeginBlock ABCIMethod = "BeginBlock"
	// DeliverTx is the string representation of the DeliverTx ABCI method
	DeliverTx ABCIMethod = "DeliverTx"
	// EndBlock is the string representation of the EndBlock ABCI method
	EndBlock ABCIMethod = "EndBlock"
	// Commit is the string representation of the Commit ABCI method
	Commit ABCIMethod = "Commit"
	// Info is the string representation of the Info ABCI method
	Info ABCIMethod = "Info"
)

// GetABCIProxyAppMock returns a dummy abci proxy app mock for testing
func GetABCIProxyAppMock(logger log.Logger) proxy.AppConns {

	app := GetAppMock()

	clientCreator := proxy.NewLocalClientCreator(app)
	proxyApp := proxy.NewAppConns(clientCreator)
	proxyApp.SetLogger(logger)

	return proxyApp
}

// GetAppMock returns a dummy abci app mock for testing
func GetAppMock(excludeMethods ...ABCIMethod) *mocks.Application {
	app := &mocks.Application{}
	app.On("InitChain", mock.Anything).Return(abci.ResponseInitChain{})
	app.On("CheckTx", mock.Anything).Return(abci.ResponseCheckTx{})
	app.On("BeginBlock", mock.Anything).Return(abci.ResponseBeginBlock{})
	app.On("DeliverTx", mock.Anything).Return(abci.ResponseDeliverTx{})
	app.On("EndBlock", mock.Anything).Return(abci.ResponseEndBlock{})
	app.On("Commit", mock.Anything).Return(abci.ResponseCommit{})
	app.On("Info", mock.Anything).Return(abci.ResponseInfo{LastBlockHeight: 0, LastBlockAppHash: []byte{0}})

	// iterate exclude methods and unset the mock
	for _, method := range excludeMethods {
		unsetFn(app.On(string(method)))
	}

	return app
}

var unsetFn = func(call *mock.Call) {
	if call != nil {
		var newList []*mock.Call
		for _, c := range call.Parent.ExpectedCalls {
			if c.Method != call.Method {
				newList = append(newList, c)
			}
		}
		call.Parent.ExpectedCalls = newList
	}
}

// MockStore is a mock store for testing
type MockStore struct {
	ShouldFailSetHeight            bool
	ShoudFailUpdateState           bool
	ShouldFailUpdateStateWithBatch bool
	*store.DefaultStore
	height uint64
}

// SetHeight sets the height of the mock store
// Don't set the height to mock failure in setting the height
func (m *MockStore) SetHeight(height uint64) {
	// Fail the first time
	if m.ShouldFailSetHeight {
		return
	}
	m.height = height
}

// Height returns the height of the mock store
func (m *MockStore) Height() uint64 {
	return m.height
}

// UpdateState updates the state of the mock store
func (m *MockStore) UpdateState(state types.State, batch store.Batch) (store.Batch, error) {
	if batch != nil && m.ShouldFailUpdateStateWithBatch || m.ShoudFailUpdateState && batch == nil {
		return nil, errors.New("failed to update state")
	}
	return m.DefaultStore.UpdateState(state, batch)
}

// NewMockStore returns a new mock store
func NewMockStore() *MockStore {
	defaultStore := store.New(store.NewDefaultInMemoryKVStore())
	return &MockStore{
		DefaultStore:         defaultStore.(*store.DefaultStore),
		height:               0,
		ShouldFailSetHeight:  false,
		ShoudFailUpdateState: false,
	}
}

const batchNotFoundErrorMessage = "batch not found"
const connectionRefusedErrorMessage = "connection refused"

// SettlementLayerClientSubmitBatchError is a mock settlement layer client that can be used to test error handling
type SettlementLayerClientSubmitBatchError struct {
	IsError      atomic.Value
	BatchCounter uint64
	slmock.SettlementLayerClient
}

// SubmitBatch submits a batch to the settlement layer
func (s *SettlementLayerClientSubmitBatchError) SubmitBatch(batch *types.Batch, daClient da.Client, daResultSubmitBatch *da.ResultSubmitBatch) *settlement.ResultSubmitBatch {
	if s.IsError.Load() == true {
		return &settlement.ResultSubmitBatch{
			BaseResult: settlement.BaseResult{Code: settlement.StatusError, Message: connectionRefusedErrorMessage},
		}
	}
	atomic.AddUint64(&s.BatchCounter, 1)
	return &settlement.ResultSubmitBatch{BaseResult: settlement.BaseResult{Code: settlement.StatusSuccess}}
}

// DALayerClientSubmitBatchError is a mock data availability layer client that can be used to test error handling
type DALayerClientSubmitBatchError struct {
	mockda.DataAvailabilityLayerClient
}

// SubmitBatch submits a batch to the data availability layer
func (s *DALayerClientSubmitBatchError) SubmitBatch(_ *types.Batch) da.ResultSubmitBatch {
	return da.ResultSubmitBatch{BaseResult: da.BaseResult{Code: da.StatusError, Message: connectionRefusedErrorMessage}}
}

// DALayerClientRetrieveBatchesError is a mock data availability layer client that can be used to test error handling
type DALayerClientRetrieveBatchesError struct {
	mockda.DataAvailabilityLayerClient
}

// RetrieveBatches retrieves batches from the data availability layer
func (m *DALayerClientRetrieveBatchesError) RetrieveBatches(_ uint64) da.ResultRetrieveBatch {
	return da.ResultRetrieveBatch{BaseResult: da.BaseResult{Code: da.StatusError, Message: batchNotFoundErrorMessage}}
}
