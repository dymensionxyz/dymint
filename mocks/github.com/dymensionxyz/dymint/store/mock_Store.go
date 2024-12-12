package store

import (
	cid "github.com/ipfs/go-cid"
	mock "github.com/stretchr/testify/mock"

	state "github.com/tendermint/tendermint/proto/tendermint/state"

	store "github.com/dymensionxyz/dymint/store"

	types "github.com/dymensionxyz/dymint/types"
)

type MockStore struct {
	mock.Mock
}

type MockStore_Expecter struct {
	mock *mock.Mock
}

func (_m *MockStore) EXPECT() *MockStore_Expecter {
	return &MockStore_Expecter{mock: &_m.Mock}
}

func (_m *MockStore) Close() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Close")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type MockStore_Close_Call struct {
	*mock.Call
}

func (_e *MockStore_Expecter) Close() *MockStore_Close_Call {
	return &MockStore_Close_Call{Call: _e.mock.On("Close")}
}

func (_c *MockStore_Close_Call) Run(run func()) *MockStore_Close_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockStore_Close_Call) Return(_a0 error) *MockStore_Close_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockStore_Close_Call) RunAndReturn(run func() error) *MockStore_Close_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockStore) LoadBaseHeight() (uint64, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for LoadBaseHeight")
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

type MockStore_LoadBaseHeight_Call struct {
	*mock.Call
}

func (_e *MockStore_Expecter) LoadBaseHeight() *MockStore_LoadBaseHeight_Call {
	return &MockStore_LoadBaseHeight_Call{Call: _e.mock.On("LoadBaseHeight")}
}

func (_c *MockStore_LoadBaseHeight_Call) Run(run func()) *MockStore_LoadBaseHeight_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockStore_LoadBaseHeight_Call) Return(_a0 uint64, _a1 error) *MockStore_LoadBaseHeight_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockStore_LoadBaseHeight_Call) RunAndReturn(run func() (uint64, error)) *MockStore_LoadBaseHeight_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockStore) LoadBlock(height uint64) (*types.Block, error) {
	ret := _m.Called(height)

	if len(ret) == 0 {
		panic("no return value specified for LoadBlock")
	}

	var r0 *types.Block
	var r1 error
	if rf, ok := ret.Get(0).(func(uint64) (*types.Block, error)); ok {
		return rf(height)
	}
	if rf, ok := ret.Get(0).(func(uint64) *types.Block); ok {
		r0 = rf(height)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Block)
		}
	}

	if rf, ok := ret.Get(1).(func(uint64) error); ok {
		r1 = rf(height)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type MockStore_LoadBlock_Call struct {
	*mock.Call
}

func (_e *MockStore_Expecter) LoadBlock(height interface{}) *MockStore_LoadBlock_Call {
	return &MockStore_LoadBlock_Call{Call: _e.mock.On("LoadBlock", height)}
}

func (_c *MockStore_LoadBlock_Call) Run(run func(height uint64)) *MockStore_LoadBlock_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(uint64))
	})
	return _c
}

func (_c *MockStore_LoadBlock_Call) Return(_a0 *types.Block, _a1 error) *MockStore_LoadBlock_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockStore_LoadBlock_Call) RunAndReturn(run func(uint64) (*types.Block, error)) *MockStore_LoadBlock_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockStore) LoadBlockByHash(hash [32]byte) (*types.Block, error) {
	ret := _m.Called(hash)

	if len(ret) == 0 {
		panic("no return value specified for LoadBlockByHash")
	}

	var r0 *types.Block
	var r1 error
	if rf, ok := ret.Get(0).(func([32]byte) (*types.Block, error)); ok {
		return rf(hash)
	}
	if rf, ok := ret.Get(0).(func([32]byte) *types.Block); ok {
		r0 = rf(hash)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Block)
		}
	}

	if rf, ok := ret.Get(1).(func([32]byte) error); ok {
		r1 = rf(hash)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type MockStore_LoadBlockByHash_Call struct {
	*mock.Call
}

func (_e *MockStore_Expecter) LoadBlockByHash(hash interface{}) *MockStore_LoadBlockByHash_Call {
	return &MockStore_LoadBlockByHash_Call{Call: _e.mock.On("LoadBlockByHash", hash)}
}

func (_c *MockStore_LoadBlockByHash_Call) Run(run func(hash [32]byte)) *MockStore_LoadBlockByHash_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].([32]byte))
	})
	return _c
}

func (_c *MockStore_LoadBlockByHash_Call) Return(_a0 *types.Block, _a1 error) *MockStore_LoadBlockByHash_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockStore_LoadBlockByHash_Call) RunAndReturn(run func([32]byte) (*types.Block, error)) *MockStore_LoadBlockByHash_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockStore) LoadBlockCid(height uint64) (cid.Cid, error) {
	ret := _m.Called(height)

	if len(ret) == 0 {
		panic("no return value specified for LoadBlockCid")
	}

	var r0 cid.Cid
	var r1 error
	if rf, ok := ret.Get(0).(func(uint64) (cid.Cid, error)); ok {
		return rf(height)
	}
	if rf, ok := ret.Get(0).(func(uint64) cid.Cid); ok {
		r0 = rf(height)
	} else {
		r0 = ret.Get(0).(cid.Cid)
	}

	if rf, ok := ret.Get(1).(func(uint64) error); ok {
		r1 = rf(height)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type MockStore_LoadBlockCid_Call struct {
	*mock.Call
}

func (_e *MockStore_Expecter) LoadBlockCid(height interface{}) *MockStore_LoadBlockCid_Call {
	return &MockStore_LoadBlockCid_Call{Call: _e.mock.On("LoadBlockCid", height)}
}

func (_c *MockStore_LoadBlockCid_Call) Run(run func(height uint64)) *MockStore_LoadBlockCid_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(uint64))
	})
	return _c
}

func (_c *MockStore_LoadBlockCid_Call) Return(_a0 cid.Cid, _a1 error) *MockStore_LoadBlockCid_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockStore_LoadBlockCid_Call) RunAndReturn(run func(uint64) (cid.Cid, error)) *MockStore_LoadBlockCid_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockStore) LoadBlockResponses(height uint64) (*state.ABCIResponses, error) {
	ret := _m.Called(height)

	if len(ret) == 0 {
		panic("no return value specified for LoadBlockResponses")
	}

	var r0 *state.ABCIResponses
	var r1 error
	if rf, ok := ret.Get(0).(func(uint64) (*state.ABCIResponses, error)); ok {
		return rf(height)
	}
	if rf, ok := ret.Get(0).(func(uint64) *state.ABCIResponses); ok {
		r0 = rf(height)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*state.ABCIResponses)
		}
	}

	if rf, ok := ret.Get(1).(func(uint64) error); ok {
		r1 = rf(height)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type MockStore_LoadBlockResponses_Call struct {
	*mock.Call
}

func (_e *MockStore_Expecter) LoadBlockResponses(height interface{}) *MockStore_LoadBlockResponses_Call {
	return &MockStore_LoadBlockResponses_Call{Call: _e.mock.On("LoadBlockResponses", height)}
}

func (_c *MockStore_LoadBlockResponses_Call) Run(run func(height uint64)) *MockStore_LoadBlockResponses_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(uint64))
	})
	return _c
}

func (_c *MockStore_LoadBlockResponses_Call) Return(_a0 *state.ABCIResponses, _a1 error) *MockStore_LoadBlockResponses_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockStore_LoadBlockResponses_Call) RunAndReturn(run func(uint64) (*state.ABCIResponses, error)) *MockStore_LoadBlockResponses_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockStore) LoadBlockSource(height uint64) (types.BlockSource, error) {
	ret := _m.Called(height)

	if len(ret) == 0 {
		panic("no return value specified for LoadBlockSource")
	}

	var r0 types.BlockSource
	var r1 error
	if rf, ok := ret.Get(0).(func(uint64) (types.BlockSource, error)); ok {
		return rf(height)
	}
	if rf, ok := ret.Get(0).(func(uint64) types.BlockSource); ok {
		r0 = rf(height)
	} else {
		r0 = ret.Get(0).(types.BlockSource)
	}

	if rf, ok := ret.Get(1).(func(uint64) error); ok {
		r1 = rf(height)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type MockStore_LoadBlockSource_Call struct {
	*mock.Call
}

func (_e *MockStore_Expecter) LoadBlockSource(height interface{}) *MockStore_LoadBlockSource_Call {
	return &MockStore_LoadBlockSource_Call{Call: _e.mock.On("LoadBlockSource", height)}
}

func (_c *MockStore_LoadBlockSource_Call) Run(run func(height uint64)) *MockStore_LoadBlockSource_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(uint64))
	})
	return _c
}

func (_c *MockStore_LoadBlockSource_Call) Return(_a0 types.BlockSource, _a1 error) *MockStore_LoadBlockSource_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockStore_LoadBlockSource_Call) RunAndReturn(run func(uint64) (types.BlockSource, error)) *MockStore_LoadBlockSource_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockStore) LoadBlockSyncBaseHeight() (uint64, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for LoadBlockSyncBaseHeight")
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

type MockStore_LoadBlockSyncBaseHeight_Call struct {
	*mock.Call
}

func (_e *MockStore_Expecter) LoadBlockSyncBaseHeight() *MockStore_LoadBlockSyncBaseHeight_Call {
	return &MockStore_LoadBlockSyncBaseHeight_Call{Call: _e.mock.On("LoadBlockSyncBaseHeight")}
}

func (_c *MockStore_LoadBlockSyncBaseHeight_Call) Run(run func()) *MockStore_LoadBlockSyncBaseHeight_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockStore_LoadBlockSyncBaseHeight_Call) Return(_a0 uint64, _a1 error) *MockStore_LoadBlockSyncBaseHeight_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockStore_LoadBlockSyncBaseHeight_Call) RunAndReturn(run func() (uint64, error)) *MockStore_LoadBlockSyncBaseHeight_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockStore) LoadCommit(height uint64) (*types.Commit, error) {
	ret := _m.Called(height)

	if len(ret) == 0 {
		panic("no return value specified for LoadCommit")
	}

	var r0 *types.Commit
	var r1 error
	if rf, ok := ret.Get(0).(func(uint64) (*types.Commit, error)); ok {
		return rf(height)
	}
	if rf, ok := ret.Get(0).(func(uint64) *types.Commit); ok {
		r0 = rf(height)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Commit)
		}
	}

	if rf, ok := ret.Get(1).(func(uint64) error); ok {
		r1 = rf(height)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type MockStore_LoadCommit_Call struct {
	*mock.Call
}

func (_e *MockStore_Expecter) LoadCommit(height interface{}) *MockStore_LoadCommit_Call {
	return &MockStore_LoadCommit_Call{Call: _e.mock.On("LoadCommit", height)}
}

func (_c *MockStore_LoadCommit_Call) Run(run func(height uint64)) *MockStore_LoadCommit_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(uint64))
	})
	return _c
}

func (_c *MockStore_LoadCommit_Call) Return(_a0 *types.Commit, _a1 error) *MockStore_LoadCommit_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockStore_LoadCommit_Call) RunAndReturn(run func(uint64) (*types.Commit, error)) *MockStore_LoadCommit_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockStore) LoadCommitByHash(hash [32]byte) (*types.Commit, error) {
	ret := _m.Called(hash)

	if len(ret) == 0 {
		panic("no return value specified for LoadCommitByHash")
	}

	var r0 *types.Commit
	var r1 error
	if rf, ok := ret.Get(0).(func([32]byte) (*types.Commit, error)); ok {
		return rf(hash)
	}
	if rf, ok := ret.Get(0).(func([32]byte) *types.Commit); ok {
		r0 = rf(hash)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Commit)
		}
	}

	if rf, ok := ret.Get(1).(func([32]byte) error); ok {
		r1 = rf(hash)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type MockStore_LoadCommitByHash_Call struct {
	*mock.Call
}

func (_e *MockStore_Expecter) LoadCommitByHash(hash interface{}) *MockStore_LoadCommitByHash_Call {
	return &MockStore_LoadCommitByHash_Call{Call: _e.mock.On("LoadCommitByHash", hash)}
}

func (_c *MockStore_LoadCommitByHash_Call) Run(run func(hash [32]byte)) *MockStore_LoadCommitByHash_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].([32]byte))
	})
	return _c
}

func (_c *MockStore_LoadCommitByHash_Call) Return(_a0 *types.Commit, _a1 error) *MockStore_LoadCommitByHash_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockStore_LoadCommitByHash_Call) RunAndReturn(run func([32]byte) (*types.Commit, error)) *MockStore_LoadCommitByHash_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockStore) LoadDRSVersion(height uint64) (uint32, error) {
	ret := _m.Called(height)

	if len(ret) == 0 {
		panic("no return value specified for LoadDRSVersion")
	}

	var r0 uint32
	var r1 error
	if rf, ok := ret.Get(0).(func(uint64) (uint32, error)); ok {
		return rf(height)
	}
	if rf, ok := ret.Get(0).(func(uint64) uint32); ok {
		r0 = rf(height)
	} else {
		r0 = ret.Get(0).(uint32)
	}

	if rf, ok := ret.Get(1).(func(uint64) error); ok {
		r1 = rf(height)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type MockStore_LoadDRSVersion_Call struct {
	*mock.Call
}

func (_e *MockStore_Expecter) LoadDRSVersion(height interface{}) *MockStore_LoadDRSVersion_Call {
	return &MockStore_LoadDRSVersion_Call{Call: _e.mock.On("LoadDRSVersion", height)}
}

func (_c *MockStore_LoadDRSVersion_Call) Run(run func(height uint64)) *MockStore_LoadDRSVersion_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(uint64))
	})
	return _c
}

func (_c *MockStore_LoadDRSVersion_Call) Return(_a0 uint32, _a1 error) *MockStore_LoadDRSVersion_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockStore_LoadDRSVersion_Call) RunAndReturn(run func(uint64) (uint32, error)) *MockStore_LoadDRSVersion_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockStore) LoadIndexerBaseHeight() (uint64, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for LoadIndexerBaseHeight")
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

type MockStore_LoadIndexerBaseHeight_Call struct {
	*mock.Call
}

func (_e *MockStore_Expecter) LoadIndexerBaseHeight() *MockStore_LoadIndexerBaseHeight_Call {
	return &MockStore_LoadIndexerBaseHeight_Call{Call: _e.mock.On("LoadIndexerBaseHeight")}
}

func (_c *MockStore_LoadIndexerBaseHeight_Call) Run(run func()) *MockStore_LoadIndexerBaseHeight_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockStore_LoadIndexerBaseHeight_Call) Return(_a0 uint64, _a1 error) *MockStore_LoadIndexerBaseHeight_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockStore_LoadIndexerBaseHeight_Call) RunAndReturn(run func() (uint64, error)) *MockStore_LoadIndexerBaseHeight_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockStore) LoadLastBlockSequencerSet() (types.Sequencers, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for LoadLastBlockSequencerSet")
	}

	var r0 types.Sequencers
	var r1 error
	if rf, ok := ret.Get(0).(func() (types.Sequencers, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() types.Sequencers); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(types.Sequencers)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type MockStore_LoadLastBlockSequencerSet_Call struct {
	*mock.Call
}

func (_e *MockStore_Expecter) LoadLastBlockSequencerSet() *MockStore_LoadLastBlockSequencerSet_Call {
	return &MockStore_LoadLastBlockSequencerSet_Call{Call: _e.mock.On("LoadLastBlockSequencerSet")}
}

func (_c *MockStore_LoadLastBlockSequencerSet_Call) Run(run func()) *MockStore_LoadLastBlockSequencerSet_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockStore_LoadLastBlockSequencerSet_Call) Return(_a0 types.Sequencers, _a1 error) *MockStore_LoadLastBlockSequencerSet_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockStore_LoadLastBlockSequencerSet_Call) RunAndReturn(run func() (types.Sequencers, error)) *MockStore_LoadLastBlockSequencerSet_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockStore) LoadProposer(height uint64) (types.Sequencer, error) {
	ret := _m.Called(height)

	if len(ret) == 0 {
		panic("no return value specified for LoadProposer")
	}

	var r0 types.Sequencer
	var r1 error
	if rf, ok := ret.Get(0).(func(uint64) (types.Sequencer, error)); ok {
		return rf(height)
	}
	if rf, ok := ret.Get(0).(func(uint64) types.Sequencer); ok {
		r0 = rf(height)
	} else {
		r0 = ret.Get(0).(types.Sequencer)
	}

	if rf, ok := ret.Get(1).(func(uint64) error); ok {
		r1 = rf(height)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type MockStore_LoadProposer_Call struct {
	*mock.Call
}

func (_e *MockStore_Expecter) LoadProposer(height interface{}) *MockStore_LoadProposer_Call {
	return &MockStore_LoadProposer_Call{Call: _e.mock.On("LoadProposer", height)}
}

func (_c *MockStore_LoadProposer_Call) Run(run func(height uint64)) *MockStore_LoadProposer_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(uint64))
	})
	return _c
}

func (_c *MockStore_LoadProposer_Call) Return(_a0 types.Sequencer, _a1 error) *MockStore_LoadProposer_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockStore_LoadProposer_Call) RunAndReturn(run func(uint64) (types.Sequencer, error)) *MockStore_LoadProposer_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockStore) LoadState() (*types.State, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for LoadState")
	}

	var r0 *types.State
	var r1 error
	if rf, ok := ret.Get(0).(func() (*types.State, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() *types.State); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.State)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type MockStore_LoadState_Call struct {
	*mock.Call
}

func (_e *MockStore_Expecter) LoadState() *MockStore_LoadState_Call {
	return &MockStore_LoadState_Call{Call: _e.mock.On("LoadState")}
}

func (_c *MockStore_LoadState_Call) Run(run func()) *MockStore_LoadState_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockStore_LoadState_Call) Return(_a0 *types.State, _a1 error) *MockStore_LoadState_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockStore_LoadState_Call) RunAndReturn(run func() (*types.State, error)) *MockStore_LoadState_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockStore) LoadValidationHeight() (uint64, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for LoadValidationHeight")
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

type MockStore_LoadValidationHeight_Call struct {
	*mock.Call
}

func (_e *MockStore_Expecter) LoadValidationHeight() *MockStore_LoadValidationHeight_Call {
	return &MockStore_LoadValidationHeight_Call{Call: _e.mock.On("LoadValidationHeight")}
}

func (_c *MockStore_LoadValidationHeight_Call) Run(run func()) *MockStore_LoadValidationHeight_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockStore_LoadValidationHeight_Call) Return(_a0 uint64, _a1 error) *MockStore_LoadValidationHeight_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockStore_LoadValidationHeight_Call) RunAndReturn(run func() (uint64, error)) *MockStore_LoadValidationHeight_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockStore) NewBatch() store.KVBatch {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for NewBatch")
	}

	var r0 store.KVBatch
	if rf, ok := ret.Get(0).(func() store.KVBatch); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(store.KVBatch)
		}
	}

	return r0
}

type MockStore_NewBatch_Call struct {
	*mock.Call
}

func (_e *MockStore_Expecter) NewBatch() *MockStore_NewBatch_Call {
	return &MockStore_NewBatch_Call{Call: _e.mock.On("NewBatch")}
}

func (_c *MockStore_NewBatch_Call) Run(run func()) *MockStore_NewBatch_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockStore_NewBatch_Call) Return(_a0 store.KVBatch) *MockStore_NewBatch_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockStore_NewBatch_Call) RunAndReturn(run func() store.KVBatch) *MockStore_NewBatch_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockStore) PruneStore(to uint64, logger types.Logger) (uint64, error) {
	ret := _m.Called(to, logger)

	if len(ret) == 0 {
		panic("no return value specified for PruneStore")
	}

	var r0 uint64
	var r1 error
	if rf, ok := ret.Get(0).(func(uint64, types.Logger) (uint64, error)); ok {
		return rf(to, logger)
	}
	if rf, ok := ret.Get(0).(func(uint64, types.Logger) uint64); ok {
		r0 = rf(to, logger)
	} else {
		r0 = ret.Get(0).(uint64)
	}

	if rf, ok := ret.Get(1).(func(uint64, types.Logger) error); ok {
		r1 = rf(to, logger)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type MockStore_PruneStore_Call struct {
	*mock.Call
}

func (_e *MockStore_Expecter) PruneStore(to interface{}, logger interface{}) *MockStore_PruneStore_Call {
	return &MockStore_PruneStore_Call{Call: _e.mock.On("PruneStore", to, logger)}
}

func (_c *MockStore_PruneStore_Call) Run(run func(to uint64, logger types.Logger)) *MockStore_PruneStore_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(uint64), args[1].(types.Logger))
	})
	return _c
}

func (_c *MockStore_PruneStore_Call) Return(_a0 uint64, _a1 error) *MockStore_PruneStore_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockStore_PruneStore_Call) RunAndReturn(run func(uint64, types.Logger) (uint64, error)) *MockStore_PruneStore_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockStore) RemoveBlockCid(height uint64) error {
	ret := _m.Called(height)

	if len(ret) == 0 {
		panic("no return value specified for RemoveBlockCid")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(uint64) error); ok {
		r0 = rf(height)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type MockStore_RemoveBlockCid_Call struct {
	*mock.Call
}

func (_e *MockStore_Expecter) RemoveBlockCid(height interface{}) *MockStore_RemoveBlockCid_Call {
	return &MockStore_RemoveBlockCid_Call{Call: _e.mock.On("RemoveBlockCid", height)}
}

func (_c *MockStore_RemoveBlockCid_Call) Run(run func(height uint64)) *MockStore_RemoveBlockCid_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(uint64))
	})
	return _c
}

func (_c *MockStore_RemoveBlockCid_Call) Return(_a0 error) *MockStore_RemoveBlockCid_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockStore_RemoveBlockCid_Call) RunAndReturn(run func(uint64) error) *MockStore_RemoveBlockCid_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockStore) SaveBaseHeight(height uint64) error {
	ret := _m.Called(height)

	if len(ret) == 0 {
		panic("no return value specified for SaveBaseHeight")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(uint64) error); ok {
		r0 = rf(height)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type MockStore_SaveBaseHeight_Call struct {
	*mock.Call
}

func (_e *MockStore_Expecter) SaveBaseHeight(height interface{}) *MockStore_SaveBaseHeight_Call {
	return &MockStore_SaveBaseHeight_Call{Call: _e.mock.On("SaveBaseHeight", height)}
}

func (_c *MockStore_SaveBaseHeight_Call) Run(run func(height uint64)) *MockStore_SaveBaseHeight_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(uint64))
	})
	return _c
}

func (_c *MockStore_SaveBaseHeight_Call) Return(_a0 error) *MockStore_SaveBaseHeight_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockStore_SaveBaseHeight_Call) RunAndReturn(run func(uint64) error) *MockStore_SaveBaseHeight_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockStore) SaveBlock(block *types.Block, commit *types.Commit, batch store.KVBatch) (store.KVBatch, error) {
	ret := _m.Called(block, commit, batch)

	if len(ret) == 0 {
		panic("no return value specified for SaveBlock")
	}

	var r0 store.KVBatch
	var r1 error
	if rf, ok := ret.Get(0).(func(*types.Block, *types.Commit, store.KVBatch) (store.KVBatch, error)); ok {
		return rf(block, commit, batch)
	}
	if rf, ok := ret.Get(0).(func(*types.Block, *types.Commit, store.KVBatch) store.KVBatch); ok {
		r0 = rf(block, commit, batch)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(store.KVBatch)
		}
	}

	if rf, ok := ret.Get(1).(func(*types.Block, *types.Commit, store.KVBatch) error); ok {
		r1 = rf(block, commit, batch)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type MockStore_SaveBlock_Call struct {
	*mock.Call
}

func (_e *MockStore_Expecter) SaveBlock(block interface{}, commit interface{}, batch interface{}) *MockStore_SaveBlock_Call {
	return &MockStore_SaveBlock_Call{Call: _e.mock.On("SaveBlock", block, commit, batch)}
}

func (_c *MockStore_SaveBlock_Call) Run(run func(block *types.Block, commit *types.Commit, batch store.KVBatch)) *MockStore_SaveBlock_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*types.Block), args[1].(*types.Commit), args[2].(store.KVBatch))
	})
	return _c
}

func (_c *MockStore_SaveBlock_Call) Return(_a0 store.KVBatch, _a1 error) *MockStore_SaveBlock_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockStore_SaveBlock_Call) RunAndReturn(run func(*types.Block, *types.Commit, store.KVBatch) (store.KVBatch, error)) *MockStore_SaveBlock_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockStore) SaveBlockCid(height uint64, _a1 cid.Cid, batch store.KVBatch) (store.KVBatch, error) {
	ret := _m.Called(height, _a1, batch)

	if len(ret) == 0 {
		panic("no return value specified for SaveBlockCid")
	}

	var r0 store.KVBatch
	var r1 error
	if rf, ok := ret.Get(0).(func(uint64, cid.Cid, store.KVBatch) (store.KVBatch, error)); ok {
		return rf(height, _a1, batch)
	}
	if rf, ok := ret.Get(0).(func(uint64, cid.Cid, store.KVBatch) store.KVBatch); ok {
		r0 = rf(height, _a1, batch)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(store.KVBatch)
		}
	}

	if rf, ok := ret.Get(1).(func(uint64, cid.Cid, store.KVBatch) error); ok {
		r1 = rf(height, _a1, batch)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type MockStore_SaveBlockCid_Call struct {
	*mock.Call
}

func (_e *MockStore_Expecter) SaveBlockCid(height interface{}, _a1 interface{}, batch interface{}) *MockStore_SaveBlockCid_Call {
	return &MockStore_SaveBlockCid_Call{Call: _e.mock.On("SaveBlockCid", height, _a1, batch)}
}

func (_c *MockStore_SaveBlockCid_Call) Run(run func(height uint64, _a1 cid.Cid, batch store.KVBatch)) *MockStore_SaveBlockCid_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(uint64), args[1].(cid.Cid), args[2].(store.KVBatch))
	})
	return _c
}

func (_c *MockStore_SaveBlockCid_Call) Return(_a0 store.KVBatch, _a1 error) *MockStore_SaveBlockCid_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockStore_SaveBlockCid_Call) RunAndReturn(run func(uint64, cid.Cid, store.KVBatch) (store.KVBatch, error)) *MockStore_SaveBlockCid_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockStore) SaveBlockResponses(height uint64, responses *state.ABCIResponses, batch store.KVBatch) (store.KVBatch, error) {
	ret := _m.Called(height, responses, batch)

	if len(ret) == 0 {
		panic("no return value specified for SaveBlockResponses")
	}

	var r0 store.KVBatch
	var r1 error
	if rf, ok := ret.Get(0).(func(uint64, *state.ABCIResponses, store.KVBatch) (store.KVBatch, error)); ok {
		return rf(height, responses, batch)
	}
	if rf, ok := ret.Get(0).(func(uint64, *state.ABCIResponses, store.KVBatch) store.KVBatch); ok {
		r0 = rf(height, responses, batch)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(store.KVBatch)
		}
	}

	if rf, ok := ret.Get(1).(func(uint64, *state.ABCIResponses, store.KVBatch) error); ok {
		r1 = rf(height, responses, batch)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type MockStore_SaveBlockResponses_Call struct {
	*mock.Call
}

func (_e *MockStore_Expecter) SaveBlockResponses(height interface{}, responses interface{}, batch interface{}) *MockStore_SaveBlockResponses_Call {
	return &MockStore_SaveBlockResponses_Call{Call: _e.mock.On("SaveBlockResponses", height, responses, batch)}
}

func (_c *MockStore_SaveBlockResponses_Call) Run(run func(height uint64, responses *state.ABCIResponses, batch store.KVBatch)) *MockStore_SaveBlockResponses_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(uint64), args[1].(*state.ABCIResponses), args[2].(store.KVBatch))
	})
	return _c
}

func (_c *MockStore_SaveBlockResponses_Call) Return(_a0 store.KVBatch, _a1 error) *MockStore_SaveBlockResponses_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockStore_SaveBlockResponses_Call) RunAndReturn(run func(uint64, *state.ABCIResponses, store.KVBatch) (store.KVBatch, error)) *MockStore_SaveBlockResponses_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockStore) SaveBlockSource(height uint64, source types.BlockSource, batch store.KVBatch) (store.KVBatch, error) {
	ret := _m.Called(height, source, batch)

	if len(ret) == 0 {
		panic("no return value specified for SaveBlockSource")
	}

	var r0 store.KVBatch
	var r1 error
	if rf, ok := ret.Get(0).(func(uint64, types.BlockSource, store.KVBatch) (store.KVBatch, error)); ok {
		return rf(height, source, batch)
	}
	if rf, ok := ret.Get(0).(func(uint64, types.BlockSource, store.KVBatch) store.KVBatch); ok {
		r0 = rf(height, source, batch)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(store.KVBatch)
		}
	}

	if rf, ok := ret.Get(1).(func(uint64, types.BlockSource, store.KVBatch) error); ok {
		r1 = rf(height, source, batch)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type MockStore_SaveBlockSource_Call struct {
	*mock.Call
}

func (_e *MockStore_Expecter) SaveBlockSource(height interface{}, source interface{}, batch interface{}) *MockStore_SaveBlockSource_Call {
	return &MockStore_SaveBlockSource_Call{Call: _e.mock.On("SaveBlockSource", height, source, batch)}
}

func (_c *MockStore_SaveBlockSource_Call) Run(run func(height uint64, source types.BlockSource, batch store.KVBatch)) *MockStore_SaveBlockSource_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(uint64), args[1].(types.BlockSource), args[2].(store.KVBatch))
	})
	return _c
}

func (_c *MockStore_SaveBlockSource_Call) Return(_a0 store.KVBatch, _a1 error) *MockStore_SaveBlockSource_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockStore_SaveBlockSource_Call) RunAndReturn(run func(uint64, types.BlockSource, store.KVBatch) (store.KVBatch, error)) *MockStore_SaveBlockSource_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockStore) SaveBlockSyncBaseHeight(height uint64) error {
	ret := _m.Called(height)

	if len(ret) == 0 {
		panic("no return value specified for SaveBlockSyncBaseHeight")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(uint64) error); ok {
		r0 = rf(height)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type MockStore_SaveBlockSyncBaseHeight_Call struct {
	*mock.Call
}

func (_e *MockStore_Expecter) SaveBlockSyncBaseHeight(height interface{}) *MockStore_SaveBlockSyncBaseHeight_Call {
	return &MockStore_SaveBlockSyncBaseHeight_Call{Call: _e.mock.On("SaveBlockSyncBaseHeight", height)}
}

func (_c *MockStore_SaveBlockSyncBaseHeight_Call) Run(run func(height uint64)) *MockStore_SaveBlockSyncBaseHeight_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(uint64))
	})
	return _c
}

func (_c *MockStore_SaveBlockSyncBaseHeight_Call) Return(_a0 error) *MockStore_SaveBlockSyncBaseHeight_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockStore_SaveBlockSyncBaseHeight_Call) RunAndReturn(run func(uint64) error) *MockStore_SaveBlockSyncBaseHeight_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockStore) SaveDRSVersion(height uint64, version uint32, batch store.KVBatch) (store.KVBatch, error) {
	ret := _m.Called(height, version, batch)

	if len(ret) == 0 {
		panic("no return value specified for SaveDRSVersion")
	}

	var r0 store.KVBatch
	var r1 error
	if rf, ok := ret.Get(0).(func(uint64, uint32, store.KVBatch) (store.KVBatch, error)); ok {
		return rf(height, version, batch)
	}
	if rf, ok := ret.Get(0).(func(uint64, uint32, store.KVBatch) store.KVBatch); ok {
		r0 = rf(height, version, batch)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(store.KVBatch)
		}
	}

	if rf, ok := ret.Get(1).(func(uint64, uint32, store.KVBatch) error); ok {
		r1 = rf(height, version, batch)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type MockStore_SaveDRSVersion_Call struct {
	*mock.Call
}

func (_e *MockStore_Expecter) SaveDRSVersion(height interface{}, version interface{}, batch interface{}) *MockStore_SaveDRSVersion_Call {
	return &MockStore_SaveDRSVersion_Call{Call: _e.mock.On("SaveDRSVersion", height, version, batch)}
}

func (_c *MockStore_SaveDRSVersion_Call) Run(run func(height uint64, version uint32, batch store.KVBatch)) *MockStore_SaveDRSVersion_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(uint64), args[1].(uint32), args[2].(store.KVBatch))
	})
	return _c
}

func (_c *MockStore_SaveDRSVersion_Call) Return(_a0 store.KVBatch, _a1 error) *MockStore_SaveDRSVersion_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockStore_SaveDRSVersion_Call) RunAndReturn(run func(uint64, uint32, store.KVBatch) (store.KVBatch, error)) *MockStore_SaveDRSVersion_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockStore) SaveIndexerBaseHeight(height uint64) error {
	ret := _m.Called(height)

	if len(ret) == 0 {
		panic("no return value specified for SaveIndexerBaseHeight")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(uint64) error); ok {
		r0 = rf(height)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type MockStore_SaveIndexerBaseHeight_Call struct {
	*mock.Call
}

func (_e *MockStore_Expecter) SaveIndexerBaseHeight(height interface{}) *MockStore_SaveIndexerBaseHeight_Call {
	return &MockStore_SaveIndexerBaseHeight_Call{Call: _e.mock.On("SaveIndexerBaseHeight", height)}
}

func (_c *MockStore_SaveIndexerBaseHeight_Call) Run(run func(height uint64)) *MockStore_SaveIndexerBaseHeight_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(uint64))
	})
	return _c
}

func (_c *MockStore_SaveIndexerBaseHeight_Call) Return(_a0 error) *MockStore_SaveIndexerBaseHeight_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockStore_SaveIndexerBaseHeight_Call) RunAndReturn(run func(uint64) error) *MockStore_SaveIndexerBaseHeight_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockStore) SaveLastBlockSequencerSet(sequencers types.Sequencers, batch store.KVBatch) (store.KVBatch, error) {
	ret := _m.Called(sequencers, batch)

	if len(ret) == 0 {
		panic("no return value specified for SaveLastBlockSequencerSet")
	}

	var r0 store.KVBatch
	var r1 error
	if rf, ok := ret.Get(0).(func(types.Sequencers, store.KVBatch) (store.KVBatch, error)); ok {
		return rf(sequencers, batch)
	}
	if rf, ok := ret.Get(0).(func(types.Sequencers, store.KVBatch) store.KVBatch); ok {
		r0 = rf(sequencers, batch)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(store.KVBatch)
		}
	}

	if rf, ok := ret.Get(1).(func(types.Sequencers, store.KVBatch) error); ok {
		r1 = rf(sequencers, batch)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type MockStore_SaveLastBlockSequencerSet_Call struct {
	*mock.Call
}

func (_e *MockStore_Expecter) SaveLastBlockSequencerSet(sequencers interface{}, batch interface{}) *MockStore_SaveLastBlockSequencerSet_Call {
	return &MockStore_SaveLastBlockSequencerSet_Call{Call: _e.mock.On("SaveLastBlockSequencerSet", sequencers, batch)}
}

func (_c *MockStore_SaveLastBlockSequencerSet_Call) Run(run func(sequencers types.Sequencers, batch store.KVBatch)) *MockStore_SaveLastBlockSequencerSet_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.Sequencers), args[1].(store.KVBatch))
	})
	return _c
}

func (_c *MockStore_SaveLastBlockSequencerSet_Call) Return(_a0 store.KVBatch, _a1 error) *MockStore_SaveLastBlockSequencerSet_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockStore_SaveLastBlockSequencerSet_Call) RunAndReturn(run func(types.Sequencers, store.KVBatch) (store.KVBatch, error)) *MockStore_SaveLastBlockSequencerSet_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockStore) SaveProposer(height uint64, proposer types.Sequencer, batch store.KVBatch) (store.KVBatch, error) {
	ret := _m.Called(height, proposer, batch)

	if len(ret) == 0 {
		panic("no return value specified for SaveProposer")
	}

	var r0 store.KVBatch
	var r1 error
	if rf, ok := ret.Get(0).(func(uint64, types.Sequencer, store.KVBatch) (store.KVBatch, error)); ok {
		return rf(height, proposer, batch)
	}
	if rf, ok := ret.Get(0).(func(uint64, types.Sequencer, store.KVBatch) store.KVBatch); ok {
		r0 = rf(height, proposer, batch)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(store.KVBatch)
		}
	}

	if rf, ok := ret.Get(1).(func(uint64, types.Sequencer, store.KVBatch) error); ok {
		r1 = rf(height, proposer, batch)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type MockStore_SaveProposer_Call struct {
	*mock.Call
}

func (_e *MockStore_Expecter) SaveProposer(height interface{}, proposer interface{}, batch interface{}) *MockStore_SaveProposer_Call {
	return &MockStore_SaveProposer_Call{Call: _e.mock.On("SaveProposer", height, proposer, batch)}
}

func (_c *MockStore_SaveProposer_Call) Run(run func(height uint64, proposer types.Sequencer, batch store.KVBatch)) *MockStore_SaveProposer_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(uint64), args[1].(types.Sequencer), args[2].(store.KVBatch))
	})
	return _c
}

func (_c *MockStore_SaveProposer_Call) Return(_a0 store.KVBatch, _a1 error) *MockStore_SaveProposer_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockStore_SaveProposer_Call) RunAndReturn(run func(uint64, types.Sequencer, store.KVBatch) (store.KVBatch, error)) *MockStore_SaveProposer_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockStore) SaveState(_a0 *types.State, batch store.KVBatch) (store.KVBatch, error) {
	ret := _m.Called(_a0, batch)

	if len(ret) == 0 {
		panic("no return value specified for SaveState")
	}

	var r0 store.KVBatch
	var r1 error
	if rf, ok := ret.Get(0).(func(*types.State, store.KVBatch) (store.KVBatch, error)); ok {
		return rf(_a0, batch)
	}
	if rf, ok := ret.Get(0).(func(*types.State, store.KVBatch) store.KVBatch); ok {
		r0 = rf(_a0, batch)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(store.KVBatch)
		}
	}

	if rf, ok := ret.Get(1).(func(*types.State, store.KVBatch) error); ok {
		r1 = rf(_a0, batch)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type MockStore_SaveState_Call struct {
	*mock.Call
}

func (_e *MockStore_Expecter) SaveState(_a0 interface{}, batch interface{}) *MockStore_SaveState_Call {
	return &MockStore_SaveState_Call{Call: _e.mock.On("SaveState", _a0, batch)}
}

func (_c *MockStore_SaveState_Call) Run(run func(_a0 *types.State, batch store.KVBatch)) *MockStore_SaveState_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*types.State), args[1].(store.KVBatch))
	})
	return _c
}

func (_c *MockStore_SaveState_Call) Return(_a0 store.KVBatch, _a1 error) *MockStore_SaveState_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockStore_SaveState_Call) RunAndReturn(run func(*types.State, store.KVBatch) (store.KVBatch, error)) *MockStore_SaveState_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockStore) SaveValidationHeight(height uint64, batch store.KVBatch) (store.KVBatch, error) {
	ret := _m.Called(height, batch)

	if len(ret) == 0 {
		panic("no return value specified for SaveValidationHeight")
	}

	var r0 store.KVBatch
	var r1 error
	if rf, ok := ret.Get(0).(func(uint64, store.KVBatch) (store.KVBatch, error)); ok {
		return rf(height, batch)
	}
	if rf, ok := ret.Get(0).(func(uint64, store.KVBatch) store.KVBatch); ok {
		r0 = rf(height, batch)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(store.KVBatch)
		}
	}

	if rf, ok := ret.Get(1).(func(uint64, store.KVBatch) error); ok {
		r1 = rf(height, batch)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type MockStore_SaveValidationHeight_Call struct {
	*mock.Call
}

func (_e *MockStore_Expecter) SaveValidationHeight(height interface{}, batch interface{}) *MockStore_SaveValidationHeight_Call {
	return &MockStore_SaveValidationHeight_Call{Call: _e.mock.On("SaveValidationHeight", height, batch)}
}

func (_c *MockStore_SaveValidationHeight_Call) Run(run func(height uint64, batch store.KVBatch)) *MockStore_SaveValidationHeight_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(uint64), args[1].(store.KVBatch))
	})
	return _c
}

func (_c *MockStore_SaveValidationHeight_Call) Return(_a0 store.KVBatch, _a1 error) *MockStore_SaveValidationHeight_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockStore_SaveValidationHeight_Call) RunAndReturn(run func(uint64, store.KVBatch) (store.KVBatch, error)) *MockStore_SaveValidationHeight_Call {
	_c.Call.Return(run)
	return _c
}

func NewMockStore(t interface {
	mock.TestingT
	Cleanup(func())
},
) *MockStore {
	mock := &MockStore{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
