package da

import (
	da "github.com/dymensionxyz/dymint/da"
	mock "github.com/stretchr/testify/mock"

	pubsub "github.com/tendermint/tendermint/libs/pubsub"

	store "github.com/dymensionxyz/dymint/store"

	types "github.com/dymensionxyz/dymint/types"
)

type MockDataAvailabilityLayerClient struct {
	mock.Mock
}

type MockDataAvailabilityLayerClient_Expecter struct {
	mock *mock.Mock
}

func (_m *MockDataAvailabilityLayerClient) EXPECT() *MockDataAvailabilityLayerClient_Expecter {
	return &MockDataAvailabilityLayerClient_Expecter{mock: &_m.Mock}
}

func (_m *MockDataAvailabilityLayerClient) CheckBatchAvailability(daMetaData *da.DASubmitMetaData) da.ResultCheckBatch {
	ret := _m.Called(daMetaData)

	if len(ret) == 0 {
		panic("no return value specified for CheckBatchAvailability")
	}

	var r0 da.ResultCheckBatch
	if rf, ok := ret.Get(0).(func(*da.DASubmitMetaData) da.ResultCheckBatch); ok {
		r0 = rf(daMetaData)
	} else {
		r0 = ret.Get(0).(da.ResultCheckBatch)
	}

	return r0
}

type MockDataAvailabilityLayerClient_CheckBatchAvailability_Call struct {
	*mock.Call
}

func (_e *MockDataAvailabilityLayerClient_Expecter) CheckBatchAvailability(daMetaData interface{}) *MockDataAvailabilityLayerClient_CheckBatchAvailability_Call {
	return &MockDataAvailabilityLayerClient_CheckBatchAvailability_Call{Call: _e.mock.On("CheckBatchAvailability", daMetaData)}
}

func (_c *MockDataAvailabilityLayerClient_CheckBatchAvailability_Call) Run(run func(daMetaData *da.DASubmitMetaData)) *MockDataAvailabilityLayerClient_CheckBatchAvailability_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*da.DASubmitMetaData))
	})
	return _c
}

func (_c *MockDataAvailabilityLayerClient_CheckBatchAvailability_Call) Return(_a0 da.ResultCheckBatch) *MockDataAvailabilityLayerClient_CheckBatchAvailability_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockDataAvailabilityLayerClient_CheckBatchAvailability_Call) RunAndReturn(run func(*da.DASubmitMetaData) da.ResultCheckBatch) *MockDataAvailabilityLayerClient_CheckBatchAvailability_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockDataAvailabilityLayerClient) GetClientType() da.Client {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetClientType")
	}

	var r0 da.Client
	if rf, ok := ret.Get(0).(func() da.Client); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(da.Client)
	}

	return r0
}

type MockDataAvailabilityLayerClient_GetClientType_Call struct {
	*mock.Call
}

func (_e *MockDataAvailabilityLayerClient_Expecter) GetClientType() *MockDataAvailabilityLayerClient_GetClientType_Call {
	return &MockDataAvailabilityLayerClient_GetClientType_Call{Call: _e.mock.On("GetClientType")}
}

func (_c *MockDataAvailabilityLayerClient_GetClientType_Call) Run(run func()) *MockDataAvailabilityLayerClient_GetClientType_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockDataAvailabilityLayerClient_GetClientType_Call) Return(_a0 da.Client) *MockDataAvailabilityLayerClient_GetClientType_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockDataAvailabilityLayerClient_GetClientType_Call) RunAndReturn(run func() da.Client) *MockDataAvailabilityLayerClient_GetClientType_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockDataAvailabilityLayerClient) GetMaxBlobSizeBytes() uint32 {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetMaxBlobSizeBytes")
	}

	var r0 uint32
	if rf, ok := ret.Get(0).(func() uint32); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint32)
	}

	return r0
}

type MockDataAvailabilityLayerClient_GetMaxBlobSizeBytes_Call struct {
	*mock.Call
}

func (_e *MockDataAvailabilityLayerClient_Expecter) GetMaxBlobSizeBytes() *MockDataAvailabilityLayerClient_GetMaxBlobSizeBytes_Call {
	return &MockDataAvailabilityLayerClient_GetMaxBlobSizeBytes_Call{Call: _e.mock.On("GetMaxBlobSizeBytes")}
}

func (_c *MockDataAvailabilityLayerClient_GetMaxBlobSizeBytes_Call) Run(run func()) *MockDataAvailabilityLayerClient_GetMaxBlobSizeBytes_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockDataAvailabilityLayerClient_GetMaxBlobSizeBytes_Call) Return(_a0 uint32) *MockDataAvailabilityLayerClient_GetMaxBlobSizeBytes_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockDataAvailabilityLayerClient_GetMaxBlobSizeBytes_Call) RunAndReturn(run func() uint32) *MockDataAvailabilityLayerClient_GetMaxBlobSizeBytes_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockDataAvailabilityLayerClient) GetSignerBalance() (da.Balance, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetSignerBalance")
	}

	var r0 da.Balance
	var r1 error
	if rf, ok := ret.Get(0).(func() (da.Balance, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() da.Balance); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(da.Balance)
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
	}

	return r0, r1
}

type MockDataAvailabilityLayerClient_GetSignerBalance_Call struct {
	*mock.Call
}

func (_e *MockDataAvailabilityLayerClient_Expecter) GetSignerBalance() *MockDataAvailabilityLayerClient_GetSignerBalance_Call {
	return &MockDataAvailabilityLayerClient_GetSignerBalance_Call{Call: _e.mock.On("GetSignerBalance")}
}

func (_c *MockDataAvailabilityLayerClient_GetSignerBalance_Call) Run(run func()) *MockDataAvailabilityLayerClient_GetSignerBalance_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockDataAvailabilityLayerClient_GetSignerBalance_Call) Return(_a0 da.Balance, _a1 error) *MockDataAvailabilityLayerClient_GetSignerBalance_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockDataAvailabilityLayerClient_GetSignerBalance_Call) RunAndReturn(run func() (da.Balance, error)) *MockDataAvailabilityLayerClient_GetSignerBalance_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockDataAvailabilityLayerClient) Init(config []byte, pubsubServer *pubsub.Server, kvStore store.KV, logger types.Logger, options ...da.Option) error {
	_va := make([]interface{}, len(options))
	for _i := range options {
		_va[_i] = options[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, config, pubsubServer, kvStore, logger)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Init")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func([]byte, *pubsub.Server, store.KV, types.Logger, ...da.Option) error); ok {
		r0 = rf(config, pubsubServer, kvStore, logger, options...)
	} else {
	}

	return r0
}

type MockDataAvailabilityLayerClient_Init_Call struct {
	*mock.Call
}

func (_e *MockDataAvailabilityLayerClient_Expecter) Init(config interface{}, pubsubServer interface{}, kvStore interface{}, logger interface{}, options ...interface{}) *MockDataAvailabilityLayerClient_Init_Call {
	return &MockDataAvailabilityLayerClient_Init_Call{Call: _e.mock.On("Init",
		append([]interface{}{config, pubsubServer, kvStore, logger}, options...)...)}
}

func (_c *MockDataAvailabilityLayerClient_Init_Call) Run(run func(config []byte, pubsubServer *pubsub.Server, kvStore store.KV, logger types.Logger, options ...da.Option)) *MockDataAvailabilityLayerClient_Init_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]da.Option, len(args)-4)
		for i, a := range args[4:] {
			if a != nil {
				variadicArgs[i] = a.(da.Option)
			}
		}
		run(args[0].([]byte), args[1].(*pubsub.Server), args[2].(store.KV), args[3].(types.Logger), variadicArgs...)
	})
	return _c
}

func (_c *MockDataAvailabilityLayerClient_Init_Call) Return(_a0 error) *MockDataAvailabilityLayerClient_Init_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockDataAvailabilityLayerClient_Init_Call) RunAndReturn(run func([]byte, *pubsub.Server, store.KV, types.Logger, ...da.Option) error) *MockDataAvailabilityLayerClient_Init_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockDataAvailabilityLayerClient) Start() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Start")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
	}

	return r0
}

type MockDataAvailabilityLayerClient_Start_Call struct {
	*mock.Call
}

func (_e *MockDataAvailabilityLayerClient_Expecter) Start() *MockDataAvailabilityLayerClient_Start_Call {
	return &MockDataAvailabilityLayerClient_Start_Call{Call: _e.mock.On("Start")}
}

func (_c *MockDataAvailabilityLayerClient_Start_Call) Run(run func()) *MockDataAvailabilityLayerClient_Start_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockDataAvailabilityLayerClient_Start_Call) Return(_a0 error) *MockDataAvailabilityLayerClient_Start_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockDataAvailabilityLayerClient_Start_Call) RunAndReturn(run func() error) *MockDataAvailabilityLayerClient_Start_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockDataAvailabilityLayerClient) Stop() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Stop")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
	}

	return r0
}

type MockDataAvailabilityLayerClient_Stop_Call struct {
	*mock.Call
}

func (_e *MockDataAvailabilityLayerClient_Expecter) Stop() *MockDataAvailabilityLayerClient_Stop_Call {
	return &MockDataAvailabilityLayerClient_Stop_Call{Call: _e.mock.On("Stop")}
}

func (_c *MockDataAvailabilityLayerClient_Stop_Call) Run(run func()) *MockDataAvailabilityLayerClient_Stop_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockDataAvailabilityLayerClient_Stop_Call) Return(_a0 error) *MockDataAvailabilityLayerClient_Stop_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockDataAvailabilityLayerClient_Stop_Call) RunAndReturn(run func() error) *MockDataAvailabilityLayerClient_Stop_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockDataAvailabilityLayerClient) SubmitBatch(batch *types.Batch) da.ResultSubmitBatch {
	ret := _m.Called(batch)

	if len(ret) == 0 {
		panic("no return value specified for SubmitBatch")
	}

	var r0 da.ResultSubmitBatch
	if rf, ok := ret.Get(0).(func(*types.Batch) da.ResultSubmitBatch); ok {
		r0 = rf(batch)
	} else {
		r0 = ret.Get(0).(da.ResultSubmitBatch)
	}

	return r0
}

type MockDataAvailabilityLayerClient_SubmitBatch_Call struct {
	*mock.Call
}

func (_e *MockDataAvailabilityLayerClient_Expecter) SubmitBatch(batch interface{}) *MockDataAvailabilityLayerClient_SubmitBatch_Call {
	return &MockDataAvailabilityLayerClient_SubmitBatch_Call{Call: _e.mock.On("SubmitBatch", batch)}
}

func (_c *MockDataAvailabilityLayerClient_SubmitBatch_Call) Run(run func(batch *types.Batch)) *MockDataAvailabilityLayerClient_SubmitBatch_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*types.Batch))
	})
	return _c
}

func (_c *MockDataAvailabilityLayerClient_SubmitBatch_Call) Return(_a0 da.ResultSubmitBatch) *MockDataAvailabilityLayerClient_SubmitBatch_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockDataAvailabilityLayerClient_SubmitBatch_Call) RunAndReturn(run func(*types.Batch) da.ResultSubmitBatch) *MockDataAvailabilityLayerClient_SubmitBatch_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockDataAvailabilityLayerClient) WaitForSyncing() {
	_m.Called()
}

type MockDataAvailabilityLayerClient_WaitForSyncing_Call struct {
	*mock.Call
}

func (_e *MockDataAvailabilityLayerClient_Expecter) WaitForSyncing() *MockDataAvailabilityLayerClient_WaitForSyncing_Call {
	return &MockDataAvailabilityLayerClient_WaitForSyncing_Call{Call: _e.mock.On("WaitForSyncing")}
}

func (_c *MockDataAvailabilityLayerClient_WaitForSyncing_Call) Run(run func()) *MockDataAvailabilityLayerClient_WaitForSyncing_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockDataAvailabilityLayerClient_WaitForSyncing_Call) Return() *MockDataAvailabilityLayerClient_WaitForSyncing_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockDataAvailabilityLayerClient_WaitForSyncing_Call) RunAndReturn(run func()) *MockDataAvailabilityLayerClient_WaitForSyncing_Call {
	_c.Call.Return(run)
	return _c
}

func NewMockDataAvailabilityLayerClient(t interface {
	mock.TestingT
	Cleanup(func())
},
) *MockDataAvailabilityLayerClient {
	mock := &MockDataAvailabilityLayerClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
