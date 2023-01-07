package testutil

import (
	"github.com/dymensionxyz/dymint/mocks"
	"github.com/stretchr/testify/mock"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/proxy"
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
func GetAppMock() *mocks.Application {
	app := &mocks.Application{}
	app.On("InitChain", mock.Anything).Return(abci.ResponseInitChain{})
	app.On("CheckTx", mock.Anything).Return(abci.ResponseCheckTx{})
	app.On("BeginBlock", mock.Anything).Return(abci.ResponseBeginBlock{})
	app.On("DeliverTx", mock.Anything).Return(abci.ResponseDeliverTx{})
	app.On("EndBlock", mock.Anything).Return(abci.ResponseEndBlock{})
	app.On("Commit", mock.Anything).Return(abci.ResponseCommit{})
	app.On("Info", mock.Anything).Return(abci.ResponseInfo{LastBlockHeight: 0, LastBlockAppHash: []byte{0}})

	return app
}
