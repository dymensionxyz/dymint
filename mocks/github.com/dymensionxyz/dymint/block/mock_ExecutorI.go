// Code generated by mockery v2.50.2. DO NOT EDIT.

package block

import (
	abcitypes "github.com/tendermint/tendermint/abci/types"

	mock "github.com/stretchr/testify/mock"

	proto "github.com/gogo/protobuf/proto"

	state "github.com/tendermint/tendermint/proto/tendermint/state"

	tenderminttypes "github.com/tendermint/tendermint/types"

	types "github.com/dymensionxyz/dymint/types"
)

// MockExecutorI is an autogenerated mock type for the ExecutorI type
type MockExecutorI struct {
	mock.Mock
}

type MockExecutorI_Expecter struct {
	mock *mock.Mock
}

func (_m *MockExecutorI) EXPECT() *MockExecutorI_Expecter {
	return &MockExecutorI_Expecter{mock: &_m.Mock}
}

// AddConsensusMsgs provides a mock function with given fields: _a0
func (_m *MockExecutorI) AddConsensusMsgs(_a0 ...proto.Message) {
	_va := make([]interface{}, len(_a0))
	for _i := range _a0 {
		_va[_i] = _a0[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, _va...)
	_m.Called(_ca...)
}

// MockExecutorI_AddConsensusMsgs_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'AddConsensusMsgs'
type MockExecutorI_AddConsensusMsgs_Call struct {
	*mock.Call
}

// AddConsensusMsgs is a helper method to define mock.On call
//   - _a0 ...proto.Message
func (_e *MockExecutorI_Expecter) AddConsensusMsgs(_a0 ...interface{}) *MockExecutorI_AddConsensusMsgs_Call {
	return &MockExecutorI_AddConsensusMsgs_Call{Call: _e.mock.On("AddConsensusMsgs",
		append([]interface{}{}, _a0...)...)}
}

func (_c *MockExecutorI_AddConsensusMsgs_Call) Run(run func(_a0 ...proto.Message)) *MockExecutorI_AddConsensusMsgs_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]proto.Message, len(args)-0)
		for i, a := range args[0:] {
			if a != nil {
				variadicArgs[i] = a.(proto.Message)
			}
		}
		run(variadicArgs...)
	})
	return _c
}

func (_c *MockExecutorI_AddConsensusMsgs_Call) Return() *MockExecutorI_AddConsensusMsgs_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockExecutorI_AddConsensusMsgs_Call) RunAndReturn(run func(...proto.Message)) *MockExecutorI_AddConsensusMsgs_Call {
	_c.Run(run)
	return _c
}

// Commit provides a mock function with given fields: _a0, _a1, resp
func (_m *MockExecutorI) Commit(_a0 *types.State, _a1 *types.Block, resp *state.ABCIResponses) ([]byte, int64, error) {
	ret := _m.Called(_a0, _a1, resp)

	if len(ret) == 0 {
		panic("no return value specified for Commit")
	}

	var r0 []byte
	var r1 int64
	var r2 error
	if rf, ok := ret.Get(0).(func(*types.State, *types.Block, *state.ABCIResponses) ([]byte, int64, error)); ok {
		return rf(_a0, _a1, resp)
	}
	if rf, ok := ret.Get(0).(func(*types.State, *types.Block, *state.ABCIResponses) []byte); ok {
		r0 = rf(_a0, _a1, resp)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	if rf, ok := ret.Get(1).(func(*types.State, *types.Block, *state.ABCIResponses) int64); ok {
		r1 = rf(_a0, _a1, resp)
	} else {
		r1 = ret.Get(1).(int64)
	}

	if rf, ok := ret.Get(2).(func(*types.State, *types.Block, *state.ABCIResponses) error); ok {
		r2 = rf(_a0, _a1, resp)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// MockExecutorI_Commit_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Commit'
type MockExecutorI_Commit_Call struct {
	*mock.Call
}

// Commit is a helper method to define mock.On call
//   - _a0 *types.State
//   - _a1 *types.Block
//   - resp *state.ABCIResponses
func (_e *MockExecutorI_Expecter) Commit(_a0 interface{}, _a1 interface{}, resp interface{}) *MockExecutorI_Commit_Call {
	return &MockExecutorI_Commit_Call{Call: _e.mock.On("Commit", _a0, _a1, resp)}
}

func (_c *MockExecutorI_Commit_Call) Run(run func(_a0 *types.State, _a1 *types.Block, resp *state.ABCIResponses)) *MockExecutorI_Commit_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*types.State), args[1].(*types.Block), args[2].(*state.ABCIResponses))
	})
	return _c
}

func (_c *MockExecutorI_Commit_Call) Return(_a0 []byte, _a1 int64, _a2 error) *MockExecutorI_Commit_Call {
	_c.Call.Return(_a0, _a1, _a2)
	return _c
}

func (_c *MockExecutorI_Commit_Call) RunAndReturn(run func(*types.State, *types.Block, *state.ABCIResponses) ([]byte, int64, error)) *MockExecutorI_Commit_Call {
	_c.Call.Return(run)
	return _c
}

// CreateBlock provides a mock function with given fields: height, lastCommit, lastHeaderHash, nextSeqHash, _a4, maxBlockDataSizeBytes
func (_m *MockExecutorI) CreateBlock(height uint64, lastCommit *types.Commit, lastHeaderHash [32]byte, nextSeqHash [32]byte, _a4 *types.State, maxBlockDataSizeBytes uint64) *types.Block {
	ret := _m.Called(height, lastCommit, lastHeaderHash, nextSeqHash, _a4, maxBlockDataSizeBytes)

	if len(ret) == 0 {
		panic("no return value specified for CreateBlock")
	}

	var r0 *types.Block
	if rf, ok := ret.Get(0).(func(uint64, *types.Commit, [32]byte, [32]byte, *types.State, uint64) *types.Block); ok {
		r0 = rf(height, lastCommit, lastHeaderHash, nextSeqHash, _a4, maxBlockDataSizeBytes)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Block)
		}
	}

	return r0
}

// MockExecutorI_CreateBlock_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CreateBlock'
type MockExecutorI_CreateBlock_Call struct {
	*mock.Call
}

// CreateBlock is a helper method to define mock.On call
//   - height uint64
//   - lastCommit *types.Commit
//   - lastHeaderHash [32]byte
//   - nextSeqHash [32]byte
//   - _a4 *types.State
//   - maxBlockDataSizeBytes uint64
func (_e *MockExecutorI_Expecter) CreateBlock(height interface{}, lastCommit interface{}, lastHeaderHash interface{}, nextSeqHash interface{}, _a4 interface{}, maxBlockDataSizeBytes interface{}) *MockExecutorI_CreateBlock_Call {
	return &MockExecutorI_CreateBlock_Call{Call: _e.mock.On("CreateBlock", height, lastCommit, lastHeaderHash, nextSeqHash, _a4, maxBlockDataSizeBytes)}
}

func (_c *MockExecutorI_CreateBlock_Call) Run(run func(height uint64, lastCommit *types.Commit, lastHeaderHash [32]byte, nextSeqHash [32]byte, _a4 *types.State, maxBlockDataSizeBytes uint64)) *MockExecutorI_CreateBlock_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(uint64), args[1].(*types.Commit), args[2].([32]byte), args[3].([32]byte), args[4].(*types.State), args[5].(uint64))
	})
	return _c
}

func (_c *MockExecutorI_CreateBlock_Call) Return(_a0 *types.Block) *MockExecutorI_CreateBlock_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockExecutorI_CreateBlock_Call) RunAndReturn(run func(uint64, *types.Commit, [32]byte, [32]byte, *types.State, uint64) *types.Block) *MockExecutorI_CreateBlock_Call {
	_c.Call.Return(run)
	return _c
}

// ExecuteBlock provides a mock function with given fields: _a0
func (_m *MockExecutorI) ExecuteBlock(_a0 *types.Block) (*state.ABCIResponses, error) {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for ExecuteBlock")
	}

	var r0 *state.ABCIResponses
	var r1 error
	if rf, ok := ret.Get(0).(func(*types.Block) (*state.ABCIResponses, error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func(*types.Block) *state.ABCIResponses); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*state.ABCIResponses)
		}
	}

	if rf, ok := ret.Get(1).(func(*types.Block) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockExecutorI_ExecuteBlock_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ExecuteBlock'
type MockExecutorI_ExecuteBlock_Call struct {
	*mock.Call
}

// ExecuteBlock is a helper method to define mock.On call
//   - _a0 *types.Block
func (_e *MockExecutorI_Expecter) ExecuteBlock(_a0 interface{}) *MockExecutorI_ExecuteBlock_Call {
	return &MockExecutorI_ExecuteBlock_Call{Call: _e.mock.On("ExecuteBlock", _a0)}
}

func (_c *MockExecutorI_ExecuteBlock_Call) Run(run func(_a0 *types.Block)) *MockExecutorI_ExecuteBlock_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*types.Block))
	})
	return _c
}

func (_c *MockExecutorI_ExecuteBlock_Call) Return(_a0 *state.ABCIResponses, _a1 error) *MockExecutorI_ExecuteBlock_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockExecutorI_ExecuteBlock_Call) RunAndReturn(run func(*types.Block) (*state.ABCIResponses, error)) *MockExecutorI_ExecuteBlock_Call {
	_c.Call.Return(run)
	return _c
}

// GetAppInfo provides a mock function with no fields
func (_m *MockExecutorI) GetAppInfo() (*abcitypes.ResponseInfo, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetAppInfo")
	}

	var r0 *abcitypes.ResponseInfo
	var r1 error
	if rf, ok := ret.Get(0).(func() (*abcitypes.ResponseInfo, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() *abcitypes.ResponseInfo); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*abcitypes.ResponseInfo)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockExecutorI_GetAppInfo_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetAppInfo'
type MockExecutorI_GetAppInfo_Call struct {
	*mock.Call
}

// GetAppInfo is a helper method to define mock.On call
func (_e *MockExecutorI_Expecter) GetAppInfo() *MockExecutorI_GetAppInfo_Call {
	return &MockExecutorI_GetAppInfo_Call{Call: _e.mock.On("GetAppInfo")}
}

func (_c *MockExecutorI_GetAppInfo_Call) Run(run func()) *MockExecutorI_GetAppInfo_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockExecutorI_GetAppInfo_Call) Return(_a0 *abcitypes.ResponseInfo, _a1 error) *MockExecutorI_GetAppInfo_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockExecutorI_GetAppInfo_Call) RunAndReturn(run func() (*abcitypes.ResponseInfo, error)) *MockExecutorI_GetAppInfo_Call {
	_c.Call.Return(run)
	return _c
}

// GetConsensusMsgs provides a mock function with no fields
func (_m *MockExecutorI) GetConsensusMsgs() []proto.Message {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetConsensusMsgs")
	}

	var r0 []proto.Message
	if rf, ok := ret.Get(0).(func() []proto.Message); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]proto.Message)
		}
	}

	return r0
}

// MockExecutorI_GetConsensusMsgs_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetConsensusMsgs'
type MockExecutorI_GetConsensusMsgs_Call struct {
	*mock.Call
}

// GetConsensusMsgs is a helper method to define mock.On call
func (_e *MockExecutorI_Expecter) GetConsensusMsgs() *MockExecutorI_GetConsensusMsgs_Call {
	return &MockExecutorI_GetConsensusMsgs_Call{Call: _e.mock.On("GetConsensusMsgs")}
}

func (_c *MockExecutorI_GetConsensusMsgs_Call) Run(run func()) *MockExecutorI_GetConsensusMsgs_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockExecutorI_GetConsensusMsgs_Call) Return(_a0 []proto.Message) *MockExecutorI_GetConsensusMsgs_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockExecutorI_GetConsensusMsgs_Call) RunAndReturn(run func() []proto.Message) *MockExecutorI_GetConsensusMsgs_Call {
	_c.Call.Return(run)
	return _c
}

// InitChain provides a mock function with given fields: genesis, genesisChecksum, valset
func (_m *MockExecutorI) InitChain(genesis *tenderminttypes.GenesisDoc, genesisChecksum string, valset []*tenderminttypes.Validator) (*abcitypes.ResponseInitChain, error) {
	ret := _m.Called(genesis, genesisChecksum, valset)

	if len(ret) == 0 {
		panic("no return value specified for InitChain")
	}

	var r0 *abcitypes.ResponseInitChain
	var r1 error
	if rf, ok := ret.Get(0).(func(*tenderminttypes.GenesisDoc, string, []*tenderminttypes.Validator) (*abcitypes.ResponseInitChain, error)); ok {
		return rf(genesis, genesisChecksum, valset)
	}
	if rf, ok := ret.Get(0).(func(*tenderminttypes.GenesisDoc, string, []*tenderminttypes.Validator) *abcitypes.ResponseInitChain); ok {
		r0 = rf(genesis, genesisChecksum, valset)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*abcitypes.ResponseInitChain)
		}
	}

	if rf, ok := ret.Get(1).(func(*tenderminttypes.GenesisDoc, string, []*tenderminttypes.Validator) error); ok {
		r1 = rf(genesis, genesisChecksum, valset)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockExecutorI_InitChain_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'InitChain'
type MockExecutorI_InitChain_Call struct {
	*mock.Call
}

// InitChain is a helper method to define mock.On call
//   - genesis *tenderminttypes.GenesisDoc
//   - genesisChecksum string
//   - valset []*tenderminttypes.Validator
func (_e *MockExecutorI_Expecter) InitChain(genesis interface{}, genesisChecksum interface{}, valset interface{}) *MockExecutorI_InitChain_Call {
	return &MockExecutorI_InitChain_Call{Call: _e.mock.On("InitChain", genesis, genesisChecksum, valset)}
}

func (_c *MockExecutorI_InitChain_Call) Run(run func(genesis *tenderminttypes.GenesisDoc, genesisChecksum string, valset []*tenderminttypes.Validator)) *MockExecutorI_InitChain_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*tenderminttypes.GenesisDoc), args[1].(string), args[2].([]*tenderminttypes.Validator))
	})
	return _c
}

func (_c *MockExecutorI_InitChain_Call) Return(_a0 *abcitypes.ResponseInitChain, _a1 error) *MockExecutorI_InitChain_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockExecutorI_InitChain_Call) RunAndReturn(run func(*tenderminttypes.GenesisDoc, string, []*tenderminttypes.Validator) (*abcitypes.ResponseInitChain, error)) *MockExecutorI_InitChain_Call {
	_c.Call.Return(run)
	return _c
}

// UpdateMempoolAfterInitChain provides a mock function with given fields: s
func (_m *MockExecutorI) UpdateMempoolAfterInitChain(s *types.State) {
	_m.Called(s)
}

// MockExecutorI_UpdateMempoolAfterInitChain_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateMempoolAfterInitChain'
type MockExecutorI_UpdateMempoolAfterInitChain_Call struct {
	*mock.Call
}

// UpdateMempoolAfterInitChain is a helper method to define mock.On call
//   - s *types.State
func (_e *MockExecutorI_Expecter) UpdateMempoolAfterInitChain(s interface{}) *MockExecutorI_UpdateMempoolAfterInitChain_Call {
	return &MockExecutorI_UpdateMempoolAfterInitChain_Call{Call: _e.mock.On("UpdateMempoolAfterInitChain", s)}
}

func (_c *MockExecutorI_UpdateMempoolAfterInitChain_Call) Run(run func(s *types.State)) *MockExecutorI_UpdateMempoolAfterInitChain_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*types.State))
	})
	return _c
}

func (_c *MockExecutorI_UpdateMempoolAfterInitChain_Call) Return() *MockExecutorI_UpdateMempoolAfterInitChain_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockExecutorI_UpdateMempoolAfterInitChain_Call) RunAndReturn(run func(*types.State)) *MockExecutorI_UpdateMempoolAfterInitChain_Call {
	_c.Run(run)
	return _c
}

// UpdateProposerFromBlock provides a mock function with given fields: s, seqSet, _a2
func (_m *MockExecutorI) UpdateProposerFromBlock(s *types.State, seqSet *types.SequencerSet, _a2 *types.Block) bool {
	ret := _m.Called(s, seqSet, _a2)

	if len(ret) == 0 {
		panic("no return value specified for UpdateProposerFromBlock")
	}

	var r0 bool
	if rf, ok := ret.Get(0).(func(*types.State, *types.SequencerSet, *types.Block) bool); ok {
		r0 = rf(s, seqSet, _a2)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// MockExecutorI_UpdateProposerFromBlock_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateProposerFromBlock'
type MockExecutorI_UpdateProposerFromBlock_Call struct {
	*mock.Call
}

// UpdateProposerFromBlock is a helper method to define mock.On call
//   - s *types.State
//   - seqSet *types.SequencerSet
//   - _a2 *types.Block
func (_e *MockExecutorI_Expecter) UpdateProposerFromBlock(s interface{}, seqSet interface{}, _a2 interface{}) *MockExecutorI_UpdateProposerFromBlock_Call {
	return &MockExecutorI_UpdateProposerFromBlock_Call{Call: _e.mock.On("UpdateProposerFromBlock", s, seqSet, _a2)}
}

func (_c *MockExecutorI_UpdateProposerFromBlock_Call) Run(run func(s *types.State, seqSet *types.SequencerSet, _a2 *types.Block)) *MockExecutorI_UpdateProposerFromBlock_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*types.State), args[1].(*types.SequencerSet), args[2].(*types.Block))
	})
	return _c
}

func (_c *MockExecutorI_UpdateProposerFromBlock_Call) Return(_a0 bool) *MockExecutorI_UpdateProposerFromBlock_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockExecutorI_UpdateProposerFromBlock_Call) RunAndReturn(run func(*types.State, *types.SequencerSet, *types.Block) bool) *MockExecutorI_UpdateProposerFromBlock_Call {
	_c.Call.Return(run)
	return _c
}

// UpdateStateAfterCommit provides a mock function with given fields: s, resp, appHash, _a3
func (_m *MockExecutorI) UpdateStateAfterCommit(s *types.State, resp *state.ABCIResponses, appHash []byte, _a3 *types.Block) {
	_m.Called(s, resp, appHash, _a3)
}

// MockExecutorI_UpdateStateAfterCommit_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateStateAfterCommit'
type MockExecutorI_UpdateStateAfterCommit_Call struct {
	*mock.Call
}

// UpdateStateAfterCommit is a helper method to define mock.On call
//   - s *types.State
//   - resp *state.ABCIResponses
//   - appHash []byte
//   - _a3 *types.Block
func (_e *MockExecutorI_Expecter) UpdateStateAfterCommit(s interface{}, resp interface{}, appHash interface{}, _a3 interface{}) *MockExecutorI_UpdateStateAfterCommit_Call {
	return &MockExecutorI_UpdateStateAfterCommit_Call{Call: _e.mock.On("UpdateStateAfterCommit", s, resp, appHash, _a3)}
}

func (_c *MockExecutorI_UpdateStateAfterCommit_Call) Run(run func(s *types.State, resp *state.ABCIResponses, appHash []byte, _a3 *types.Block)) *MockExecutorI_UpdateStateAfterCommit_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*types.State), args[1].(*state.ABCIResponses), args[2].([]byte), args[3].(*types.Block))
	})
	return _c
}

func (_c *MockExecutorI_UpdateStateAfterCommit_Call) Return() *MockExecutorI_UpdateStateAfterCommit_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockExecutorI_UpdateStateAfterCommit_Call) RunAndReturn(run func(*types.State, *state.ABCIResponses, []byte, *types.Block)) *MockExecutorI_UpdateStateAfterCommit_Call {
	_c.Run(run)
	return _c
}

// UpdateStateAfterInitChain provides a mock function with given fields: s, res
func (_m *MockExecutorI) UpdateStateAfterInitChain(s *types.State, res *abcitypes.ResponseInitChain) {
	_m.Called(s, res)
}

// MockExecutorI_UpdateStateAfterInitChain_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateStateAfterInitChain'
type MockExecutorI_UpdateStateAfterInitChain_Call struct {
	*mock.Call
}

// UpdateStateAfterInitChain is a helper method to define mock.On call
//   - s *types.State
//   - res *abcitypes.ResponseInitChain
func (_e *MockExecutorI_Expecter) UpdateStateAfterInitChain(s interface{}, res interface{}) *MockExecutorI_UpdateStateAfterInitChain_Call {
	return &MockExecutorI_UpdateStateAfterInitChain_Call{Call: _e.mock.On("UpdateStateAfterInitChain", s, res)}
}

func (_c *MockExecutorI_UpdateStateAfterInitChain_Call) Run(run func(s *types.State, res *abcitypes.ResponseInitChain)) *MockExecutorI_UpdateStateAfterInitChain_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*types.State), args[1].(*abcitypes.ResponseInitChain))
	})
	return _c
}

func (_c *MockExecutorI_UpdateStateAfterInitChain_Call) Return() *MockExecutorI_UpdateStateAfterInitChain_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockExecutorI_UpdateStateAfterInitChain_Call) RunAndReturn(run func(*types.State, *abcitypes.ResponseInitChain)) *MockExecutorI_UpdateStateAfterInitChain_Call {
	_c.Run(run)
	return _c
}

// NewMockExecutorI creates a new instance of MockExecutorI. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockExecutorI(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockExecutorI {
	mock := &MockExecutorI{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
