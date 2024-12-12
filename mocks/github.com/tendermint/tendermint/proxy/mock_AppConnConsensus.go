

package proxy

import (
	mock "github.com/stretchr/testify/mock"
	abcicli "github.com/tendermint/tendermint/abci/client"

	types "github.com/tendermint/tendermint/abci/types"
)


type MockAppConnConsensus struct {
	mock.Mock
}

type MockAppConnConsensus_Expecter struct {
	mock *mock.Mock
}

func (_m *MockAppConnConsensus) EXPECT() *MockAppConnConsensus_Expecter {
	return &MockAppConnConsensus_Expecter{mock: &_m.Mock}
}


func (_m *MockAppConnConsensus) BeginBlockSync(_a0 types.RequestBeginBlock) (*types.ResponseBeginBlock, error) {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for BeginBlockSync")
	}

	var r0 *types.ResponseBeginBlock
	var r1 error
	if rf, ok := ret.Get(0).(func(types.RequestBeginBlock) (*types.ResponseBeginBlock, error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func(types.RequestBeginBlock) *types.ResponseBeginBlock); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.ResponseBeginBlock)
		}
	}

	if rf, ok := ret.Get(1).(func(types.RequestBeginBlock) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockAppConnConsensus_BeginBlockSync_Call struct {
	*mock.Call
}



func (_e *MockAppConnConsensus_Expecter) BeginBlockSync(_a0 interface{}) *MockAppConnConsensus_BeginBlockSync_Call {
	return &MockAppConnConsensus_BeginBlockSync_Call{Call: _e.mock.On("BeginBlockSync", _a0)}
}

func (_c *MockAppConnConsensus_BeginBlockSync_Call) Run(run func(_a0 types.RequestBeginBlock)) *MockAppConnConsensus_BeginBlockSync_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.RequestBeginBlock))
	})
	return _c
}

func (_c *MockAppConnConsensus_BeginBlockSync_Call) Return(_a0 *types.ResponseBeginBlock, _a1 error) *MockAppConnConsensus_BeginBlockSync_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockAppConnConsensus_BeginBlockSync_Call) RunAndReturn(run func(types.RequestBeginBlock) (*types.ResponseBeginBlock, error)) *MockAppConnConsensus_BeginBlockSync_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockAppConnConsensus) CommitSync() (*types.ResponseCommit, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for CommitSync")
	}

	var r0 *types.ResponseCommit
	var r1 error
	if rf, ok := ret.Get(0).(func() (*types.ResponseCommit, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() *types.ResponseCommit); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.ResponseCommit)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockAppConnConsensus_CommitSync_Call struct {
	*mock.Call
}


func (_e *MockAppConnConsensus_Expecter) CommitSync() *MockAppConnConsensus_CommitSync_Call {
	return &MockAppConnConsensus_CommitSync_Call{Call: _e.mock.On("CommitSync")}
}

func (_c *MockAppConnConsensus_CommitSync_Call) Run(run func()) *MockAppConnConsensus_CommitSync_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockAppConnConsensus_CommitSync_Call) Return(_a0 *types.ResponseCommit, _a1 error) *MockAppConnConsensus_CommitSync_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockAppConnConsensus_CommitSync_Call) RunAndReturn(run func() (*types.ResponseCommit, error)) *MockAppConnConsensus_CommitSync_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockAppConnConsensus) DeliverTxAsync(_a0 types.RequestDeliverTx) *abcicli.ReqRes {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for DeliverTxAsync")
	}

	var r0 *abcicli.ReqRes
	if rf, ok := ret.Get(0).(func(types.RequestDeliverTx) *abcicli.ReqRes); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*abcicli.ReqRes)
		}
	}

	return r0
}


type MockAppConnConsensus_DeliverTxAsync_Call struct {
	*mock.Call
}



func (_e *MockAppConnConsensus_Expecter) DeliverTxAsync(_a0 interface{}) *MockAppConnConsensus_DeliverTxAsync_Call {
	return &MockAppConnConsensus_DeliverTxAsync_Call{Call: _e.mock.On("DeliverTxAsync", _a0)}
}

func (_c *MockAppConnConsensus_DeliverTxAsync_Call) Run(run func(_a0 types.RequestDeliverTx)) *MockAppConnConsensus_DeliverTxAsync_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.RequestDeliverTx))
	})
	return _c
}

func (_c *MockAppConnConsensus_DeliverTxAsync_Call) Return(_a0 *abcicli.ReqRes) *MockAppConnConsensus_DeliverTxAsync_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockAppConnConsensus_DeliverTxAsync_Call) RunAndReturn(run func(types.RequestDeliverTx) *abcicli.ReqRes) *MockAppConnConsensus_DeliverTxAsync_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockAppConnConsensus) EndBlockSync(_a0 types.RequestEndBlock) (*types.ResponseEndBlock, error) {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for EndBlockSync")
	}

	var r0 *types.ResponseEndBlock
	var r1 error
	if rf, ok := ret.Get(0).(func(types.RequestEndBlock) (*types.ResponseEndBlock, error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func(types.RequestEndBlock) *types.ResponseEndBlock); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.ResponseEndBlock)
		}
	}

	if rf, ok := ret.Get(1).(func(types.RequestEndBlock) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockAppConnConsensus_EndBlockSync_Call struct {
	*mock.Call
}



func (_e *MockAppConnConsensus_Expecter) EndBlockSync(_a0 interface{}) *MockAppConnConsensus_EndBlockSync_Call {
	return &MockAppConnConsensus_EndBlockSync_Call{Call: _e.mock.On("EndBlockSync", _a0)}
}

func (_c *MockAppConnConsensus_EndBlockSync_Call) Run(run func(_a0 types.RequestEndBlock)) *MockAppConnConsensus_EndBlockSync_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.RequestEndBlock))
	})
	return _c
}

func (_c *MockAppConnConsensus_EndBlockSync_Call) Return(_a0 *types.ResponseEndBlock, _a1 error) *MockAppConnConsensus_EndBlockSync_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockAppConnConsensus_EndBlockSync_Call) RunAndReturn(run func(types.RequestEndBlock) (*types.ResponseEndBlock, error)) *MockAppConnConsensus_EndBlockSync_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockAppConnConsensus) Error() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Error")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}


type MockAppConnConsensus_Error_Call struct {
	*mock.Call
}


func (_e *MockAppConnConsensus_Expecter) Error() *MockAppConnConsensus_Error_Call {
	return &MockAppConnConsensus_Error_Call{Call: _e.mock.On("Error")}
}

func (_c *MockAppConnConsensus_Error_Call) Run(run func()) *MockAppConnConsensus_Error_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockAppConnConsensus_Error_Call) Return(_a0 error) *MockAppConnConsensus_Error_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockAppConnConsensus_Error_Call) RunAndReturn(run func() error) *MockAppConnConsensus_Error_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockAppConnConsensus) InitChainSync(_a0 types.RequestInitChain) (*types.ResponseInitChain, error) {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for InitChainSync")
	}

	var r0 *types.ResponseInitChain
	var r1 error
	if rf, ok := ret.Get(0).(func(types.RequestInitChain) (*types.ResponseInitChain, error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func(types.RequestInitChain) *types.ResponseInitChain); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.ResponseInitChain)
		}
	}

	if rf, ok := ret.Get(1).(func(types.RequestInitChain) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockAppConnConsensus_InitChainSync_Call struct {
	*mock.Call
}



func (_e *MockAppConnConsensus_Expecter) InitChainSync(_a0 interface{}) *MockAppConnConsensus_InitChainSync_Call {
	return &MockAppConnConsensus_InitChainSync_Call{Call: _e.mock.On("InitChainSync", _a0)}
}

func (_c *MockAppConnConsensus_InitChainSync_Call) Run(run func(_a0 types.RequestInitChain)) *MockAppConnConsensus_InitChainSync_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.RequestInitChain))
	})
	return _c
}

func (_c *MockAppConnConsensus_InitChainSync_Call) Return(_a0 *types.ResponseInitChain, _a1 error) *MockAppConnConsensus_InitChainSync_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockAppConnConsensus_InitChainSync_Call) RunAndReturn(run func(types.RequestInitChain) (*types.ResponseInitChain, error)) *MockAppConnConsensus_InitChainSync_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockAppConnConsensus) SetResponseCallback(_a0 abcicli.Callback) {
	_m.Called(_a0)
}


type MockAppConnConsensus_SetResponseCallback_Call struct {
	*mock.Call
}



func (_e *MockAppConnConsensus_Expecter) SetResponseCallback(_a0 interface{}) *MockAppConnConsensus_SetResponseCallback_Call {
	return &MockAppConnConsensus_SetResponseCallback_Call{Call: _e.mock.On("SetResponseCallback", _a0)}
}

func (_c *MockAppConnConsensus_SetResponseCallback_Call) Run(run func(_a0 abcicli.Callback)) *MockAppConnConsensus_SetResponseCallback_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(abcicli.Callback))
	})
	return _c
}

func (_c *MockAppConnConsensus_SetResponseCallback_Call) Return() *MockAppConnConsensus_SetResponseCallback_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockAppConnConsensus_SetResponseCallback_Call) RunAndReturn(run func(abcicli.Callback)) *MockAppConnConsensus_SetResponseCallback_Call {
	_c.Call.Return(run)
	return _c
}



func NewMockAppConnConsensus(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockAppConnConsensus {
	mock := &MockAppConnConsensus{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
