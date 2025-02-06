// Code generated by mockery v2.50.2. DO NOT EDIT.

package da

import (
	da "github.com/dymensionxyz/dymint/da"
	mock "github.com/stretchr/testify/mock"

	pubsub "github.com/tendermint/tendermint/libs/pubsub"

	store "github.com/dymensionxyz/dymint/store"

	types "github.com/dymensionxyz/dymint/types"
)

// MockDataAvailabilityLayerClient is an autogenerated mock type for the DataAvailabilityLayerClient type
type MockDataAvailabilityLayerClient struct {
	mock.Mock
}

type MockDataAvailabilityLayerClient_Expecter struct {
	mock *mock.Mock
}

func (_m *MockDataAvailabilityLayerClient) EXPECT() *MockDataAvailabilityLayerClient_Expecter {
	return &MockDataAvailabilityLayerClient_Expecter{mock: &_m.Mock}
}

// CheckBatchAvailability provides a mock function with given fields: daPath
func (_m *MockDataAvailabilityLayerClient) CheckBatchAvailability(daPath string) da.ResultCheckBatch {
	ret := _m.Called(daPath)

	if len(ret) == 0 {
		panic("no return value specified for CheckBatchAvailability")
	}

	var r0 da.ResultCheckBatch
	if rf, ok := ret.Get(0).(func(string) da.ResultCheckBatch); ok {
		r0 = rf(daPath)
	} else {
		r0 = ret.Get(0).(da.ResultCheckBatch)
	}

	return r0
}

// MockDataAvailabilityLayerClient_CheckBatchAvailability_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CheckBatchAvailability'
type MockDataAvailabilityLayerClient_CheckBatchAvailability_Call struct {
	*mock.Call
}

// CheckBatchAvailability is a helper method to define mock.On call
//   - daPath string
func (_e *MockDataAvailabilityLayerClient_Expecter) CheckBatchAvailability(daPath interface{}) *MockDataAvailabilityLayerClient_CheckBatchAvailability_Call {
	return &MockDataAvailabilityLayerClient_CheckBatchAvailability_Call{Call: _e.mock.On("CheckBatchAvailability", daPath)}
}

func (_c *MockDataAvailabilityLayerClient_CheckBatchAvailability_Call) Run(run func(daPath string)) *MockDataAvailabilityLayerClient_CheckBatchAvailability_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *MockDataAvailabilityLayerClient_CheckBatchAvailability_Call) Return(_a0 da.ResultCheckBatch) *MockDataAvailabilityLayerClient_CheckBatchAvailability_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockDataAvailabilityLayerClient_CheckBatchAvailability_Call) RunAndReturn(run func(string) da.ResultCheckBatch) *MockDataAvailabilityLayerClient_CheckBatchAvailability_Call {
	_c.Call.Return(run)
	return _c
}

// DAPath provides a mock function with no fields
func (_m *MockDataAvailabilityLayerClient) DAPath() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for DAPath")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// MockDataAvailabilityLayerClient_DAPath_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DAPath'
type MockDataAvailabilityLayerClient_DAPath_Call struct {
	*mock.Call
}

// DAPath is a helper method to define mock.On call
func (_e *MockDataAvailabilityLayerClient_Expecter) DAPath() *MockDataAvailabilityLayerClient_DAPath_Call {
	return &MockDataAvailabilityLayerClient_DAPath_Call{Call: _e.mock.On("DAPath")}
}

func (_c *MockDataAvailabilityLayerClient_DAPath_Call) Run(run func()) *MockDataAvailabilityLayerClient_DAPath_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockDataAvailabilityLayerClient_DAPath_Call) Return(_a0 string) *MockDataAvailabilityLayerClient_DAPath_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockDataAvailabilityLayerClient_DAPath_Call) RunAndReturn(run func() string) *MockDataAvailabilityLayerClient_DAPath_Call {
	_c.Call.Return(run)
	return _c
}

// GetClientType provides a mock function with no fields
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

// MockDataAvailabilityLayerClient_GetClientType_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetClientType'
type MockDataAvailabilityLayerClient_GetClientType_Call struct {
	*mock.Call
}

// GetClientType is a helper method to define mock.On call
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

// GetMaxBlobSizeBytes provides a mock function with no fields
func (_m *MockDataAvailabilityLayerClient) GetMaxBlobSizeBytes() uint64 {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetMaxBlobSizeBytes")
	}

	var r0 uint64
	if rf, ok := ret.Get(0).(func() uint64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint64)
	}

	return r0
}

// MockDataAvailabilityLayerClient_GetMaxBlobSizeBytes_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetMaxBlobSizeBytes'
type MockDataAvailabilityLayerClient_GetMaxBlobSizeBytes_Call struct {
	*mock.Call
}

// GetMaxBlobSizeBytes is a helper method to define mock.On call
func (_e *MockDataAvailabilityLayerClient_Expecter) GetMaxBlobSizeBytes() *MockDataAvailabilityLayerClient_GetMaxBlobSizeBytes_Call {
	return &MockDataAvailabilityLayerClient_GetMaxBlobSizeBytes_Call{Call: _e.mock.On("GetMaxBlobSizeBytes")}
}

func (_c *MockDataAvailabilityLayerClient_GetMaxBlobSizeBytes_Call) Run(run func()) *MockDataAvailabilityLayerClient_GetMaxBlobSizeBytes_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockDataAvailabilityLayerClient_GetMaxBlobSizeBytes_Call) Return(_a0 uint64) *MockDataAvailabilityLayerClient_GetMaxBlobSizeBytes_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockDataAvailabilityLayerClient_GetMaxBlobSizeBytes_Call) RunAndReturn(run func() uint64) *MockDataAvailabilityLayerClient_GetMaxBlobSizeBytes_Call {
	_c.Call.Return(run)
	return _c
}

// GetSignerBalance provides a mock function with no fields
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
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockDataAvailabilityLayerClient_GetSignerBalance_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetSignerBalance'
type MockDataAvailabilityLayerClient_GetSignerBalance_Call struct {
	*mock.Call
}

// GetSignerBalance is a helper method to define mock.On call
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

// Init provides a mock function with given fields: config, pubsubServer, kvStore, logger, options
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
		r0 = ret.Error(0)
	}

	return r0
}

// MockDataAvailabilityLayerClient_Init_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Init'
type MockDataAvailabilityLayerClient_Init_Call struct {
	*mock.Call
}

// Init is a helper method to define mock.On call
//   - config []byte
//   - pubsubServer *pubsub.Server
//   - kvStore store.KV
//   - logger types.Logger
//   - options ...da.Option
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

// Start provides a mock function with no fields
func (_m *MockDataAvailabilityLayerClient) Start() error {
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

// MockDataAvailabilityLayerClient_Start_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Start'
type MockDataAvailabilityLayerClient_Start_Call struct {
	*mock.Call
}

// Start is a helper method to define mock.On call
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

// Stop provides a mock function with no fields
func (_m *MockDataAvailabilityLayerClient) Stop() error {
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

// MockDataAvailabilityLayerClient_Stop_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Stop'
type MockDataAvailabilityLayerClient_Stop_Call struct {
	*mock.Call
}

// Stop is a helper method to define mock.On call
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

// SubmitBatch provides a mock function with given fields: batch
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

// MockDataAvailabilityLayerClient_SubmitBatch_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SubmitBatch'
type MockDataAvailabilityLayerClient_SubmitBatch_Call struct {
	*mock.Call
}

// SubmitBatch is a helper method to define mock.On call
//   - batch *types.Batch
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

// NewMockDataAvailabilityLayerClient creates a new instance of MockDataAvailabilityLayerClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockDataAvailabilityLayerClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockDataAvailabilityLayerClient {
	mock := &MockDataAvailabilityLayerClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
