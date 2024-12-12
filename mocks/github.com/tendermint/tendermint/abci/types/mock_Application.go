

package types

import (
	mock "github.com/stretchr/testify/mock"
	types "github.com/tendermint/tendermint/abci/types"
)


type MockApplication struct {
	mock.Mock
}

type MockApplication_Expecter struct {
	mock *mock.Mock
}

func (_m *MockApplication) EXPECT() *MockApplication_Expecter {
	return &MockApplication_Expecter{mock: &_m.Mock}
}


func (_m *MockApplication) ApplySnapshotChunk(_a0 types.RequestApplySnapshotChunk) types.ResponseApplySnapshotChunk {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for ApplySnapshotChunk")
	}

	var r0 types.ResponseApplySnapshotChunk
	if rf, ok := ret.Get(0).(func(types.RequestApplySnapshotChunk) types.ResponseApplySnapshotChunk); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(types.ResponseApplySnapshotChunk)
	}

	return r0
}


type MockApplication_ApplySnapshotChunk_Call struct {
	*mock.Call
}



func (_e *MockApplication_Expecter) ApplySnapshotChunk(_a0 interface{}) *MockApplication_ApplySnapshotChunk_Call {
	return &MockApplication_ApplySnapshotChunk_Call{Call: _e.mock.On("ApplySnapshotChunk", _a0)}
}

func (_c *MockApplication_ApplySnapshotChunk_Call) Run(run func(_a0 types.RequestApplySnapshotChunk)) *MockApplication_ApplySnapshotChunk_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.RequestApplySnapshotChunk))
	})
	return _c
}

func (_c *MockApplication_ApplySnapshotChunk_Call) Return(_a0 types.ResponseApplySnapshotChunk) *MockApplication_ApplySnapshotChunk_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockApplication_ApplySnapshotChunk_Call) RunAndReturn(run func(types.RequestApplySnapshotChunk) types.ResponseApplySnapshotChunk) *MockApplication_ApplySnapshotChunk_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockApplication) BeginBlock(_a0 types.RequestBeginBlock) types.ResponseBeginBlock {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for BeginBlock")
	}

	var r0 types.ResponseBeginBlock
	if rf, ok := ret.Get(0).(func(types.RequestBeginBlock) types.ResponseBeginBlock); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(types.ResponseBeginBlock)
	}

	return r0
}


type MockApplication_BeginBlock_Call struct {
	*mock.Call
}



func (_e *MockApplication_Expecter) BeginBlock(_a0 interface{}) *MockApplication_BeginBlock_Call {
	return &MockApplication_BeginBlock_Call{Call: _e.mock.On("BeginBlock", _a0)}
}

func (_c *MockApplication_BeginBlock_Call) Run(run func(_a0 types.RequestBeginBlock)) *MockApplication_BeginBlock_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.RequestBeginBlock))
	})
	return _c
}

func (_c *MockApplication_BeginBlock_Call) Return(_a0 types.ResponseBeginBlock) *MockApplication_BeginBlock_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockApplication_BeginBlock_Call) RunAndReturn(run func(types.RequestBeginBlock) types.ResponseBeginBlock) *MockApplication_BeginBlock_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockApplication) CheckTx(_a0 types.RequestCheckTx) types.ResponseCheckTx {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for CheckTx")
	}

	var r0 types.ResponseCheckTx
	if rf, ok := ret.Get(0).(func(types.RequestCheckTx) types.ResponseCheckTx); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(types.ResponseCheckTx)
	}

	return r0
}


type MockApplication_CheckTx_Call struct {
	*mock.Call
}



func (_e *MockApplication_Expecter) CheckTx(_a0 interface{}) *MockApplication_CheckTx_Call {
	return &MockApplication_CheckTx_Call{Call: _e.mock.On("CheckTx", _a0)}
}

func (_c *MockApplication_CheckTx_Call) Run(run func(_a0 types.RequestCheckTx)) *MockApplication_CheckTx_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.RequestCheckTx))
	})
	return _c
}

func (_c *MockApplication_CheckTx_Call) Return(_a0 types.ResponseCheckTx) *MockApplication_CheckTx_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockApplication_CheckTx_Call) RunAndReturn(run func(types.RequestCheckTx) types.ResponseCheckTx) *MockApplication_CheckTx_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockApplication) Commit() types.ResponseCommit {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Commit")
	}

	var r0 types.ResponseCommit
	if rf, ok := ret.Get(0).(func() types.ResponseCommit); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(types.ResponseCommit)
	}

	return r0
}


type MockApplication_Commit_Call struct {
	*mock.Call
}


func (_e *MockApplication_Expecter) Commit() *MockApplication_Commit_Call {
	return &MockApplication_Commit_Call{Call: _e.mock.On("Commit")}
}

func (_c *MockApplication_Commit_Call) Run(run func()) *MockApplication_Commit_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockApplication_Commit_Call) Return(_a0 types.ResponseCommit) *MockApplication_Commit_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockApplication_Commit_Call) RunAndReturn(run func() types.ResponseCommit) *MockApplication_Commit_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockApplication) DeliverTx(_a0 types.RequestDeliverTx) types.ResponseDeliverTx {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for DeliverTx")
	}

	var r0 types.ResponseDeliverTx
	if rf, ok := ret.Get(0).(func(types.RequestDeliverTx) types.ResponseDeliverTx); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(types.ResponseDeliverTx)
	}

	return r0
}


type MockApplication_DeliverTx_Call struct {
	*mock.Call
}



func (_e *MockApplication_Expecter) DeliverTx(_a0 interface{}) *MockApplication_DeliverTx_Call {
	return &MockApplication_DeliverTx_Call{Call: _e.mock.On("DeliverTx", _a0)}
}

func (_c *MockApplication_DeliverTx_Call) Run(run func(_a0 types.RequestDeliverTx)) *MockApplication_DeliverTx_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.RequestDeliverTx))
	})
	return _c
}

func (_c *MockApplication_DeliverTx_Call) Return(_a0 types.ResponseDeliverTx) *MockApplication_DeliverTx_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockApplication_DeliverTx_Call) RunAndReturn(run func(types.RequestDeliverTx) types.ResponseDeliverTx) *MockApplication_DeliverTx_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockApplication) EndBlock(_a0 types.RequestEndBlock) types.ResponseEndBlock {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for EndBlock")
	}

	var r0 types.ResponseEndBlock
	if rf, ok := ret.Get(0).(func(types.RequestEndBlock) types.ResponseEndBlock); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(types.ResponseEndBlock)
	}

	return r0
}


type MockApplication_EndBlock_Call struct {
	*mock.Call
}



func (_e *MockApplication_Expecter) EndBlock(_a0 interface{}) *MockApplication_EndBlock_Call {
	return &MockApplication_EndBlock_Call{Call: _e.mock.On("EndBlock", _a0)}
}

func (_c *MockApplication_EndBlock_Call) Run(run func(_a0 types.RequestEndBlock)) *MockApplication_EndBlock_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.RequestEndBlock))
	})
	return _c
}

func (_c *MockApplication_EndBlock_Call) Return(_a0 types.ResponseEndBlock) *MockApplication_EndBlock_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockApplication_EndBlock_Call) RunAndReturn(run func(types.RequestEndBlock) types.ResponseEndBlock) *MockApplication_EndBlock_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockApplication) Info(_a0 types.RequestInfo) types.ResponseInfo {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for Info")
	}

	var r0 types.ResponseInfo
	if rf, ok := ret.Get(0).(func(types.RequestInfo) types.ResponseInfo); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(types.ResponseInfo)
	}

	return r0
}


type MockApplication_Info_Call struct {
	*mock.Call
}



func (_e *MockApplication_Expecter) Info(_a0 interface{}) *MockApplication_Info_Call {
	return &MockApplication_Info_Call{Call: _e.mock.On("Info", _a0)}
}

func (_c *MockApplication_Info_Call) Run(run func(_a0 types.RequestInfo)) *MockApplication_Info_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.RequestInfo))
	})
	return _c
}

func (_c *MockApplication_Info_Call) Return(_a0 types.ResponseInfo) *MockApplication_Info_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockApplication_Info_Call) RunAndReturn(run func(types.RequestInfo) types.ResponseInfo) *MockApplication_Info_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockApplication) InitChain(_a0 types.RequestInitChain) types.ResponseInitChain {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for InitChain")
	}

	var r0 types.ResponseInitChain
	if rf, ok := ret.Get(0).(func(types.RequestInitChain) types.ResponseInitChain); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(types.ResponseInitChain)
	}

	return r0
}


type MockApplication_InitChain_Call struct {
	*mock.Call
}



func (_e *MockApplication_Expecter) InitChain(_a0 interface{}) *MockApplication_InitChain_Call {
	return &MockApplication_InitChain_Call{Call: _e.mock.On("InitChain", _a0)}
}

func (_c *MockApplication_InitChain_Call) Run(run func(_a0 types.RequestInitChain)) *MockApplication_InitChain_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.RequestInitChain))
	})
	return _c
}

func (_c *MockApplication_InitChain_Call) Return(_a0 types.ResponseInitChain) *MockApplication_InitChain_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockApplication_InitChain_Call) RunAndReturn(run func(types.RequestInitChain) types.ResponseInitChain) *MockApplication_InitChain_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockApplication) ListSnapshots(_a0 types.RequestListSnapshots) types.ResponseListSnapshots {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for ListSnapshots")
	}

	var r0 types.ResponseListSnapshots
	if rf, ok := ret.Get(0).(func(types.RequestListSnapshots) types.ResponseListSnapshots); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(types.ResponseListSnapshots)
	}

	return r0
}


type MockApplication_ListSnapshots_Call struct {
	*mock.Call
}



func (_e *MockApplication_Expecter) ListSnapshots(_a0 interface{}) *MockApplication_ListSnapshots_Call {
	return &MockApplication_ListSnapshots_Call{Call: _e.mock.On("ListSnapshots", _a0)}
}

func (_c *MockApplication_ListSnapshots_Call) Run(run func(_a0 types.RequestListSnapshots)) *MockApplication_ListSnapshots_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.RequestListSnapshots))
	})
	return _c
}

func (_c *MockApplication_ListSnapshots_Call) Return(_a0 types.ResponseListSnapshots) *MockApplication_ListSnapshots_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockApplication_ListSnapshots_Call) RunAndReturn(run func(types.RequestListSnapshots) types.ResponseListSnapshots) *MockApplication_ListSnapshots_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockApplication) LoadSnapshotChunk(_a0 types.RequestLoadSnapshotChunk) types.ResponseLoadSnapshotChunk {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for LoadSnapshotChunk")
	}

	var r0 types.ResponseLoadSnapshotChunk
	if rf, ok := ret.Get(0).(func(types.RequestLoadSnapshotChunk) types.ResponseLoadSnapshotChunk); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(types.ResponseLoadSnapshotChunk)
	}

	return r0
}


type MockApplication_LoadSnapshotChunk_Call struct {
	*mock.Call
}



func (_e *MockApplication_Expecter) LoadSnapshotChunk(_a0 interface{}) *MockApplication_LoadSnapshotChunk_Call {
	return &MockApplication_LoadSnapshotChunk_Call{Call: _e.mock.On("LoadSnapshotChunk", _a0)}
}

func (_c *MockApplication_LoadSnapshotChunk_Call) Run(run func(_a0 types.RequestLoadSnapshotChunk)) *MockApplication_LoadSnapshotChunk_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.RequestLoadSnapshotChunk))
	})
	return _c
}

func (_c *MockApplication_LoadSnapshotChunk_Call) Return(_a0 types.ResponseLoadSnapshotChunk) *MockApplication_LoadSnapshotChunk_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockApplication_LoadSnapshotChunk_Call) RunAndReturn(run func(types.RequestLoadSnapshotChunk) types.ResponseLoadSnapshotChunk) *MockApplication_LoadSnapshotChunk_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockApplication) OfferSnapshot(_a0 types.RequestOfferSnapshot) types.ResponseOfferSnapshot {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for OfferSnapshot")
	}

	var r0 types.ResponseOfferSnapshot
	if rf, ok := ret.Get(0).(func(types.RequestOfferSnapshot) types.ResponseOfferSnapshot); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(types.ResponseOfferSnapshot)
	}

	return r0
}


type MockApplication_OfferSnapshot_Call struct {
	*mock.Call
}



func (_e *MockApplication_Expecter) OfferSnapshot(_a0 interface{}) *MockApplication_OfferSnapshot_Call {
	return &MockApplication_OfferSnapshot_Call{Call: _e.mock.On("OfferSnapshot", _a0)}
}

func (_c *MockApplication_OfferSnapshot_Call) Run(run func(_a0 types.RequestOfferSnapshot)) *MockApplication_OfferSnapshot_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.RequestOfferSnapshot))
	})
	return _c
}

func (_c *MockApplication_OfferSnapshot_Call) Return(_a0 types.ResponseOfferSnapshot) *MockApplication_OfferSnapshot_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockApplication_OfferSnapshot_Call) RunAndReturn(run func(types.RequestOfferSnapshot) types.ResponseOfferSnapshot) *MockApplication_OfferSnapshot_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockApplication) Query(_a0 types.RequestQuery) types.ResponseQuery {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for Query")
	}

	var r0 types.ResponseQuery
	if rf, ok := ret.Get(0).(func(types.RequestQuery) types.ResponseQuery); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(types.ResponseQuery)
	}

	return r0
}


type MockApplication_Query_Call struct {
	*mock.Call
}



func (_e *MockApplication_Expecter) Query(_a0 interface{}) *MockApplication_Query_Call {
	return &MockApplication_Query_Call{Call: _e.mock.On("Query", _a0)}
}

func (_c *MockApplication_Query_Call) Run(run func(_a0 types.RequestQuery)) *MockApplication_Query_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.RequestQuery))
	})
	return _c
}

func (_c *MockApplication_Query_Call) Return(_a0 types.ResponseQuery) *MockApplication_Query_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockApplication_Query_Call) RunAndReturn(run func(types.RequestQuery) types.ResponseQuery) *MockApplication_Query_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockApplication) SetOption(_a0 types.RequestSetOption) types.ResponseSetOption {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for SetOption")
	}

	var r0 types.ResponseSetOption
	if rf, ok := ret.Get(0).(func(types.RequestSetOption) types.ResponseSetOption); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(types.ResponseSetOption)
	}

	return r0
}


type MockApplication_SetOption_Call struct {
	*mock.Call
}



func (_e *MockApplication_Expecter) SetOption(_a0 interface{}) *MockApplication_SetOption_Call {
	return &MockApplication_SetOption_Call{Call: _e.mock.On("SetOption", _a0)}
}

func (_c *MockApplication_SetOption_Call) Run(run func(_a0 types.RequestSetOption)) *MockApplication_SetOption_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.RequestSetOption))
	})
	return _c
}

func (_c *MockApplication_SetOption_Call) Return(_a0 types.ResponseSetOption) *MockApplication_SetOption_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockApplication_SetOption_Call) RunAndReturn(run func(types.RequestSetOption) types.ResponseSetOption) *MockApplication_SetOption_Call {
	_c.Call.Return(run)
	return _c
}



func NewMockApplication(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockApplication {
	mock := &MockApplication{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
