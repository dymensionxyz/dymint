package testutil

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/celestiaorg/celestia-openrpc/types/blob"
	"github.com/celestiaorg/celestia-openrpc/types/header"
	"github.com/celestiaorg/nmt"
	"github.com/stretchr/testify/mock"
	abci "github.com/tendermint/tendermint/abci/types"
	tmjson "github.com/tendermint/tendermint/libs/json"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/proxy"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/celestia"
	localda "github.com/dymensionxyz/dymint/da/local"
	"github.com/dymensionxyz/dymint/da/registry"
	damocks "github.com/dymensionxyz/dymint/mocks/github.com/dymensionxyz/dymint/da/celestia/types"
	tmmocks "github.com/dymensionxyz/dymint/mocks/github.com/tendermint/tendermint/abci/types"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
	rollapptypes "github.com/dymensionxyz/dymint/types/pb/dymensionxyz/dymension/rollapp"
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
func GetAppMock(excludeMethods ...ABCIMethod) *tmmocks.MockApplication {
	app := &tmmocks.MockApplication{}
	gbdBz, _ := tmjson.Marshal(rollapptypes.GenesisBridgeData{})
	app.On("InitChain", mock.Anything).Return(abci.ResponseInitChain{GenesisBridgeDataBytes: gbdBz})
	app.On("CheckTx", mock.Anything).Return(abci.ResponseCheckTx{})
	app.On("BeginBlock", mock.Anything).Return(abci.ResponseBeginBlock{})
	app.On("DeliverTx", mock.Anything).Return(abci.ResponseDeliverTx{})
	app.On("EndBlock", mock.Anything).Return(abci.ResponseEndBlock{})
	app.On("Commit", mock.Anything).Return(abci.ResponseCommit{})
	app.On("Info", mock.Anything).Return(abci.ResponseInfo{LastBlockHeight: 0, LastBlockAppHash: []byte{0}})

	// iterate exclude methods and unset the mock
	for _, method := range excludeMethods {
		UnsetMockFn(app.On(string(method)))
	}

	return app
}

var UnsetMockFn = func(call *mock.Call) {
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

// CountMockCalls returns the number of times a mock specific function was called
func CountMockCalls(totalCalls []mock.Call, methodName string) int {
	var count int
	for _, call := range totalCalls {
		if call.Method == methodName {
			count++
		}
	}
	return count
}

// MockStore is a mock store for testing
type MockStore struct {
	ShoudFailSaveState             bool
	ShouldFailUpdateStateWithBatch bool
	*store.DefaultStore
	height uint64
}

// SetHeight sets the height of the mock store
// Don't set the height to mock failure in setting the height
func (m *MockStore) SetHeight(height uint64) {
	m.height = height
}

func (m *MockStore) Height() uint64 {
	return m.height
}

func (m *MockStore) NextHeight() uint64 {
	return m.height + 1
}

// UpdateState updates the state of the mock store
func (m *MockStore) SaveState(state *types.State, batch store.KVBatch) (store.KVBatch, error) {
	if batch != nil && m.ShouldFailUpdateStateWithBatch || m.ShoudFailSaveState && batch == nil {
		return nil, errors.New("failed to update state")
	}
	return m.DefaultStore.SaveState(state, batch)
}

// NewMockStore returns a new mock store
func NewMockStore() *MockStore {
	defaultStore := store.New(store.NewDefaultInMemoryKVStore())
	return &MockStore{
		DefaultStore:       defaultStore,
		height:             0,
		ShoudFailSaveState: false,
	}
}

const (
	batchNotFoundErrorMessage     = "batch not found"
	connectionRefusedErrorMessage = "connection refused"
)

// DALayerClientSubmitBatchError is a mock data availability layer client that can be used to test error handling
type DALayerClientSubmitBatchError struct {
	localda.DataAvailabilityLayerClient
}

// SubmitBatch submits a batch to the data availability layer
func (s *DALayerClientSubmitBatchError) SubmitBatch(_ *types.Batch) da.ResultSubmitBatch {
	return da.ResultSubmitBatch{BaseResult: da.BaseResult{Code: da.StatusError, Message: connectionRefusedErrorMessage, Error: errors.New(connectionRefusedErrorMessage)}}
}

// DALayerClientRetrieveBatchesError is a mock data availability layer client that can be used to test error handling
type DALayerClientRetrieveBatchesError struct {
	localda.DataAvailabilityLayerClient
}

// RetrieveBatches retrieves batches from the data availability layer
func (m *DALayerClientRetrieveBatchesError) RetrieveBatches(_ *da.DASubmitMetaData) da.ResultRetrieveBatch {
	return da.ResultRetrieveBatch{BaseResult: da.BaseResult{Code: da.StatusError, Message: batchNotFoundErrorMessage, Error: da.ErrBlobNotFound}}
}

// SubscribeMock is a mock to provide a subscription like behavior for testing
type SubscribeMock struct {
	messageCh chan interface{}
}

func NewSubscribeMock(messageCh chan interface{}) *SubscribeMock {
	return &SubscribeMock{messageCh: make(chan interface{})}
}

func (s *SubscribeMock) Chan() <-chan interface{} {
	return s.messageCh
}

func (s *SubscribeMock) Unsubscribe() {
	close(s.messageCh)
}

type MockDA struct {
	DaClient  da.DataAvailabilityLayerClient
	MockRPC   *damocks.MockCelestiaRPCClient
	NID       []byte
	Header    *header.ExtendedHeader
	BlobProof blob.Proof
}

func NewMockDA(t *testing.T) (*MockDA, error) {
	mockDA := &MockDA{}
	// Create DA
	// init celestia DA with mock RPC client
	mockDA.DaClient = registry.GetClient("celestia")

	config := celestia.Config{
		BaseURL:        "http://localhost:26658",
		Timeout:        30 * time.Second,
		GasPrices:      celestia.DefaultGasPrices,
		NamespaceIDStr: "0000000000000000ffff",
	}
	err := config.InitNamespaceID()
	if err != nil {
		return nil, err
	}
	conf, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	mockDA.MockRPC = damocks.NewMockCelestiaRPCClient(t)
	options := []da.Option{
		celestia.WithRPCClient(mockDA.MockRPC),
		celestia.WithRPCAttempts(1),
		celestia.WithRPCRetryDelay(time.Second * 2),
	}
	roots := [][]byte{[]byte("apple"), []byte("watermelon"), []byte("kiwi")}
	dah := &header.DataAvailabilityHeader{
		RowRoots:    roots,
		ColumnRoots: roots,
	}
	mockDA.Header = &header.ExtendedHeader{
		DAH: dah,
	}

	mockDA.NID = config.NamespaceID.Bytes()

	nIDSize := 1
	tree := exampleNMT(nIDSize, true, 1, 2, 3, 4)
	// build a proof for an NID that is within the namespace range of the tree
	proof, _ := tree.ProveNamespace(mockDA.NID)
	mockDA.BlobProof = blob.Proof([]*nmt.Proof{&proof})

	err = mockDA.DaClient.Init(conf, nil, store.NewDefaultInMemoryKVStore(), log.TestingLogger(), options...)
	if err != nil {
		return nil, err
	}
	return mockDA, nil
}

// exampleNMT creates a new NamespacedMerkleTree with the given namespace ID size and leaf namespace IDs. Each byte in the leavesNIDs parameter corresponds to one leaf's namespace ID. If nidSize is greater than 1, the function repeats each NID in leavesNIDs nidSize times before prepending it to the leaf data.
func exampleNMT(nidSize int, ignoreMaxNamespace bool, leavesNIDs ...byte) *nmt.NamespacedMerkleTree {
	tree := nmt.New(sha256.New(), nmt.NamespaceIDSize(nidSize), nmt.IgnoreMaxNamespace(ignoreMaxNamespace))
	for i, nid := range leavesNIDs {
		namespace := bytes.Repeat([]byte{nid}, nidSize)
		d := append(namespace, []byte(fmt.Sprintf("leaf_%d", i))...)
		if err := tree.Push(d); err != nil {
			panic(fmt.Sprintf("unexpected error: %v", err))
		}
	}
	return tree
}
