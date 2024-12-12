package proxy

import (
	mock "github.com/stretchr/testify/mock"
	log "github.com/tendermint/tendermint/libs/log"

	proxy "github.com/tendermint/tendermint/proxy"
)

type MockAppConns struct {
	mock.Mock
}

type MockAppConns_Expecter struct {
	mock *mock.Mock
}

func (_m *MockAppConns) EXPECT() *MockAppConns_Expecter {
	return &MockAppConns_Expecter{mock: &_m.Mock}
}

func (_m *MockAppConns) Consensus() proxy.AppConnConsensus {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Consensus")
	}

	var r0 proxy.AppConnConsensus
	if rf, ok := ret.Get(0).(func() proxy.AppConnConsensus); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(proxy.AppConnConsensus)
		}
	}

	return r0
}

type MockAppConns_Consensus_Call struct {
	*mock.Call
}

func (_e *MockAppConns_Expecter) Consensus() *MockAppConns_Consensus_Call {
	return &MockAppConns_Consensus_Call{Call: _e.mock.On("Consensus")}
}

func (_c *MockAppConns_Consensus_Call) Run(run func()) *MockAppConns_Consensus_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockAppConns_Consensus_Call) Return(_a0 proxy.AppConnConsensus) *MockAppConns_Consensus_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockAppConns_Consensus_Call) RunAndReturn(run func() proxy.AppConnConsensus) *MockAppConns_Consensus_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockAppConns) IsRunning() bool {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for IsRunning")
	}

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

type MockAppConns_IsRunning_Call struct {
	*mock.Call
}

func (_e *MockAppConns_Expecter) IsRunning() *MockAppConns_IsRunning_Call {
	return &MockAppConns_IsRunning_Call{Call: _e.mock.On("IsRunning")}
}

func (_c *MockAppConns_IsRunning_Call) Run(run func()) *MockAppConns_IsRunning_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockAppConns_IsRunning_Call) Return(_a0 bool) *MockAppConns_IsRunning_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockAppConns_IsRunning_Call) RunAndReturn(run func() bool) *MockAppConns_IsRunning_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockAppConns) Mempool() proxy.AppConnMempool {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Mempool")
	}

	var r0 proxy.AppConnMempool
	if rf, ok := ret.Get(0).(func() proxy.AppConnMempool); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(proxy.AppConnMempool)
		}
	}

	return r0
}

type MockAppConns_Mempool_Call struct {
	*mock.Call
}

func (_e *MockAppConns_Expecter) Mempool() *MockAppConns_Mempool_Call {
	return &MockAppConns_Mempool_Call{Call: _e.mock.On("Mempool")}
}

func (_c *MockAppConns_Mempool_Call) Run(run func()) *MockAppConns_Mempool_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockAppConns_Mempool_Call) Return(_a0 proxy.AppConnMempool) *MockAppConns_Mempool_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockAppConns_Mempool_Call) RunAndReturn(run func() proxy.AppConnMempool) *MockAppConns_Mempool_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockAppConns) OnReset() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for OnReset")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type MockAppConns_OnReset_Call struct {
	*mock.Call
}

func (_e *MockAppConns_Expecter) OnReset() *MockAppConns_OnReset_Call {
	return &MockAppConns_OnReset_Call{Call: _e.mock.On("OnReset")}
}

func (_c *MockAppConns_OnReset_Call) Run(run func()) *MockAppConns_OnReset_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockAppConns_OnReset_Call) Return(_a0 error) *MockAppConns_OnReset_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockAppConns_OnReset_Call) RunAndReturn(run func() error) *MockAppConns_OnReset_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockAppConns) OnStart() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for OnStart")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type MockAppConns_OnStart_Call struct {
	*mock.Call
}

func (_e *MockAppConns_Expecter) OnStart() *MockAppConns_OnStart_Call {
	return &MockAppConns_OnStart_Call{Call: _e.mock.On("OnStart")}
}

func (_c *MockAppConns_OnStart_Call) Run(run func()) *MockAppConns_OnStart_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockAppConns_OnStart_Call) Return(_a0 error) *MockAppConns_OnStart_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockAppConns_OnStart_Call) RunAndReturn(run func() error) *MockAppConns_OnStart_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockAppConns) OnStop() {
	_m.Called()
}

type MockAppConns_OnStop_Call struct {
	*mock.Call
}

func (_e *MockAppConns_Expecter) OnStop() *MockAppConns_OnStop_Call {
	return &MockAppConns_OnStop_Call{Call: _e.mock.On("OnStop")}
}

func (_c *MockAppConns_OnStop_Call) Run(run func()) *MockAppConns_OnStop_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockAppConns_OnStop_Call) Return() *MockAppConns_OnStop_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockAppConns_OnStop_Call) RunAndReturn(run func()) *MockAppConns_OnStop_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockAppConns) Query() proxy.AppConnQuery {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Query")
	}

	var r0 proxy.AppConnQuery
	if rf, ok := ret.Get(0).(func() proxy.AppConnQuery); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(proxy.AppConnQuery)
		}
	}

	return r0
}

type MockAppConns_Query_Call struct {
	*mock.Call
}

func (_e *MockAppConns_Expecter) Query() *MockAppConns_Query_Call {
	return &MockAppConns_Query_Call{Call: _e.mock.On("Query")}
}

func (_c *MockAppConns_Query_Call) Run(run func()) *MockAppConns_Query_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockAppConns_Query_Call) Return(_a0 proxy.AppConnQuery) *MockAppConns_Query_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockAppConns_Query_Call) RunAndReturn(run func() proxy.AppConnQuery) *MockAppConns_Query_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockAppConns) Quit() <-chan struct{} {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Quit")
	}

	var r0 <-chan struct{}
	if rf, ok := ret.Get(0).(func() <-chan struct{}); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan struct{})
		}
	}

	return r0
}

type MockAppConns_Quit_Call struct {
	*mock.Call
}

func (_e *MockAppConns_Expecter) Quit() *MockAppConns_Quit_Call {
	return &MockAppConns_Quit_Call{Call: _e.mock.On("Quit")}
}

func (_c *MockAppConns_Quit_Call) Run(run func()) *MockAppConns_Quit_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockAppConns_Quit_Call) Return(_a0 <-chan struct{}) *MockAppConns_Quit_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockAppConns_Quit_Call) RunAndReturn(run func() <-chan struct{}) *MockAppConns_Quit_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockAppConns) Reset() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Reset")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type MockAppConns_Reset_Call struct {
	*mock.Call
}

func (_e *MockAppConns_Expecter) Reset() *MockAppConns_Reset_Call {
	return &MockAppConns_Reset_Call{Call: _e.mock.On("Reset")}
}

func (_c *MockAppConns_Reset_Call) Run(run func()) *MockAppConns_Reset_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockAppConns_Reset_Call) Return(_a0 error) *MockAppConns_Reset_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockAppConns_Reset_Call) RunAndReturn(run func() error) *MockAppConns_Reset_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockAppConns) SetLogger(_a0 log.Logger) {
	_m.Called(_a0)
}

type MockAppConns_SetLogger_Call struct {
	*mock.Call
}

func (_e *MockAppConns_Expecter) SetLogger(_a0 interface{}) *MockAppConns_SetLogger_Call {
	return &MockAppConns_SetLogger_Call{Call: _e.mock.On("SetLogger", _a0)}
}

func (_c *MockAppConns_SetLogger_Call) Run(run func(_a0 log.Logger)) *MockAppConns_SetLogger_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(log.Logger))
	})
	return _c
}

func (_c *MockAppConns_SetLogger_Call) Return() *MockAppConns_SetLogger_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockAppConns_SetLogger_Call) RunAndReturn(run func(log.Logger)) *MockAppConns_SetLogger_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockAppConns) Snapshot() proxy.AppConnSnapshot {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Snapshot")
	}

	var r0 proxy.AppConnSnapshot
	if rf, ok := ret.Get(0).(func() proxy.AppConnSnapshot); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(proxy.AppConnSnapshot)
		}
	}

	return r0
}

type MockAppConns_Snapshot_Call struct {
	*mock.Call
}

func (_e *MockAppConns_Expecter) Snapshot() *MockAppConns_Snapshot_Call {
	return &MockAppConns_Snapshot_Call{Call: _e.mock.On("Snapshot")}
}

func (_c *MockAppConns_Snapshot_Call) Run(run func()) *MockAppConns_Snapshot_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockAppConns_Snapshot_Call) Return(_a0 proxy.AppConnSnapshot) *MockAppConns_Snapshot_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockAppConns_Snapshot_Call) RunAndReturn(run func() proxy.AppConnSnapshot) *MockAppConns_Snapshot_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockAppConns) Start() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Start")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type MockAppConns_Start_Call struct {
	*mock.Call
}

func (_e *MockAppConns_Expecter) Start() *MockAppConns_Start_Call {
	return &MockAppConns_Start_Call{Call: _e.mock.On("Start")}
}

func (_c *MockAppConns_Start_Call) Run(run func()) *MockAppConns_Start_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockAppConns_Start_Call) Return(_a0 error) *MockAppConns_Start_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockAppConns_Start_Call) RunAndReturn(run func() error) *MockAppConns_Start_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockAppConns) Stop() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Stop")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type MockAppConns_Stop_Call struct {
	*mock.Call
}

func (_e *MockAppConns_Expecter) Stop() *MockAppConns_Stop_Call {
	return &MockAppConns_Stop_Call{Call: _e.mock.On("Stop")}
}

func (_c *MockAppConns_Stop_Call) Run(run func()) *MockAppConns_Stop_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockAppConns_Stop_Call) Return(_a0 error) *MockAppConns_Stop_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockAppConns_Stop_Call) RunAndReturn(run func() error) *MockAppConns_Stop_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockAppConns) String() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for String")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

type MockAppConns_String_Call struct {
	*mock.Call
}

func (_e *MockAppConns_Expecter) String() *MockAppConns_String_Call {
	return &MockAppConns_String_Call{Call: _e.mock.On("String")}
}

func (_c *MockAppConns_String_Call) Run(run func()) *MockAppConns_String_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockAppConns_String_Call) Return(_a0 string) *MockAppConns_String_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockAppConns_String_Call) RunAndReturn(run func() string) *MockAppConns_String_Call {
	_c.Call.Return(run)
	return _c
}

func NewMockAppConns(t interface {
	mock.TestingT
	Cleanup(func())
},
) *MockAppConns {
	mock := &MockAppConns{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
