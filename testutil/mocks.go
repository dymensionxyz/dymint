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


type ABCIMethod string

const (
	
	InitChain ABCIMethod = "InitChain"
	
	CheckTx ABCIMethod = "CheckTx"
	
	BeginBlock ABCIMethod = "BeginBlock"
	
	DeliverTx ABCIMethod = "DeliverTx"
	
	EndBlock ABCIMethod = "EndBlock"
	
	Commit ABCIMethod = "Commit"
	
	Info ABCIMethod = "Info"
)


func GetABCIProxyAppMock(logger log.Logger) proxy.AppConns {
	app := GetAppMock()

	clientCreator := proxy.NewLocalClientCreator(app)
	proxyApp := proxy.NewAppConns(clientCreator)
	proxyApp.SetLogger(logger)

	return proxyApp
}


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


func CountMockCalls(totalCalls []mock.Call, methodName string) int {
	var count int
	for _, call := range totalCalls {
		if call.Method == methodName {
			count++
		}
	}
	return count
}


type MockStore struct {
	ShoudFailSaveState             bool
	ShouldFailUpdateStateWithBatch bool
	*store.DefaultStore
	height uint64
}



func (m *MockStore) SetHeight(height uint64) {
	m.height = height
}

func (m *MockStore) Height() uint64 {
	return m.height
}

func (m *MockStore) NextHeight() uint64 {
	return m.height + 1
}


func (m *MockStore) SaveState(state *types.State, batch store.KVBatch) (store.KVBatch, error) {
	if batch != nil && m.ShouldFailUpdateStateWithBatch || m.ShoudFailSaveState && batch == nil {
		return nil, errors.New("failed to update state")
	}
	return m.DefaultStore.SaveState(state, batch)
}


func NewMockStore() *MockStore {
	defaultStore := store.New(store.NewDefaultInMemoryKVStore())
	return &MockStore{
		DefaultStore:       defaultStore.(*store.DefaultStore),
		height:             0,
		ShoudFailSaveState: false,
	}
}

const (
	batchNotFoundErrorMessage     = "batch not found"
	connectionRefusedErrorMessage = "connection refused"
)


type DALayerClientSubmitBatchError struct {
	localda.DataAvailabilityLayerClient
}


func (s *DALayerClientSubmitBatchError) SubmitBatch(_ *types.Batch) da.ResultSubmitBatch {
	return da.ResultSubmitBatch{BaseResult: da.BaseResult{Code: da.StatusError, Message: connectionRefusedErrorMessage, Error: errors.New(connectionRefusedErrorMessage)}}
}


type DALayerClientRetrieveBatchesError struct {
	localda.DataAvailabilityLayerClient
}


func (m *DALayerClientRetrieveBatchesError) RetrieveBatches(_ *da.DASubmitMetaData) da.ResultRetrieveBatch {
	return da.ResultRetrieveBatch{BaseResult: da.BaseResult{Code: da.StatusError, Message: batchNotFoundErrorMessage, Error: da.ErrBlobNotFound}}
}


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
	
	proof, _ := tree.ProveNamespace(mockDA.NID)
	mockDA.BlobProof = blob.Proof([]*nmt.Proof{&proof})

	err = mockDA.DaClient.Init(conf, nil, store.NewDefaultInMemoryKVStore(), log.TestingLogger(), options...)
	if err != nil {
		return nil, err
	}
	return mockDA, nil
}


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
