package testutil

import (
	"github.com/dymensionxyz/dymint/mocks"
	"github.com/stretchr/testify/mock"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/proxy"
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
