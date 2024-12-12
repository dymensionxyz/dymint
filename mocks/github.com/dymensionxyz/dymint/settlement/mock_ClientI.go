package settlement

import (
	da "github.com/dymensionxyz/dymint/da"
	mock "github.com/stretchr/testify/mock"

	pubsub "github.com/tendermint/tendermint/libs/pubsub"

	rollapp "github.com/dymensionxyz/dymint/types/pb/dymensionxyz/dymension/rollapp"

	settlement "github.com/dymensionxyz/dymint/settlement"

	types "github.com/dymensionxyz/dymint/types"
)

type MockClientI struct {
	mock.Mock
}

type MockClientI_Expecter struct {
	mock *mock.Mock
}

func (_m *MockClientI) EXPECT() *MockClientI_Expecter {
	return &MockClientI_Expecter{mock: &_m.Mock}
}

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

type MockClientI_GetAllSequencers_Call struct {
	*mock.Call
}

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

func (_m *MockClientI) GetBatchAtHeight(index uint64) (*settlement.ResultRetrieveBatch, error) {
	ret := _m.Called(index)

	if len(ret) == 0 {
		panic("no return value specified for GetBatchAtHeight")
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

type MockClientI_GetBatchAtHeight_Call struct {
	*mock.Call
}

func (_e *MockClientI_Expecter) GetBatchAtHeight(index interface{}) *MockClientI_GetBatchAtHeight_Call {
	return &MockClientI_GetBatchAtHeight_Call{Call: _e.mock.On("GetBatchAtHeight", index)}
}

func (_c *MockClientI_GetBatchAtHeight_Call) Run(run func(index uint64)) *MockClientI_GetBatchAtHeight_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(uint64))
	})
	return _c
}

func (_c *MockClientI_GetBatchAtHeight_Call) Return(_a0 *settlement.ResultRetrieveBatch, _a1 error) *MockClientI_GetBatchAtHeight_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockClientI_GetBatchAtHeight_Call) RunAndReturn(run func(uint64) (*settlement.ResultRetrieveBatch, error)) *MockClientI_GetBatchAtHeight_Call {
	_c.Call.Return(run)
	return _c
}

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

type MockClientI_GetBatchAtIndex_Call struct {
	*mock.Call
}

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

type MockClientI_GetBondedSequencers_Call struct {
	*mock.Call
}

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

type MockClientI_GetLatestBatch_Call struct {
	*mock.Call
}

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

func (_m *MockClientI) GetLatestFinalizedHeight() (uint64, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetLatestFinalizedHeight")
	}

	var r0 uint64
	var r1 error
	if rf, ok := ret.Get(0).(func() (uint64, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() uint64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint64)
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type MockClientI_GetLatestFinalizedHeight_Call struct {
	*mock.Call
}

func (_e *MockClientI_Expecter) GetLatestFinalizedHeight() *MockClientI_GetLatestFinalizedHeight_Call {
	return &MockClientI_GetLatestFinalizedHeight_Call{Call: _e.mock.On("GetLatestFinalizedHeight")}
}

func (_c *MockClientI_GetLatestFinalizedHeight_Call) Run(run func()) *MockClientI_GetLatestFinalizedHeight_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockClientI_GetLatestFinalizedHeight_Call) Return(_a0 uint64, _a1 error) *MockClientI_GetLatestFinalizedHeight_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockClientI_GetLatestFinalizedHeight_Call) RunAndReturn(run func() (uint64, error)) *MockClientI_GetLatestFinalizedHeight_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockClientI) GetLatestHeight() (uint64, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetLatestHeight")
	}

	var r0 uint64
	var r1 error
	if rf, ok := ret.Get(0).(func() (uint64, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() uint64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint64)
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type MockClientI_GetLatestHeight_Call struct {
	*mock.Call
}

func (_e *MockClientI_Expecter) GetLatestHeight() *MockClientI_GetLatestHeight_Call {
	return &MockClientI_GetLatestHeight_Call{Call: _e.mock.On("GetLatestHeight")}
}

func (_c *MockClientI_GetLatestHeight_Call) Run(run func()) *MockClientI_GetLatestHeight_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockClientI_GetLatestHeight_Call) Return(_a0 uint64, _a1 error) *MockClientI_GetLatestHeight_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockClientI_GetLatestHeight_Call) RunAndReturn(run func() (uint64, error)) *MockClientI_GetLatestHeight_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockClientI) GetNextProposer() (*types.Sequencer, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetNextProposer")
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

type MockClientI_GetNextProposer_Call struct {
	*mock.Call
}

func (_e *MockClientI_Expecter) GetNextProposer() *MockClientI_GetNextProposer_Call {
	return &MockClientI_GetNextProposer_Call{Call: _e.mock.On("GetNextProposer")}
}

func (_c *MockClientI_GetNextProposer_Call) Run(run func()) *MockClientI_GetNextProposer_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockClientI_GetNextProposer_Call) Return(_a0 *types.Sequencer, _a1 error) *MockClientI_GetNextProposer_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockClientI_GetNextProposer_Call) RunAndReturn(run func() (*types.Sequencer, error)) *MockClientI_GetNextProposer_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockClientI) GetObsoleteDrs() ([]uint32, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetObsoleteDrs")
	}

	var r0 []uint32
	var r1 error
	if rf, ok := ret.Get(0).(func() ([]uint32, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() []uint32); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]uint32)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type MockClientI_GetObsoleteDrs_Call struct {
	*mock.Call
}

func (_e *MockClientI_Expecter) GetObsoleteDrs() *MockClientI_GetObsoleteDrs_Call {
	return &MockClientI_GetObsoleteDrs_Call{Call: _e.mock.On("GetObsoleteDrs")}
}

func (_c *MockClientI_GetObsoleteDrs_Call) Run(run func()) *MockClientI_GetObsoleteDrs_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockClientI_GetObsoleteDrs_Call) Return(_a0 []uint32, _a1 error) *MockClientI_GetObsoleteDrs_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockClientI_GetObsoleteDrs_Call) RunAndReturn(run func() ([]uint32, error)) *MockClientI_GetObsoleteDrs_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockClientI) GetProposerAtHeight(height int64) (*types.Sequencer, error) {
	ret := _m.Called(height)

	if len(ret) == 0 {
		panic("no return value specified for GetProposerAtHeight")
	}

	var r0 *types.Sequencer
	var r1 error
	if rf, ok := ret.Get(0).(func(int64) (*types.Sequencer, error)); ok {
		return rf(height)
	}
	if rf, ok := ret.Get(0).(func(int64) *types.Sequencer); ok {
		r0 = rf(height)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Sequencer)
		}
	}

	if rf, ok := ret.Get(1).(func(int64) error); ok {
		r1 = rf(height)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type MockClientI_GetProposerAtHeight_Call struct {
	*mock.Call
}

func (_e *MockClientI_Expecter) GetProposerAtHeight(height interface{}) *MockClientI_GetProposerAtHeight_Call {
	return &MockClientI_GetProposerAtHeight_Call{Call: _e.mock.On("GetProposerAtHeight", height)}
}

func (_c *MockClientI_GetProposerAtHeight_Call) Run(run func(height int64)) *MockClientI_GetProposerAtHeight_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(int64))
	})
	return _c
}

func (_c *MockClientI_GetProposerAtHeight_Call) Return(_a0 *types.Sequencer, _a1 error) *MockClientI_GetProposerAtHeight_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockClientI_GetProposerAtHeight_Call) RunAndReturn(run func(int64) (*types.Sequencer, error)) *MockClientI_GetProposerAtHeight_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockClientI) GetRollapp() (*types.Rollapp, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetRollapp")
	}

	var r0 *types.Rollapp
	var r1 error
	if rf, ok := ret.Get(0).(func() (*types.Rollapp, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() *types.Rollapp); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Rollapp)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type MockClientI_GetRollapp_Call struct {
	*mock.Call
}

func (_e *MockClientI_Expecter) GetRollapp() *MockClientI_GetRollapp_Call {
	return &MockClientI_GetRollapp_Call{Call: _e.mock.On("GetRollapp")}
}

func (_c *MockClientI_GetRollapp_Call) Run(run func()) *MockClientI_GetRollapp_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockClientI_GetRollapp_Call) Return(_a0 *types.Rollapp, _a1 error) *MockClientI_GetRollapp_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockClientI_GetRollapp_Call) RunAndReturn(run func() (*types.Rollapp, error)) *MockClientI_GetRollapp_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockClientI) GetSequencerByAddress(address string) (types.Sequencer, error) {
	ret := _m.Called(address)

	if len(ret) == 0 {
		panic("no return value specified for GetSequencerByAddress")
	}

	var r0 types.Sequencer
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (types.Sequencer, error)); ok {
		return rf(address)
	}
	if rf, ok := ret.Get(0).(func(string) types.Sequencer); ok {
		r0 = rf(address)
	} else {
		r0 = ret.Get(0).(types.Sequencer)
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(address)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type MockClientI_GetSequencerByAddress_Call struct {
	*mock.Call
}

func (_e *MockClientI_Expecter) GetSequencerByAddress(address interface{}) *MockClientI_GetSequencerByAddress_Call {
	return &MockClientI_GetSequencerByAddress_Call{Call: _e.mock.On("GetSequencerByAddress", address)}
}

func (_c *MockClientI_GetSequencerByAddress_Call) Run(run func(address string)) *MockClientI_GetSequencerByAddress_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *MockClientI_GetSequencerByAddress_Call) Return(_a0 types.Sequencer, _a1 error) *MockClientI_GetSequencerByAddress_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockClientI_GetSequencerByAddress_Call) RunAndReturn(run func(string) (types.Sequencer, error)) *MockClientI_GetSequencerByAddress_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockClientI) GetSignerBalance() (types.Balance, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetSignerBalance")
	}

	var r0 types.Balance
	var r1 error
	if rf, ok := ret.Get(0).(func() (types.Balance, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() types.Balance); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(types.Balance)
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type MockClientI_GetSignerBalance_Call struct {
	*mock.Call
}

func (_e *MockClientI_Expecter) GetSignerBalance() *MockClientI_GetSignerBalance_Call {
	return &MockClientI_GetSignerBalance_Call{Call: _e.mock.On("GetSignerBalance")}
}

func (_c *MockClientI_GetSignerBalance_Call) Run(run func()) *MockClientI_GetSignerBalance_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockClientI_GetSignerBalance_Call) Return(_a0 types.Balance, _a1 error) *MockClientI_GetSignerBalance_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockClientI_GetSignerBalance_Call) RunAndReturn(run func() (types.Balance, error)) *MockClientI_GetSignerBalance_Call {
	_c.Call.Return(run)
	return _c
}

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

type MockClientI_Init_Call struct {
	*mock.Call
}

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

type MockClientI_Start_Call struct {
	*mock.Call
}

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

type MockClientI_Stop_Call struct {
	*mock.Call
}

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

type MockClientI_SubmitBatch_Call struct {
	*mock.Call
}

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

func (_m *MockClientI) ValidateGenesisBridgeData(data rollapp.GenesisBridgeData) error {
	ret := _m.Called(data)

	if len(ret) == 0 {
		panic("no return value specified for ValidateGenesisBridgeData")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(rollapp.GenesisBridgeData) error); ok {
		r0 = rf(data)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type MockClientI_ValidateGenesisBridgeData_Call struct {
	*mock.Call
}

func (_e *MockClientI_Expecter) ValidateGenesisBridgeData(data interface{}) *MockClientI_ValidateGenesisBridgeData_Call {
	return &MockClientI_ValidateGenesisBridgeData_Call{Call: _e.mock.On("ValidateGenesisBridgeData", data)}
}

func (_c *MockClientI_ValidateGenesisBridgeData_Call) Run(run func(data rollapp.GenesisBridgeData)) *MockClientI_ValidateGenesisBridgeData_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(rollapp.GenesisBridgeData))
	})
	return _c
}

func (_c *MockClientI_ValidateGenesisBridgeData_Call) Return(_a0 error) *MockClientI_ValidateGenesisBridgeData_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockClientI_ValidateGenesisBridgeData_Call) RunAndReturn(run func(rollapp.GenesisBridgeData) error) *MockClientI_ValidateGenesisBridgeData_Call {
	_c.Call.Return(run)
	return _c
}

func NewMockClientI(t interface {
	mock.TestingT
	Cleanup(func())
},
) *MockClientI {
	mock := &MockClientI{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
