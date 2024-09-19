// Code generated by mockery v2.46.0. DO NOT EDIT.

package settlement

import (
	da "github.com/dymensionxyz/dymint/da"
	mock "github.com/stretchr/testify/mock"

	pubsub "github.com/tendermint/tendermint/libs/pubsub"

	settlement "github.com/dymensionxyz/dymint/settlement"

	types "github.com/dymensionxyz/dymint/types"
)

// MockClientI is an autogenerated mock type for the ClientI type
type MockClientI struct {
	mock.Mock
}

type MockClientI_Expecter struct {
	mock *mock.Mock
}

func (_m *MockClientI) EXPECT() *MockClientI_Expecter {
	return &MockClientI_Expecter{mock: &_m.Mock}
}

// CheckRotationInProgress provides a mock function with given fields:
func (_m *MockClientI) CheckRotationInProgress() (*types.Sequencer, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for CheckRotationInProgress")
	}

	var r0 *types.Sequencer
	var r1 error
	if rf, ok := ret.Get(0).(func() (*types.Sequencer, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() *types.Sequencer); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Sequencer)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockClientI_CheckRotationInProgress_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CheckRotationInProgress'
type MockClientI_CheckRotationInProgress_Call struct {
	*mock.Call
}

// CheckRotationInProgress is a helper method to define mock.On call
func (_e *MockClientI_Expecter) CheckRotationInProgress() *MockClientI_CheckRotationInProgress_Call {
	return &MockClientI_CheckRotationInProgress_Call{Call: _e.mock.On("CheckRotationInProgress")}
}

func (_c *MockClientI_CheckRotationInProgress_Call) Run(run func()) *MockClientI_CheckRotationInProgress_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockClientI_CheckRotationInProgress_Call) Return(_a0 *types.Sequencer, _a1 error) *MockClientI_CheckRotationInProgress_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockClientI_CheckRotationInProgress_Call) RunAndReturn(run func() (*types.Sequencer, error)) *MockClientI_CheckRotationInProgress_Call {
	_c.Call.Return(run)
	return _c
}

// GetAllSequencers provides a mock function with given fields:
func (_m *MockClientI) GetAllSequencers() ([]types.Sequencer, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetAllSequencers")
	}

	var r0 []types.Sequencer
	var r1 error
	if rf, ok := ret.Get(0).(func() ([]types.Sequencer, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() []types.Sequencer); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]types.Sequencer)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockClientI_GetAllSequencers_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetAllSequencers'
type MockClientI_GetAllSequencers_Call struct {
	*mock.Call
}

// GetAllSequencers is a helper method to define mock.On call
func (_e *MockClientI_Expecter) GetAllSequencers() *MockClientI_GetAllSequencers_Call {
	return &MockClientI_GetAllSequencers_Call{Call: _e.mock.On("GetAllSequencers")}
}

func (_c *MockClientI_GetAllSequencers_Call) Run(run func()) *MockClientI_GetAllSequencers_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockClientI_GetAllSequencers_Call) Return(_a0 []types.Sequencer, _a1 error) *MockClientI_GetAllSequencers_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockClientI_GetAllSequencers_Call) RunAndReturn(run func() ([]types.Sequencer, error)) *MockClientI_GetAllSequencers_Call {
	_c.Call.Return(run)
	return _c
}

// GetBatchAtIndex provides a mock function with given fields: index
func (_m *MockClientI) GetBatchAtIndex(index uint64) (*settlement.ResultRetrieveBatch, error) {
	ret := _m.Called(index)

	if len(ret) == 0 {
		panic("no return value specified for GetBatchAtIndex")
	}

	var r0 *settlement.ResultRetrieveBatch
	var r1 error
	if rf, ok := ret.Get(0).(func(uint64) (*settlement.ResultRetrieveBatch, error)); ok {
		return rf(index)
	}
	if rf, ok := ret.Get(0).(func(uint64) *settlement.ResultRetrieveBatch); ok {
		r0 = rf(index)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*settlement.ResultRetrieveBatch)
		}
	}

	if rf, ok := ret.Get(1).(func(uint64) error); ok {
		r1 = rf(index)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockClientI_GetBatchAtIndex_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetBatchAtIndex'
type MockClientI_GetBatchAtIndex_Call struct {
	*mock.Call
}

// GetBatchAtIndex is a helper method to define mock.On call
//   - index uint64
func (_e *MockClientI_Expecter) GetBatchAtIndex(index interface{}) *MockClientI_GetBatchAtIndex_Call {
	return &MockClientI_GetBatchAtIndex_Call{Call: _e.mock.On("GetBatchAtIndex", index)}
}

func (_c *MockClientI_GetBatchAtIndex_Call) Run(run func(index uint64)) *MockClientI_GetBatchAtIndex_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(uint64))
	})
	return _c
}

func (_c *MockClientI_GetBatchAtIndex_Call) Return(_a0 *settlement.ResultRetrieveBatch, _a1 error) *MockClientI_GetBatchAtIndex_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockClientI_GetBatchAtIndex_Call) RunAndReturn(run func(uint64) (*settlement.ResultRetrieveBatch, error)) *MockClientI_GetBatchAtIndex_Call {
	_c.Call.Return(run)
	return _c
}

// GetBondedSequencers provides a mock function with given fields:
func (_m *MockClientI) GetBondedSequencers() ([]types.Sequencer, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetBondedSequencers")
	}

	var r0 []types.Sequencer
	var r1 error
	if rf, ok := ret.Get(0).(func() ([]types.Sequencer, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() []types.Sequencer); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]types.Sequencer)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockClientI_GetBondedSequencers_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetBondedSequencers'
type MockClientI_GetBondedSequencers_Call struct {
	*mock.Call
}

// GetBondedSequencers is a helper method to define mock.On call
func (_e *MockClientI_Expecter) GetBondedSequencers() *MockClientI_GetBondedSequencers_Call {
	return &MockClientI_GetBondedSequencers_Call{Call: _e.mock.On("GetBondedSequencers")}
}

func (_c *MockClientI_GetBondedSequencers_Call) Run(run func()) *MockClientI_GetBondedSequencers_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockClientI_GetBondedSequencers_Call) Return(_a0 []types.Sequencer, _a1 error) *MockClientI_GetBondedSequencers_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockClientI_GetBondedSequencers_Call) RunAndReturn(run func() ([]types.Sequencer, error)) *MockClientI_GetBondedSequencers_Call {
	_c.Call.Return(run)
	return _c
}

// GetHeightState provides a mock function with given fields: _a0
func (_m *MockClientI) GetHeightState(_a0 uint64) (*settlement.ResultGetHeightState, error) {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for GetHeightState")
	}

	var r0 *settlement.ResultGetHeightState
	var r1 error
	if rf, ok := ret.Get(0).(func(uint64) (*settlement.ResultGetHeightState, error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func(uint64) *settlement.ResultGetHeightState); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*settlement.ResultGetHeightState)
		}
	}

	if rf, ok := ret.Get(1).(func(uint64) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockClientI_GetHeightState_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetHeightState'
type MockClientI_GetHeightState_Call struct {
	*mock.Call
}

// GetHeightState is a helper method to define mock.On call
//   - _a0 uint64
func (_e *MockClientI_Expecter) GetHeightState(_a0 interface{}) *MockClientI_GetHeightState_Call {
	return &MockClientI_GetHeightState_Call{Call: _e.mock.On("GetHeightState", _a0)}
}

func (_c *MockClientI_GetHeightState_Call) Run(run func(_a0 uint64)) *MockClientI_GetHeightState_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(uint64))
	})
	return _c
}

func (_c *MockClientI_GetHeightState_Call) Return(_a0 *settlement.ResultGetHeightState, _a1 error) *MockClientI_GetHeightState_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockClientI_GetHeightState_Call) RunAndReturn(run func(uint64) (*settlement.ResultGetHeightState, error)) *MockClientI_GetHeightState_Call {
	_c.Call.Return(run)
	return _c
}

// GetLatestBatch provides a mock function with given fields:
func (_m *MockClientI) GetLatestBatch() (*settlement.ResultRetrieveBatch, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetLatestBatch")
	}

	var r0 *settlement.ResultRetrieveBatch
	var r1 error
	if rf, ok := ret.Get(0).(func() (*settlement.ResultRetrieveBatch, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() *settlement.ResultRetrieveBatch); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*settlement.ResultRetrieveBatch)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockClientI_GetLatestBatch_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetLatestBatch'
type MockClientI_GetLatestBatch_Call struct {
	*mock.Call
}

// GetLatestBatch is a helper method to define mock.On call
func (_e *MockClientI_Expecter) GetLatestBatch() *MockClientI_GetLatestBatch_Call {
	return &MockClientI_GetLatestBatch_Call{Call: _e.mock.On("GetLatestBatch")}
}

func (_c *MockClientI_GetLatestBatch_Call) Run(run func()) *MockClientI_GetLatestBatch_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockClientI_GetLatestBatch_Call) Return(_a0 *settlement.ResultRetrieveBatch, _a1 error) *MockClientI_GetLatestBatch_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockClientI_GetLatestBatch_Call) RunAndReturn(run func() (*settlement.ResultRetrieveBatch, error)) *MockClientI_GetLatestBatch_Call {
	_c.Call.Return(run)
	return _c
}

// GetProposer provides a mock function with given fields:
func (_m *MockClientI) GetProposer() *types.Sequencer {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetProposer")
	}

	var r0 *types.Sequencer
	if rf, ok := ret.Get(0).(func() *types.Sequencer); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Sequencer)
		}
	}

	return r0
}

// MockClientI_GetProposer_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetProposer'
type MockClientI_GetProposer_Call struct {
	*mock.Call
}

// GetProposer is a helper method to define mock.On call
func (_e *MockClientI_Expecter) GetProposer() *MockClientI_GetProposer_Call {
	return &MockClientI_GetProposer_Call{Call: _e.mock.On("GetProposer")}
}

func (_c *MockClientI_GetProposer_Call) Run(run func()) *MockClientI_GetProposer_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockClientI_GetProposer_Call) Return(_a0 *types.Sequencer) *MockClientI_GetProposer_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockClientI_GetProposer_Call) RunAndReturn(run func() *types.Sequencer) *MockClientI_GetProposer_Call {
	_c.Call.Return(run)
	return _c
}

// Init provides a mock function with given fields: config, rollappId, _a2, logger, options
func (_m *MockClientI) Init(config settlement.Config, rollappId string, _a2 *pubsub.Server, logger types.Logger, options ...settlement.Option) error {
	_va := make([]interface{}, len(options))
	for _i := range options {
		_va[_i] = options[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, config, rollappId, _a2, logger)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Init")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(settlement.Config, string, *pubsub.Server, types.Logger, ...settlement.Option) error); ok {
		r0 = rf(config, rollappId, _a2, logger, options...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockClientI_Init_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Init'
type MockClientI_Init_Call struct {
	*mock.Call
}

// Init is a helper method to define mock.On call
//   - config settlement.Config
//   - rollappId string
//   - _a2 *pubsub.Server
//   - logger types.Logger
//   - options ...settlement.Option
func (_e *MockClientI_Expecter) Init(config interface{}, rollappId interface{}, _a2 interface{}, logger interface{}, options ...interface{}) *MockClientI_Init_Call {
	return &MockClientI_Init_Call{Call: _e.mock.On("Init",
		append([]interface{}{config, rollappId, _a2, logger}, options...)...)}
}

func (_c *MockClientI_Init_Call) Run(run func(config settlement.Config, rollappId string, _a2 *pubsub.Server, logger types.Logger, options ...settlement.Option)) *MockClientI_Init_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]settlement.Option, len(args)-4)
		for i, a := range args[4:] {
			if a != nil {
				variadicArgs[i] = a.(settlement.Option)
			}
		}
		run(args[0].(settlement.Config), args[1].(string), args[2].(*pubsub.Server), args[3].(types.Logger), variadicArgs...)
	})
	return _c
}

func (_c *MockClientI_Init_Call) Return(_a0 error) *MockClientI_Init_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockClientI_Init_Call) RunAndReturn(run func(settlement.Config, string, *pubsub.Server, types.Logger, ...settlement.Option) error) *MockClientI_Init_Call {
	_c.Call.Return(run)
	return _c
}

// Start provides a mock function with given fields:
func (_m *MockClientI) Start() error {
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

// MockClientI_Start_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Start'
type MockClientI_Start_Call struct {
	*mock.Call
}

// Start is a helper method to define mock.On call
func (_e *MockClientI_Expecter) Start() *MockClientI_Start_Call {
	return &MockClientI_Start_Call{Call: _e.mock.On("Start")}
}

func (_c *MockClientI_Start_Call) Run(run func()) *MockClientI_Start_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockClientI_Start_Call) Return(_a0 error) *MockClientI_Start_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockClientI_Start_Call) RunAndReturn(run func() error) *MockClientI_Start_Call {
	_c.Call.Return(run)
	return _c
}

// Stop provides a mock function with given fields:
func (_m *MockClientI) Stop() error {
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

// MockClientI_Stop_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Stop'
type MockClientI_Stop_Call struct {
	*mock.Call
}

// Stop is a helper method to define mock.On call
func (_e *MockClientI_Expecter) Stop() *MockClientI_Stop_Call {
	return &MockClientI_Stop_Call{Call: _e.mock.On("Stop")}
}

func (_c *MockClientI_Stop_Call) Run(run func()) *MockClientI_Stop_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockClientI_Stop_Call) Return(_a0 error) *MockClientI_Stop_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockClientI_Stop_Call) RunAndReturn(run func() error) *MockClientI_Stop_Call {
	_c.Call.Return(run)
	return _c
}

// SubmitBatch provides a mock function with given fields: batch, daClient, daResult
func (_m *MockClientI) SubmitBatch(batch *types.Batch, daClient da.Client, daResult *da.ResultSubmitBatch) error {
	ret := _m.Called(batch, daClient, daResult)

	if len(ret) == 0 {
		panic("no return value specified for SubmitBatch")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(*types.Batch, da.Client, *da.ResultSubmitBatch) error); ok {
		r0 = rf(batch, daClient, daResult)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockClientI_SubmitBatch_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SubmitBatch'
type MockClientI_SubmitBatch_Call struct {
	*mock.Call
}

// SubmitBatch is a helper method to define mock.On call
//   - batch *types.Batch
//   - daClient da.Client
//   - daResult *da.ResultSubmitBatch
func (_e *MockClientI_Expecter) SubmitBatch(batch interface{}, daClient interface{}, daResult interface{}) *MockClientI_SubmitBatch_Call {
	return &MockClientI_SubmitBatch_Call{Call: _e.mock.On("SubmitBatch", batch, daClient, daResult)}
}

func (_c *MockClientI_SubmitBatch_Call) Run(run func(batch *types.Batch, daClient da.Client, daResult *da.ResultSubmitBatch)) *MockClientI_SubmitBatch_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*types.Batch), args[1].(da.Client), args[2].(*da.ResultSubmitBatch))
	})
	return _c
}

func (_c *MockClientI_SubmitBatch_Call) Return(_a0 error) *MockClientI_SubmitBatch_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockClientI_SubmitBatch_Call) RunAndReturn(run func(*types.Batch, da.Client, *da.ResultSubmitBatch) error) *MockClientI_SubmitBatch_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockClientI creates a new instance of MockClientI. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockClientI(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockClientI {
	mock := &MockClientI{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
