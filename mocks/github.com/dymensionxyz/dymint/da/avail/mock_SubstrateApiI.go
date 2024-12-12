

package avail

import (
	author "github.com/centrifuge/go-substrate-rpc-client/v4/rpc/author"

	chain "github.com/centrifuge/go-substrate-rpc-client/v4/rpc/chain"

	mock "github.com/stretchr/testify/mock"

	state "github.com/centrifuge/go-substrate-rpc-client/v4/rpc/state"

	types "github.com/centrifuge/go-substrate-rpc-client/v4/types"
)


type MockSubstrateApiI struct {
	mock.Mock
}

type MockSubstrateApiI_Expecter struct {
	mock *mock.Mock
}

func (_m *MockSubstrateApiI) EXPECT() *MockSubstrateApiI_Expecter {
	return &MockSubstrateApiI_Expecter{mock: &_m.Mock}
}


func (_m *MockSubstrateApiI) GetBlock(blockHash types.Hash) (*types.SignedBlock, error) {
	ret := _m.Called(blockHash)

	if len(ret) == 0 {
		panic("no return value specified for GetBlock")
	}

	var r0 *types.SignedBlock
	var r1 error
	if rf, ok := ret.Get(0).(func(types.Hash) (*types.SignedBlock, error)); ok {
		return rf(blockHash)
	}
	if rf, ok := ret.Get(0).(func(types.Hash) *types.SignedBlock); ok {
		r0 = rf(blockHash)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.SignedBlock)
		}
	}

	if rf, ok := ret.Get(1).(func(types.Hash) error); ok {
		r1 = rf(blockHash)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_GetBlock_Call struct {
	*mock.Call
}



func (_e *MockSubstrateApiI_Expecter) GetBlock(blockHash interface{}) *MockSubstrateApiI_GetBlock_Call {
	return &MockSubstrateApiI_GetBlock_Call{Call: _e.mock.On("GetBlock", blockHash)}
}

func (_c *MockSubstrateApiI_GetBlock_Call) Run(run func(blockHash types.Hash)) *MockSubstrateApiI_GetBlock_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.Hash))
	})
	return _c
}

func (_c *MockSubstrateApiI_GetBlock_Call) Return(_a0 *types.SignedBlock, _a1 error) *MockSubstrateApiI_GetBlock_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSubstrateApiI_GetBlock_Call) RunAndReturn(run func(types.Hash) (*types.SignedBlock, error)) *MockSubstrateApiI_GetBlock_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockSubstrateApiI) GetBlockHash(blockNumber uint64) (types.Hash, error) {
	ret := _m.Called(blockNumber)

	if len(ret) == 0 {
		panic("no return value specified for GetBlockHash")
	}

	var r0 types.Hash
	var r1 error
	if rf, ok := ret.Get(0).(func(uint64) (types.Hash, error)); ok {
		return rf(blockNumber)
	}
	if rf, ok := ret.Get(0).(func(uint64) types.Hash); ok {
		r0 = rf(blockNumber)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(types.Hash)
		}
	}

	if rf, ok := ret.Get(1).(func(uint64) error); ok {
		r1 = rf(blockNumber)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_GetBlockHash_Call struct {
	*mock.Call
}



func (_e *MockSubstrateApiI_Expecter) GetBlockHash(blockNumber interface{}) *MockSubstrateApiI_GetBlockHash_Call {
	return &MockSubstrateApiI_GetBlockHash_Call{Call: _e.mock.On("GetBlockHash", blockNumber)}
}

func (_c *MockSubstrateApiI_GetBlockHash_Call) Run(run func(blockNumber uint64)) *MockSubstrateApiI_GetBlockHash_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(uint64))
	})
	return _c
}

func (_c *MockSubstrateApiI_GetBlockHash_Call) Return(_a0 types.Hash, _a1 error) *MockSubstrateApiI_GetBlockHash_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSubstrateApiI_GetBlockHash_Call) RunAndReturn(run func(uint64) (types.Hash, error)) *MockSubstrateApiI_GetBlockHash_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockSubstrateApiI) GetBlockHashLatest() (types.Hash, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetBlockHashLatest")
	}

	var r0 types.Hash
	var r1 error
	if rf, ok := ret.Get(0).(func() (types.Hash, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() types.Hash); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(types.Hash)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_GetBlockHashLatest_Call struct {
	*mock.Call
}


func (_e *MockSubstrateApiI_Expecter) GetBlockHashLatest() *MockSubstrateApiI_GetBlockHashLatest_Call {
	return &MockSubstrateApiI_GetBlockHashLatest_Call{Call: _e.mock.On("GetBlockHashLatest")}
}

func (_c *MockSubstrateApiI_GetBlockHashLatest_Call) Run(run func()) *MockSubstrateApiI_GetBlockHashLatest_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockSubstrateApiI_GetBlockHashLatest_Call) Return(_a0 types.Hash, _a1 error) *MockSubstrateApiI_GetBlockHashLatest_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSubstrateApiI_GetBlockHashLatest_Call) RunAndReturn(run func() (types.Hash, error)) *MockSubstrateApiI_GetBlockHashLatest_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockSubstrateApiI) GetBlockLatest() (*types.SignedBlock, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetBlockLatest")
	}

	var r0 *types.SignedBlock
	var r1 error
	if rf, ok := ret.Get(0).(func() (*types.SignedBlock, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() *types.SignedBlock); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.SignedBlock)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_GetBlockLatest_Call struct {
	*mock.Call
}


func (_e *MockSubstrateApiI_Expecter) GetBlockLatest() *MockSubstrateApiI_GetBlockLatest_Call {
	return &MockSubstrateApiI_GetBlockLatest_Call{Call: _e.mock.On("GetBlockLatest")}
}

func (_c *MockSubstrateApiI_GetBlockLatest_Call) Run(run func()) *MockSubstrateApiI_GetBlockLatest_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockSubstrateApiI_GetBlockLatest_Call) Return(_a0 *types.SignedBlock, _a1 error) *MockSubstrateApiI_GetBlockLatest_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSubstrateApiI_GetBlockLatest_Call) RunAndReturn(run func() (*types.SignedBlock, error)) *MockSubstrateApiI_GetBlockLatest_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockSubstrateApiI) GetChildKeys(childStorageKey types.StorageKey, prefix types.StorageKey, blockHash types.Hash) ([]types.StorageKey, error) {
	ret := _m.Called(childStorageKey, prefix, blockHash)

	if len(ret) == 0 {
		panic("no return value specified for GetChildKeys")
	}

	var r0 []types.StorageKey
	var r1 error
	if rf, ok := ret.Get(0).(func(types.StorageKey, types.StorageKey, types.Hash) ([]types.StorageKey, error)); ok {
		return rf(childStorageKey, prefix, blockHash)
	}
	if rf, ok := ret.Get(0).(func(types.StorageKey, types.StorageKey, types.Hash) []types.StorageKey); ok {
		r0 = rf(childStorageKey, prefix, blockHash)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]types.StorageKey)
		}
	}

	if rf, ok := ret.Get(1).(func(types.StorageKey, types.StorageKey, types.Hash) error); ok {
		r1 = rf(childStorageKey, prefix, blockHash)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_GetChildKeys_Call struct {
	*mock.Call
}





func (_e *MockSubstrateApiI_Expecter) GetChildKeys(childStorageKey interface{}, prefix interface{}, blockHash interface{}) *MockSubstrateApiI_GetChildKeys_Call {
	return &MockSubstrateApiI_GetChildKeys_Call{Call: _e.mock.On("GetChildKeys", childStorageKey, prefix, blockHash)}
}

func (_c *MockSubstrateApiI_GetChildKeys_Call) Run(run func(childStorageKey types.StorageKey, prefix types.StorageKey, blockHash types.Hash)) *MockSubstrateApiI_GetChildKeys_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.StorageKey), args[1].(types.StorageKey), args[2].(types.Hash))
	})
	return _c
}

func (_c *MockSubstrateApiI_GetChildKeys_Call) Return(_a0 []types.StorageKey, _a1 error) *MockSubstrateApiI_GetChildKeys_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSubstrateApiI_GetChildKeys_Call) RunAndReturn(run func(types.StorageKey, types.StorageKey, types.Hash) ([]types.StorageKey, error)) *MockSubstrateApiI_GetChildKeys_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockSubstrateApiI) GetChildKeysLatest(childStorageKey types.StorageKey, prefix types.StorageKey) ([]types.StorageKey, error) {
	ret := _m.Called(childStorageKey, prefix)

	if len(ret) == 0 {
		panic("no return value specified for GetChildKeysLatest")
	}

	var r0 []types.StorageKey
	var r1 error
	if rf, ok := ret.Get(0).(func(types.StorageKey, types.StorageKey) ([]types.StorageKey, error)); ok {
		return rf(childStorageKey, prefix)
	}
	if rf, ok := ret.Get(0).(func(types.StorageKey, types.StorageKey) []types.StorageKey); ok {
		r0 = rf(childStorageKey, prefix)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]types.StorageKey)
		}
	}

	if rf, ok := ret.Get(1).(func(types.StorageKey, types.StorageKey) error); ok {
		r1 = rf(childStorageKey, prefix)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_GetChildKeysLatest_Call struct {
	*mock.Call
}




func (_e *MockSubstrateApiI_Expecter) GetChildKeysLatest(childStorageKey interface{}, prefix interface{}) *MockSubstrateApiI_GetChildKeysLatest_Call {
	return &MockSubstrateApiI_GetChildKeysLatest_Call{Call: _e.mock.On("GetChildKeysLatest", childStorageKey, prefix)}
}

func (_c *MockSubstrateApiI_GetChildKeysLatest_Call) Run(run func(childStorageKey types.StorageKey, prefix types.StorageKey)) *MockSubstrateApiI_GetChildKeysLatest_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.StorageKey), args[1].(types.StorageKey))
	})
	return _c
}

func (_c *MockSubstrateApiI_GetChildKeysLatest_Call) Return(_a0 []types.StorageKey, _a1 error) *MockSubstrateApiI_GetChildKeysLatest_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSubstrateApiI_GetChildKeysLatest_Call) RunAndReturn(run func(types.StorageKey, types.StorageKey) ([]types.StorageKey, error)) *MockSubstrateApiI_GetChildKeysLatest_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockSubstrateApiI) GetChildStorage(childStorageKey types.StorageKey, key types.StorageKey, target interface{}, blockHash types.Hash) (bool, error) {
	ret := _m.Called(childStorageKey, key, target, blockHash)

	if len(ret) == 0 {
		panic("no return value specified for GetChildStorage")
	}

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(types.StorageKey, types.StorageKey, interface{}, types.Hash) (bool, error)); ok {
		return rf(childStorageKey, key, target, blockHash)
	}
	if rf, ok := ret.Get(0).(func(types.StorageKey, types.StorageKey, interface{}, types.Hash) bool); ok {
		r0 = rf(childStorageKey, key, target, blockHash)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(types.StorageKey, types.StorageKey, interface{}, types.Hash) error); ok {
		r1 = rf(childStorageKey, key, target, blockHash)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_GetChildStorage_Call struct {
	*mock.Call
}






func (_e *MockSubstrateApiI_Expecter) GetChildStorage(childStorageKey interface{}, key interface{}, target interface{}, blockHash interface{}) *MockSubstrateApiI_GetChildStorage_Call {
	return &MockSubstrateApiI_GetChildStorage_Call{Call: _e.mock.On("GetChildStorage", childStorageKey, key, target, blockHash)}
}

func (_c *MockSubstrateApiI_GetChildStorage_Call) Run(run func(childStorageKey types.StorageKey, key types.StorageKey, target interface{}, blockHash types.Hash)) *MockSubstrateApiI_GetChildStorage_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.StorageKey), args[1].(types.StorageKey), args[2].(interface{}), args[3].(types.Hash))
	})
	return _c
}

func (_c *MockSubstrateApiI_GetChildStorage_Call) Return(ok bool, err error) *MockSubstrateApiI_GetChildStorage_Call {
	_c.Call.Return(ok, err)
	return _c
}

func (_c *MockSubstrateApiI_GetChildStorage_Call) RunAndReturn(run func(types.StorageKey, types.StorageKey, interface{}, types.Hash) (bool, error)) *MockSubstrateApiI_GetChildStorage_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockSubstrateApiI) GetChildStorageHash(childStorageKey types.StorageKey, key types.StorageKey, blockHash types.Hash) (types.Hash, error) {
	ret := _m.Called(childStorageKey, key, blockHash)

	if len(ret) == 0 {
		panic("no return value specified for GetChildStorageHash")
	}

	var r0 types.Hash
	var r1 error
	if rf, ok := ret.Get(0).(func(types.StorageKey, types.StorageKey, types.Hash) (types.Hash, error)); ok {
		return rf(childStorageKey, key, blockHash)
	}
	if rf, ok := ret.Get(0).(func(types.StorageKey, types.StorageKey, types.Hash) types.Hash); ok {
		r0 = rf(childStorageKey, key, blockHash)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(types.Hash)
		}
	}

	if rf, ok := ret.Get(1).(func(types.StorageKey, types.StorageKey, types.Hash) error); ok {
		r1 = rf(childStorageKey, key, blockHash)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_GetChildStorageHash_Call struct {
	*mock.Call
}





func (_e *MockSubstrateApiI_Expecter) GetChildStorageHash(childStorageKey interface{}, key interface{}, blockHash interface{}) *MockSubstrateApiI_GetChildStorageHash_Call {
	return &MockSubstrateApiI_GetChildStorageHash_Call{Call: _e.mock.On("GetChildStorageHash", childStorageKey, key, blockHash)}
}

func (_c *MockSubstrateApiI_GetChildStorageHash_Call) Run(run func(childStorageKey types.StorageKey, key types.StorageKey, blockHash types.Hash)) *MockSubstrateApiI_GetChildStorageHash_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.StorageKey), args[1].(types.StorageKey), args[2].(types.Hash))
	})
	return _c
}

func (_c *MockSubstrateApiI_GetChildStorageHash_Call) Return(_a0 types.Hash, _a1 error) *MockSubstrateApiI_GetChildStorageHash_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSubstrateApiI_GetChildStorageHash_Call) RunAndReturn(run func(types.StorageKey, types.StorageKey, types.Hash) (types.Hash, error)) *MockSubstrateApiI_GetChildStorageHash_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockSubstrateApiI) GetChildStorageHashLatest(childStorageKey types.StorageKey, key types.StorageKey) (types.Hash, error) {
	ret := _m.Called(childStorageKey, key)

	if len(ret) == 0 {
		panic("no return value specified for GetChildStorageHashLatest")
	}

	var r0 types.Hash
	var r1 error
	if rf, ok := ret.Get(0).(func(types.StorageKey, types.StorageKey) (types.Hash, error)); ok {
		return rf(childStorageKey, key)
	}
	if rf, ok := ret.Get(0).(func(types.StorageKey, types.StorageKey) types.Hash); ok {
		r0 = rf(childStorageKey, key)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(types.Hash)
		}
	}

	if rf, ok := ret.Get(1).(func(types.StorageKey, types.StorageKey) error); ok {
		r1 = rf(childStorageKey, key)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_GetChildStorageHashLatest_Call struct {
	*mock.Call
}




func (_e *MockSubstrateApiI_Expecter) GetChildStorageHashLatest(childStorageKey interface{}, key interface{}) *MockSubstrateApiI_GetChildStorageHashLatest_Call {
	return &MockSubstrateApiI_GetChildStorageHashLatest_Call{Call: _e.mock.On("GetChildStorageHashLatest", childStorageKey, key)}
}

func (_c *MockSubstrateApiI_GetChildStorageHashLatest_Call) Run(run func(childStorageKey types.StorageKey, key types.StorageKey)) *MockSubstrateApiI_GetChildStorageHashLatest_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.StorageKey), args[1].(types.StorageKey))
	})
	return _c
}

func (_c *MockSubstrateApiI_GetChildStorageHashLatest_Call) Return(_a0 types.Hash, _a1 error) *MockSubstrateApiI_GetChildStorageHashLatest_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSubstrateApiI_GetChildStorageHashLatest_Call) RunAndReturn(run func(types.StorageKey, types.StorageKey) (types.Hash, error)) *MockSubstrateApiI_GetChildStorageHashLatest_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockSubstrateApiI) GetChildStorageLatest(childStorageKey types.StorageKey, key types.StorageKey, target interface{}) (bool, error) {
	ret := _m.Called(childStorageKey, key, target)

	if len(ret) == 0 {
		panic("no return value specified for GetChildStorageLatest")
	}

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(types.StorageKey, types.StorageKey, interface{}) (bool, error)); ok {
		return rf(childStorageKey, key, target)
	}
	if rf, ok := ret.Get(0).(func(types.StorageKey, types.StorageKey, interface{}) bool); ok {
		r0 = rf(childStorageKey, key, target)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(types.StorageKey, types.StorageKey, interface{}) error); ok {
		r1 = rf(childStorageKey, key, target)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_GetChildStorageLatest_Call struct {
	*mock.Call
}





func (_e *MockSubstrateApiI_Expecter) GetChildStorageLatest(childStorageKey interface{}, key interface{}, target interface{}) *MockSubstrateApiI_GetChildStorageLatest_Call {
	return &MockSubstrateApiI_GetChildStorageLatest_Call{Call: _e.mock.On("GetChildStorageLatest", childStorageKey, key, target)}
}

func (_c *MockSubstrateApiI_GetChildStorageLatest_Call) Run(run func(childStorageKey types.StorageKey, key types.StorageKey, target interface{})) *MockSubstrateApiI_GetChildStorageLatest_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.StorageKey), args[1].(types.StorageKey), args[2].(interface{}))
	})
	return _c
}

func (_c *MockSubstrateApiI_GetChildStorageLatest_Call) Return(ok bool, err error) *MockSubstrateApiI_GetChildStorageLatest_Call {
	_c.Call.Return(ok, err)
	return _c
}

func (_c *MockSubstrateApiI_GetChildStorageLatest_Call) RunAndReturn(run func(types.StorageKey, types.StorageKey, interface{}) (bool, error)) *MockSubstrateApiI_GetChildStorageLatest_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockSubstrateApiI) GetChildStorageRaw(childStorageKey types.StorageKey, key types.StorageKey, blockHash types.Hash) (*types.StorageDataRaw, error) {
	ret := _m.Called(childStorageKey, key, blockHash)

	if len(ret) == 0 {
		panic("no return value specified for GetChildStorageRaw")
	}

	var r0 *types.StorageDataRaw
	var r1 error
	if rf, ok := ret.Get(0).(func(types.StorageKey, types.StorageKey, types.Hash) (*types.StorageDataRaw, error)); ok {
		return rf(childStorageKey, key, blockHash)
	}
	if rf, ok := ret.Get(0).(func(types.StorageKey, types.StorageKey, types.Hash) *types.StorageDataRaw); ok {
		r0 = rf(childStorageKey, key, blockHash)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.StorageDataRaw)
		}
	}

	if rf, ok := ret.Get(1).(func(types.StorageKey, types.StorageKey, types.Hash) error); ok {
		r1 = rf(childStorageKey, key, blockHash)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_GetChildStorageRaw_Call struct {
	*mock.Call
}





func (_e *MockSubstrateApiI_Expecter) GetChildStorageRaw(childStorageKey interface{}, key interface{}, blockHash interface{}) *MockSubstrateApiI_GetChildStorageRaw_Call {
	return &MockSubstrateApiI_GetChildStorageRaw_Call{Call: _e.mock.On("GetChildStorageRaw", childStorageKey, key, blockHash)}
}

func (_c *MockSubstrateApiI_GetChildStorageRaw_Call) Run(run func(childStorageKey types.StorageKey, key types.StorageKey, blockHash types.Hash)) *MockSubstrateApiI_GetChildStorageRaw_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.StorageKey), args[1].(types.StorageKey), args[2].(types.Hash))
	})
	return _c
}

func (_c *MockSubstrateApiI_GetChildStorageRaw_Call) Return(_a0 *types.StorageDataRaw, _a1 error) *MockSubstrateApiI_GetChildStorageRaw_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSubstrateApiI_GetChildStorageRaw_Call) RunAndReturn(run func(types.StorageKey, types.StorageKey, types.Hash) (*types.StorageDataRaw, error)) *MockSubstrateApiI_GetChildStorageRaw_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockSubstrateApiI) GetChildStorageRawLatest(childStorageKey types.StorageKey, key types.StorageKey) (*types.StorageDataRaw, error) {
	ret := _m.Called(childStorageKey, key)

	if len(ret) == 0 {
		panic("no return value specified for GetChildStorageRawLatest")
	}

	var r0 *types.StorageDataRaw
	var r1 error
	if rf, ok := ret.Get(0).(func(types.StorageKey, types.StorageKey) (*types.StorageDataRaw, error)); ok {
		return rf(childStorageKey, key)
	}
	if rf, ok := ret.Get(0).(func(types.StorageKey, types.StorageKey) *types.StorageDataRaw); ok {
		r0 = rf(childStorageKey, key)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.StorageDataRaw)
		}
	}

	if rf, ok := ret.Get(1).(func(types.StorageKey, types.StorageKey) error); ok {
		r1 = rf(childStorageKey, key)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_GetChildStorageRawLatest_Call struct {
	*mock.Call
}




func (_e *MockSubstrateApiI_Expecter) GetChildStorageRawLatest(childStorageKey interface{}, key interface{}) *MockSubstrateApiI_GetChildStorageRawLatest_Call {
	return &MockSubstrateApiI_GetChildStorageRawLatest_Call{Call: _e.mock.On("GetChildStorageRawLatest", childStorageKey, key)}
}

func (_c *MockSubstrateApiI_GetChildStorageRawLatest_Call) Run(run func(childStorageKey types.StorageKey, key types.StorageKey)) *MockSubstrateApiI_GetChildStorageRawLatest_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.StorageKey), args[1].(types.StorageKey))
	})
	return _c
}

func (_c *MockSubstrateApiI_GetChildStorageRawLatest_Call) Return(_a0 *types.StorageDataRaw, _a1 error) *MockSubstrateApiI_GetChildStorageRawLatest_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSubstrateApiI_GetChildStorageRawLatest_Call) RunAndReturn(run func(types.StorageKey, types.StorageKey) (*types.StorageDataRaw, error)) *MockSubstrateApiI_GetChildStorageRawLatest_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockSubstrateApiI) GetChildStorageSize(childStorageKey types.StorageKey, key types.StorageKey, blockHash types.Hash) (types.U64, error) {
	ret := _m.Called(childStorageKey, key, blockHash)

	if len(ret) == 0 {
		panic("no return value specified for GetChildStorageSize")
	}

	var r0 types.U64
	var r1 error
	if rf, ok := ret.Get(0).(func(types.StorageKey, types.StorageKey, types.Hash) (types.U64, error)); ok {
		return rf(childStorageKey, key, blockHash)
	}
	if rf, ok := ret.Get(0).(func(types.StorageKey, types.StorageKey, types.Hash) types.U64); ok {
		r0 = rf(childStorageKey, key, blockHash)
	} else {
		r0 = ret.Get(0).(types.U64)
	}

	if rf, ok := ret.Get(1).(func(types.StorageKey, types.StorageKey, types.Hash) error); ok {
		r1 = rf(childStorageKey, key, blockHash)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_GetChildStorageSize_Call struct {
	*mock.Call
}





func (_e *MockSubstrateApiI_Expecter) GetChildStorageSize(childStorageKey interface{}, key interface{}, blockHash interface{}) *MockSubstrateApiI_GetChildStorageSize_Call {
	return &MockSubstrateApiI_GetChildStorageSize_Call{Call: _e.mock.On("GetChildStorageSize", childStorageKey, key, blockHash)}
}

func (_c *MockSubstrateApiI_GetChildStorageSize_Call) Run(run func(childStorageKey types.StorageKey, key types.StorageKey, blockHash types.Hash)) *MockSubstrateApiI_GetChildStorageSize_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.StorageKey), args[1].(types.StorageKey), args[2].(types.Hash))
	})
	return _c
}

func (_c *MockSubstrateApiI_GetChildStorageSize_Call) Return(_a0 types.U64, _a1 error) *MockSubstrateApiI_GetChildStorageSize_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSubstrateApiI_GetChildStorageSize_Call) RunAndReturn(run func(types.StorageKey, types.StorageKey, types.Hash) (types.U64, error)) *MockSubstrateApiI_GetChildStorageSize_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockSubstrateApiI) GetChildStorageSizeLatest(childStorageKey types.StorageKey, key types.StorageKey) (types.U64, error) {
	ret := _m.Called(childStorageKey, key)

	if len(ret) == 0 {
		panic("no return value specified for GetChildStorageSizeLatest")
	}

	var r0 types.U64
	var r1 error
	if rf, ok := ret.Get(0).(func(types.StorageKey, types.StorageKey) (types.U64, error)); ok {
		return rf(childStorageKey, key)
	}
	if rf, ok := ret.Get(0).(func(types.StorageKey, types.StorageKey) types.U64); ok {
		r0 = rf(childStorageKey, key)
	} else {
		r0 = ret.Get(0).(types.U64)
	}

	if rf, ok := ret.Get(1).(func(types.StorageKey, types.StorageKey) error); ok {
		r1 = rf(childStorageKey, key)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_GetChildStorageSizeLatest_Call struct {
	*mock.Call
}




func (_e *MockSubstrateApiI_Expecter) GetChildStorageSizeLatest(childStorageKey interface{}, key interface{}) *MockSubstrateApiI_GetChildStorageSizeLatest_Call {
	return &MockSubstrateApiI_GetChildStorageSizeLatest_Call{Call: _e.mock.On("GetChildStorageSizeLatest", childStorageKey, key)}
}

func (_c *MockSubstrateApiI_GetChildStorageSizeLatest_Call) Run(run func(childStorageKey types.StorageKey, key types.StorageKey)) *MockSubstrateApiI_GetChildStorageSizeLatest_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.StorageKey), args[1].(types.StorageKey))
	})
	return _c
}

func (_c *MockSubstrateApiI_GetChildStorageSizeLatest_Call) Return(_a0 types.U64, _a1 error) *MockSubstrateApiI_GetChildStorageSizeLatest_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSubstrateApiI_GetChildStorageSizeLatest_Call) RunAndReturn(run func(types.StorageKey, types.StorageKey) (types.U64, error)) *MockSubstrateApiI_GetChildStorageSizeLatest_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockSubstrateApiI) GetFinalizedHead() (types.Hash, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetFinalizedHead")
	}

	var r0 types.Hash
	var r1 error
	if rf, ok := ret.Get(0).(func() (types.Hash, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() types.Hash); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(types.Hash)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_GetFinalizedHead_Call struct {
	*mock.Call
}


func (_e *MockSubstrateApiI_Expecter) GetFinalizedHead() *MockSubstrateApiI_GetFinalizedHead_Call {
	return &MockSubstrateApiI_GetFinalizedHead_Call{Call: _e.mock.On("GetFinalizedHead")}
}

func (_c *MockSubstrateApiI_GetFinalizedHead_Call) Run(run func()) *MockSubstrateApiI_GetFinalizedHead_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockSubstrateApiI_GetFinalizedHead_Call) Return(_a0 types.Hash, _a1 error) *MockSubstrateApiI_GetFinalizedHead_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSubstrateApiI_GetFinalizedHead_Call) RunAndReturn(run func() (types.Hash, error)) *MockSubstrateApiI_GetFinalizedHead_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockSubstrateApiI) GetHeader(blockHash types.Hash) (*types.Header, error) {
	ret := _m.Called(blockHash)

	if len(ret) == 0 {
		panic("no return value specified for GetHeader")
	}

	var r0 *types.Header
	var r1 error
	if rf, ok := ret.Get(0).(func(types.Hash) (*types.Header, error)); ok {
		return rf(blockHash)
	}
	if rf, ok := ret.Get(0).(func(types.Hash) *types.Header); ok {
		r0 = rf(blockHash)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Header)
		}
	}

	if rf, ok := ret.Get(1).(func(types.Hash) error); ok {
		r1 = rf(blockHash)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_GetHeader_Call struct {
	*mock.Call
}



func (_e *MockSubstrateApiI_Expecter) GetHeader(blockHash interface{}) *MockSubstrateApiI_GetHeader_Call {
	return &MockSubstrateApiI_GetHeader_Call{Call: _e.mock.On("GetHeader", blockHash)}
}

func (_c *MockSubstrateApiI_GetHeader_Call) Run(run func(blockHash types.Hash)) *MockSubstrateApiI_GetHeader_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.Hash))
	})
	return _c
}

func (_c *MockSubstrateApiI_GetHeader_Call) Return(_a0 *types.Header, _a1 error) *MockSubstrateApiI_GetHeader_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSubstrateApiI_GetHeader_Call) RunAndReturn(run func(types.Hash) (*types.Header, error)) *MockSubstrateApiI_GetHeader_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockSubstrateApiI) GetHeaderLatest() (*types.Header, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetHeaderLatest")
	}

	var r0 *types.Header
	var r1 error
	if rf, ok := ret.Get(0).(func() (*types.Header, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() *types.Header); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Header)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_GetHeaderLatest_Call struct {
	*mock.Call
}


func (_e *MockSubstrateApiI_Expecter) GetHeaderLatest() *MockSubstrateApiI_GetHeaderLatest_Call {
	return &MockSubstrateApiI_GetHeaderLatest_Call{Call: _e.mock.On("GetHeaderLatest")}
}

func (_c *MockSubstrateApiI_GetHeaderLatest_Call) Run(run func()) *MockSubstrateApiI_GetHeaderLatest_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockSubstrateApiI_GetHeaderLatest_Call) Return(_a0 *types.Header, _a1 error) *MockSubstrateApiI_GetHeaderLatest_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSubstrateApiI_GetHeaderLatest_Call) RunAndReturn(run func() (*types.Header, error)) *MockSubstrateApiI_GetHeaderLatest_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockSubstrateApiI) GetKeys(prefix types.StorageKey, blockHash types.Hash) ([]types.StorageKey, error) {
	ret := _m.Called(prefix, blockHash)

	if len(ret) == 0 {
		panic("no return value specified for GetKeys")
	}

	var r0 []types.StorageKey
	var r1 error
	if rf, ok := ret.Get(0).(func(types.StorageKey, types.Hash) ([]types.StorageKey, error)); ok {
		return rf(prefix, blockHash)
	}
	if rf, ok := ret.Get(0).(func(types.StorageKey, types.Hash) []types.StorageKey); ok {
		r0 = rf(prefix, blockHash)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]types.StorageKey)
		}
	}

	if rf, ok := ret.Get(1).(func(types.StorageKey, types.Hash) error); ok {
		r1 = rf(prefix, blockHash)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_GetKeys_Call struct {
	*mock.Call
}




func (_e *MockSubstrateApiI_Expecter) GetKeys(prefix interface{}, blockHash interface{}) *MockSubstrateApiI_GetKeys_Call {
	return &MockSubstrateApiI_GetKeys_Call{Call: _e.mock.On("GetKeys", prefix, blockHash)}
}

func (_c *MockSubstrateApiI_GetKeys_Call) Run(run func(prefix types.StorageKey, blockHash types.Hash)) *MockSubstrateApiI_GetKeys_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.StorageKey), args[1].(types.Hash))
	})
	return _c
}

func (_c *MockSubstrateApiI_GetKeys_Call) Return(_a0 []types.StorageKey, _a1 error) *MockSubstrateApiI_GetKeys_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSubstrateApiI_GetKeys_Call) RunAndReturn(run func(types.StorageKey, types.Hash) ([]types.StorageKey, error)) *MockSubstrateApiI_GetKeys_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockSubstrateApiI) GetKeysLatest(prefix types.StorageKey) ([]types.StorageKey, error) {
	ret := _m.Called(prefix)

	if len(ret) == 0 {
		panic("no return value specified for GetKeysLatest")
	}

	var r0 []types.StorageKey
	var r1 error
	if rf, ok := ret.Get(0).(func(types.StorageKey) ([]types.StorageKey, error)); ok {
		return rf(prefix)
	}
	if rf, ok := ret.Get(0).(func(types.StorageKey) []types.StorageKey); ok {
		r0 = rf(prefix)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]types.StorageKey)
		}
	}

	if rf, ok := ret.Get(1).(func(types.StorageKey) error); ok {
		r1 = rf(prefix)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_GetKeysLatest_Call struct {
	*mock.Call
}



func (_e *MockSubstrateApiI_Expecter) GetKeysLatest(prefix interface{}) *MockSubstrateApiI_GetKeysLatest_Call {
	return &MockSubstrateApiI_GetKeysLatest_Call{Call: _e.mock.On("GetKeysLatest", prefix)}
}

func (_c *MockSubstrateApiI_GetKeysLatest_Call) Run(run func(prefix types.StorageKey)) *MockSubstrateApiI_GetKeysLatest_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.StorageKey))
	})
	return _c
}

func (_c *MockSubstrateApiI_GetKeysLatest_Call) Return(_a0 []types.StorageKey, _a1 error) *MockSubstrateApiI_GetKeysLatest_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSubstrateApiI_GetKeysLatest_Call) RunAndReturn(run func(types.StorageKey) ([]types.StorageKey, error)) *MockSubstrateApiI_GetKeysLatest_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockSubstrateApiI) GetMetadata(blockHash types.Hash) (*types.Metadata, error) {
	ret := _m.Called(blockHash)

	if len(ret) == 0 {
		panic("no return value specified for GetMetadata")
	}

	var r0 *types.Metadata
	var r1 error
	if rf, ok := ret.Get(0).(func(types.Hash) (*types.Metadata, error)); ok {
		return rf(blockHash)
	}
	if rf, ok := ret.Get(0).(func(types.Hash) *types.Metadata); ok {
		r0 = rf(blockHash)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Metadata)
		}
	}

	if rf, ok := ret.Get(1).(func(types.Hash) error); ok {
		r1 = rf(blockHash)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_GetMetadata_Call struct {
	*mock.Call
}



func (_e *MockSubstrateApiI_Expecter) GetMetadata(blockHash interface{}) *MockSubstrateApiI_GetMetadata_Call {
	return &MockSubstrateApiI_GetMetadata_Call{Call: _e.mock.On("GetMetadata", blockHash)}
}

func (_c *MockSubstrateApiI_GetMetadata_Call) Run(run func(blockHash types.Hash)) *MockSubstrateApiI_GetMetadata_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.Hash))
	})
	return _c
}

func (_c *MockSubstrateApiI_GetMetadata_Call) Return(_a0 *types.Metadata, _a1 error) *MockSubstrateApiI_GetMetadata_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSubstrateApiI_GetMetadata_Call) RunAndReturn(run func(types.Hash) (*types.Metadata, error)) *MockSubstrateApiI_GetMetadata_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockSubstrateApiI) GetMetadataLatest() (*types.Metadata, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetMetadataLatest")
	}

	var r0 *types.Metadata
	var r1 error
	if rf, ok := ret.Get(0).(func() (*types.Metadata, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() *types.Metadata); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Metadata)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_GetMetadataLatest_Call struct {
	*mock.Call
}


func (_e *MockSubstrateApiI_Expecter) GetMetadataLatest() *MockSubstrateApiI_GetMetadataLatest_Call {
	return &MockSubstrateApiI_GetMetadataLatest_Call{Call: _e.mock.On("GetMetadataLatest")}
}

func (_c *MockSubstrateApiI_GetMetadataLatest_Call) Run(run func()) *MockSubstrateApiI_GetMetadataLatest_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockSubstrateApiI_GetMetadataLatest_Call) Return(_a0 *types.Metadata, _a1 error) *MockSubstrateApiI_GetMetadataLatest_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSubstrateApiI_GetMetadataLatest_Call) RunAndReturn(run func() (*types.Metadata, error)) *MockSubstrateApiI_GetMetadataLatest_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockSubstrateApiI) GetRuntimeVersion(blockHash types.Hash) (*types.RuntimeVersion, error) {
	ret := _m.Called(blockHash)

	if len(ret) == 0 {
		panic("no return value specified for GetRuntimeVersion")
	}

	var r0 *types.RuntimeVersion
	var r1 error
	if rf, ok := ret.Get(0).(func(types.Hash) (*types.RuntimeVersion, error)); ok {
		return rf(blockHash)
	}
	if rf, ok := ret.Get(0).(func(types.Hash) *types.RuntimeVersion); ok {
		r0 = rf(blockHash)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.RuntimeVersion)
		}
	}

	if rf, ok := ret.Get(1).(func(types.Hash) error); ok {
		r1 = rf(blockHash)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_GetRuntimeVersion_Call struct {
	*mock.Call
}



func (_e *MockSubstrateApiI_Expecter) GetRuntimeVersion(blockHash interface{}) *MockSubstrateApiI_GetRuntimeVersion_Call {
	return &MockSubstrateApiI_GetRuntimeVersion_Call{Call: _e.mock.On("GetRuntimeVersion", blockHash)}
}

func (_c *MockSubstrateApiI_GetRuntimeVersion_Call) Run(run func(blockHash types.Hash)) *MockSubstrateApiI_GetRuntimeVersion_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.Hash))
	})
	return _c
}

func (_c *MockSubstrateApiI_GetRuntimeVersion_Call) Return(_a0 *types.RuntimeVersion, _a1 error) *MockSubstrateApiI_GetRuntimeVersion_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSubstrateApiI_GetRuntimeVersion_Call) RunAndReturn(run func(types.Hash) (*types.RuntimeVersion, error)) *MockSubstrateApiI_GetRuntimeVersion_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockSubstrateApiI) GetRuntimeVersionLatest() (*types.RuntimeVersion, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetRuntimeVersionLatest")
	}

	var r0 *types.RuntimeVersion
	var r1 error
	if rf, ok := ret.Get(0).(func() (*types.RuntimeVersion, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() *types.RuntimeVersion); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.RuntimeVersion)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_GetRuntimeVersionLatest_Call struct {
	*mock.Call
}


func (_e *MockSubstrateApiI_Expecter) GetRuntimeVersionLatest() *MockSubstrateApiI_GetRuntimeVersionLatest_Call {
	return &MockSubstrateApiI_GetRuntimeVersionLatest_Call{Call: _e.mock.On("GetRuntimeVersionLatest")}
}

func (_c *MockSubstrateApiI_GetRuntimeVersionLatest_Call) Run(run func()) *MockSubstrateApiI_GetRuntimeVersionLatest_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockSubstrateApiI_GetRuntimeVersionLatest_Call) Return(_a0 *types.RuntimeVersion, _a1 error) *MockSubstrateApiI_GetRuntimeVersionLatest_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSubstrateApiI_GetRuntimeVersionLatest_Call) RunAndReturn(run func() (*types.RuntimeVersion, error)) *MockSubstrateApiI_GetRuntimeVersionLatest_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockSubstrateApiI) GetStorage(key types.StorageKey, target interface{}, blockHash types.Hash) (bool, error) {
	ret := _m.Called(key, target, blockHash)

	if len(ret) == 0 {
		panic("no return value specified for GetStorage")
	}

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(types.StorageKey, interface{}, types.Hash) (bool, error)); ok {
		return rf(key, target, blockHash)
	}
	if rf, ok := ret.Get(0).(func(types.StorageKey, interface{}, types.Hash) bool); ok {
		r0 = rf(key, target, blockHash)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(types.StorageKey, interface{}, types.Hash) error); ok {
		r1 = rf(key, target, blockHash)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_GetStorage_Call struct {
	*mock.Call
}





func (_e *MockSubstrateApiI_Expecter) GetStorage(key interface{}, target interface{}, blockHash interface{}) *MockSubstrateApiI_GetStorage_Call {
	return &MockSubstrateApiI_GetStorage_Call{Call: _e.mock.On("GetStorage", key, target, blockHash)}
}

func (_c *MockSubstrateApiI_GetStorage_Call) Run(run func(key types.StorageKey, target interface{}, blockHash types.Hash)) *MockSubstrateApiI_GetStorage_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.StorageKey), args[1].(interface{}), args[2].(types.Hash))
	})
	return _c
}

func (_c *MockSubstrateApiI_GetStorage_Call) Return(ok bool, err error) *MockSubstrateApiI_GetStorage_Call {
	_c.Call.Return(ok, err)
	return _c
}

func (_c *MockSubstrateApiI_GetStorage_Call) RunAndReturn(run func(types.StorageKey, interface{}, types.Hash) (bool, error)) *MockSubstrateApiI_GetStorage_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockSubstrateApiI) GetStorageHash(key types.StorageKey, blockHash types.Hash) (types.Hash, error) {
	ret := _m.Called(key, blockHash)

	if len(ret) == 0 {
		panic("no return value specified for GetStorageHash")
	}

	var r0 types.Hash
	var r1 error
	if rf, ok := ret.Get(0).(func(types.StorageKey, types.Hash) (types.Hash, error)); ok {
		return rf(key, blockHash)
	}
	if rf, ok := ret.Get(0).(func(types.StorageKey, types.Hash) types.Hash); ok {
		r0 = rf(key, blockHash)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(types.Hash)
		}
	}

	if rf, ok := ret.Get(1).(func(types.StorageKey, types.Hash) error); ok {
		r1 = rf(key, blockHash)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_GetStorageHash_Call struct {
	*mock.Call
}




func (_e *MockSubstrateApiI_Expecter) GetStorageHash(key interface{}, blockHash interface{}) *MockSubstrateApiI_GetStorageHash_Call {
	return &MockSubstrateApiI_GetStorageHash_Call{Call: _e.mock.On("GetStorageHash", key, blockHash)}
}

func (_c *MockSubstrateApiI_GetStorageHash_Call) Run(run func(key types.StorageKey, blockHash types.Hash)) *MockSubstrateApiI_GetStorageHash_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.StorageKey), args[1].(types.Hash))
	})
	return _c
}

func (_c *MockSubstrateApiI_GetStorageHash_Call) Return(_a0 types.Hash, _a1 error) *MockSubstrateApiI_GetStorageHash_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSubstrateApiI_GetStorageHash_Call) RunAndReturn(run func(types.StorageKey, types.Hash) (types.Hash, error)) *MockSubstrateApiI_GetStorageHash_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockSubstrateApiI) GetStorageHashLatest(key types.StorageKey) (types.Hash, error) {
	ret := _m.Called(key)

	if len(ret) == 0 {
		panic("no return value specified for GetStorageHashLatest")
	}

	var r0 types.Hash
	var r1 error
	if rf, ok := ret.Get(0).(func(types.StorageKey) (types.Hash, error)); ok {
		return rf(key)
	}
	if rf, ok := ret.Get(0).(func(types.StorageKey) types.Hash); ok {
		r0 = rf(key)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(types.Hash)
		}
	}

	if rf, ok := ret.Get(1).(func(types.StorageKey) error); ok {
		r1 = rf(key)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_GetStorageHashLatest_Call struct {
	*mock.Call
}



func (_e *MockSubstrateApiI_Expecter) GetStorageHashLatest(key interface{}) *MockSubstrateApiI_GetStorageHashLatest_Call {
	return &MockSubstrateApiI_GetStorageHashLatest_Call{Call: _e.mock.On("GetStorageHashLatest", key)}
}

func (_c *MockSubstrateApiI_GetStorageHashLatest_Call) Run(run func(key types.StorageKey)) *MockSubstrateApiI_GetStorageHashLatest_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.StorageKey))
	})
	return _c
}

func (_c *MockSubstrateApiI_GetStorageHashLatest_Call) Return(_a0 types.Hash, _a1 error) *MockSubstrateApiI_GetStorageHashLatest_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSubstrateApiI_GetStorageHashLatest_Call) RunAndReturn(run func(types.StorageKey) (types.Hash, error)) *MockSubstrateApiI_GetStorageHashLatest_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockSubstrateApiI) GetStorageLatest(key types.StorageKey, target interface{}) (bool, error) {
	ret := _m.Called(key, target)

	if len(ret) == 0 {
		panic("no return value specified for GetStorageLatest")
	}

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(types.StorageKey, interface{}) (bool, error)); ok {
		return rf(key, target)
	}
	if rf, ok := ret.Get(0).(func(types.StorageKey, interface{}) bool); ok {
		r0 = rf(key, target)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(types.StorageKey, interface{}) error); ok {
		r1 = rf(key, target)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_GetStorageLatest_Call struct {
	*mock.Call
}




func (_e *MockSubstrateApiI_Expecter) GetStorageLatest(key interface{}, target interface{}) *MockSubstrateApiI_GetStorageLatest_Call {
	return &MockSubstrateApiI_GetStorageLatest_Call{Call: _e.mock.On("GetStorageLatest", key, target)}
}

func (_c *MockSubstrateApiI_GetStorageLatest_Call) Run(run func(key types.StorageKey, target interface{})) *MockSubstrateApiI_GetStorageLatest_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.StorageKey), args[1].(interface{}))
	})
	return _c
}

func (_c *MockSubstrateApiI_GetStorageLatest_Call) Return(ok bool, err error) *MockSubstrateApiI_GetStorageLatest_Call {
	_c.Call.Return(ok, err)
	return _c
}

func (_c *MockSubstrateApiI_GetStorageLatest_Call) RunAndReturn(run func(types.StorageKey, interface{}) (bool, error)) *MockSubstrateApiI_GetStorageLatest_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockSubstrateApiI) GetStorageRaw(key types.StorageKey, blockHash types.Hash) (*types.StorageDataRaw, error) {
	ret := _m.Called(key, blockHash)

	if len(ret) == 0 {
		panic("no return value specified for GetStorageRaw")
	}

	var r0 *types.StorageDataRaw
	var r1 error
	if rf, ok := ret.Get(0).(func(types.StorageKey, types.Hash) (*types.StorageDataRaw, error)); ok {
		return rf(key, blockHash)
	}
	if rf, ok := ret.Get(0).(func(types.StorageKey, types.Hash) *types.StorageDataRaw); ok {
		r0 = rf(key, blockHash)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.StorageDataRaw)
		}
	}

	if rf, ok := ret.Get(1).(func(types.StorageKey, types.Hash) error); ok {
		r1 = rf(key, blockHash)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_GetStorageRaw_Call struct {
	*mock.Call
}




func (_e *MockSubstrateApiI_Expecter) GetStorageRaw(key interface{}, blockHash interface{}) *MockSubstrateApiI_GetStorageRaw_Call {
	return &MockSubstrateApiI_GetStorageRaw_Call{Call: _e.mock.On("GetStorageRaw", key, blockHash)}
}

func (_c *MockSubstrateApiI_GetStorageRaw_Call) Run(run func(key types.StorageKey, blockHash types.Hash)) *MockSubstrateApiI_GetStorageRaw_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.StorageKey), args[1].(types.Hash))
	})
	return _c
}

func (_c *MockSubstrateApiI_GetStorageRaw_Call) Return(_a0 *types.StorageDataRaw, _a1 error) *MockSubstrateApiI_GetStorageRaw_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSubstrateApiI_GetStorageRaw_Call) RunAndReturn(run func(types.StorageKey, types.Hash) (*types.StorageDataRaw, error)) *MockSubstrateApiI_GetStorageRaw_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockSubstrateApiI) GetStorageRawLatest(key types.StorageKey) (*types.StorageDataRaw, error) {
	ret := _m.Called(key)

	if len(ret) == 0 {
		panic("no return value specified for GetStorageRawLatest")
	}

	var r0 *types.StorageDataRaw
	var r1 error
	if rf, ok := ret.Get(0).(func(types.StorageKey) (*types.StorageDataRaw, error)); ok {
		return rf(key)
	}
	if rf, ok := ret.Get(0).(func(types.StorageKey) *types.StorageDataRaw); ok {
		r0 = rf(key)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.StorageDataRaw)
		}
	}

	if rf, ok := ret.Get(1).(func(types.StorageKey) error); ok {
		r1 = rf(key)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_GetStorageRawLatest_Call struct {
	*mock.Call
}



func (_e *MockSubstrateApiI_Expecter) GetStorageRawLatest(key interface{}) *MockSubstrateApiI_GetStorageRawLatest_Call {
	return &MockSubstrateApiI_GetStorageRawLatest_Call{Call: _e.mock.On("GetStorageRawLatest", key)}
}

func (_c *MockSubstrateApiI_GetStorageRawLatest_Call) Run(run func(key types.StorageKey)) *MockSubstrateApiI_GetStorageRawLatest_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.StorageKey))
	})
	return _c
}

func (_c *MockSubstrateApiI_GetStorageRawLatest_Call) Return(_a0 *types.StorageDataRaw, _a1 error) *MockSubstrateApiI_GetStorageRawLatest_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSubstrateApiI_GetStorageRawLatest_Call) RunAndReturn(run func(types.StorageKey) (*types.StorageDataRaw, error)) *MockSubstrateApiI_GetStorageRawLatest_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockSubstrateApiI) GetStorageSize(key types.StorageKey, blockHash types.Hash) (types.U64, error) {
	ret := _m.Called(key, blockHash)

	if len(ret) == 0 {
		panic("no return value specified for GetStorageSize")
	}

	var r0 types.U64
	var r1 error
	if rf, ok := ret.Get(0).(func(types.StorageKey, types.Hash) (types.U64, error)); ok {
		return rf(key, blockHash)
	}
	if rf, ok := ret.Get(0).(func(types.StorageKey, types.Hash) types.U64); ok {
		r0 = rf(key, blockHash)
	} else {
		r0 = ret.Get(0).(types.U64)
	}

	if rf, ok := ret.Get(1).(func(types.StorageKey, types.Hash) error); ok {
		r1 = rf(key, blockHash)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_GetStorageSize_Call struct {
	*mock.Call
}




func (_e *MockSubstrateApiI_Expecter) GetStorageSize(key interface{}, blockHash interface{}) *MockSubstrateApiI_GetStorageSize_Call {
	return &MockSubstrateApiI_GetStorageSize_Call{Call: _e.mock.On("GetStorageSize", key, blockHash)}
}

func (_c *MockSubstrateApiI_GetStorageSize_Call) Run(run func(key types.StorageKey, blockHash types.Hash)) *MockSubstrateApiI_GetStorageSize_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.StorageKey), args[1].(types.Hash))
	})
	return _c
}

func (_c *MockSubstrateApiI_GetStorageSize_Call) Return(_a0 types.U64, _a1 error) *MockSubstrateApiI_GetStorageSize_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSubstrateApiI_GetStorageSize_Call) RunAndReturn(run func(types.StorageKey, types.Hash) (types.U64, error)) *MockSubstrateApiI_GetStorageSize_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockSubstrateApiI) GetStorageSizeLatest(key types.StorageKey) (types.U64, error) {
	ret := _m.Called(key)

	if len(ret) == 0 {
		panic("no return value specified for GetStorageSizeLatest")
	}

	var r0 types.U64
	var r1 error
	if rf, ok := ret.Get(0).(func(types.StorageKey) (types.U64, error)); ok {
		return rf(key)
	}
	if rf, ok := ret.Get(0).(func(types.StorageKey) types.U64); ok {
		r0 = rf(key)
	} else {
		r0 = ret.Get(0).(types.U64)
	}

	if rf, ok := ret.Get(1).(func(types.StorageKey) error); ok {
		r1 = rf(key)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_GetStorageSizeLatest_Call struct {
	*mock.Call
}



func (_e *MockSubstrateApiI_Expecter) GetStorageSizeLatest(key interface{}) *MockSubstrateApiI_GetStorageSizeLatest_Call {
	return &MockSubstrateApiI_GetStorageSizeLatest_Call{Call: _e.mock.On("GetStorageSizeLatest", key)}
}

func (_c *MockSubstrateApiI_GetStorageSizeLatest_Call) Run(run func(key types.StorageKey)) *MockSubstrateApiI_GetStorageSizeLatest_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.StorageKey))
	})
	return _c
}

func (_c *MockSubstrateApiI_GetStorageSizeLatest_Call) Return(_a0 types.U64, _a1 error) *MockSubstrateApiI_GetStorageSizeLatest_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSubstrateApiI_GetStorageSizeLatest_Call) RunAndReturn(run func(types.StorageKey) (types.U64, error)) *MockSubstrateApiI_GetStorageSizeLatest_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockSubstrateApiI) PendingExtrinsics() ([]types.Extrinsic, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for PendingExtrinsics")
	}

	var r0 []types.Extrinsic
	var r1 error
	if rf, ok := ret.Get(0).(func() ([]types.Extrinsic, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() []types.Extrinsic); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]types.Extrinsic)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_PendingExtrinsics_Call struct {
	*mock.Call
}


func (_e *MockSubstrateApiI_Expecter) PendingExtrinsics() *MockSubstrateApiI_PendingExtrinsics_Call {
	return &MockSubstrateApiI_PendingExtrinsics_Call{Call: _e.mock.On("PendingExtrinsics")}
}

func (_c *MockSubstrateApiI_PendingExtrinsics_Call) Run(run func()) *MockSubstrateApiI_PendingExtrinsics_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockSubstrateApiI_PendingExtrinsics_Call) Return(_a0 []types.Extrinsic, _a1 error) *MockSubstrateApiI_PendingExtrinsics_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSubstrateApiI_PendingExtrinsics_Call) RunAndReturn(run func() ([]types.Extrinsic, error)) *MockSubstrateApiI_PendingExtrinsics_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockSubstrateApiI) QueryStorage(keys []types.StorageKey, startBlock types.Hash, block types.Hash) ([]types.StorageChangeSet, error) {
	ret := _m.Called(keys, startBlock, block)

	if len(ret) == 0 {
		panic("no return value specified for QueryStorage")
	}

	var r0 []types.StorageChangeSet
	var r1 error
	if rf, ok := ret.Get(0).(func([]types.StorageKey, types.Hash, types.Hash) ([]types.StorageChangeSet, error)); ok {
		return rf(keys, startBlock, block)
	}
	if rf, ok := ret.Get(0).(func([]types.StorageKey, types.Hash, types.Hash) []types.StorageChangeSet); ok {
		r0 = rf(keys, startBlock, block)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]types.StorageChangeSet)
		}
	}

	if rf, ok := ret.Get(1).(func([]types.StorageKey, types.Hash, types.Hash) error); ok {
		r1 = rf(keys, startBlock, block)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_QueryStorage_Call struct {
	*mock.Call
}





func (_e *MockSubstrateApiI_Expecter) QueryStorage(keys interface{}, startBlock interface{}, block interface{}) *MockSubstrateApiI_QueryStorage_Call {
	return &MockSubstrateApiI_QueryStorage_Call{Call: _e.mock.On("QueryStorage", keys, startBlock, block)}
}

func (_c *MockSubstrateApiI_QueryStorage_Call) Run(run func(keys []types.StorageKey, startBlock types.Hash, block types.Hash)) *MockSubstrateApiI_QueryStorage_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].([]types.StorageKey), args[1].(types.Hash), args[2].(types.Hash))
	})
	return _c
}

func (_c *MockSubstrateApiI_QueryStorage_Call) Return(_a0 []types.StorageChangeSet, _a1 error) *MockSubstrateApiI_QueryStorage_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSubstrateApiI_QueryStorage_Call) RunAndReturn(run func([]types.StorageKey, types.Hash, types.Hash) ([]types.StorageChangeSet, error)) *MockSubstrateApiI_QueryStorage_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockSubstrateApiI) QueryStorageAt(keys []types.StorageKey, block types.Hash) ([]types.StorageChangeSet, error) {
	ret := _m.Called(keys, block)

	if len(ret) == 0 {
		panic("no return value specified for QueryStorageAt")
	}

	var r0 []types.StorageChangeSet
	var r1 error
	if rf, ok := ret.Get(0).(func([]types.StorageKey, types.Hash) ([]types.StorageChangeSet, error)); ok {
		return rf(keys, block)
	}
	if rf, ok := ret.Get(0).(func([]types.StorageKey, types.Hash) []types.StorageChangeSet); ok {
		r0 = rf(keys, block)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]types.StorageChangeSet)
		}
	}

	if rf, ok := ret.Get(1).(func([]types.StorageKey, types.Hash) error); ok {
		r1 = rf(keys, block)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_QueryStorageAt_Call struct {
	*mock.Call
}




func (_e *MockSubstrateApiI_Expecter) QueryStorageAt(keys interface{}, block interface{}) *MockSubstrateApiI_QueryStorageAt_Call {
	return &MockSubstrateApiI_QueryStorageAt_Call{Call: _e.mock.On("QueryStorageAt", keys, block)}
}

func (_c *MockSubstrateApiI_QueryStorageAt_Call) Run(run func(keys []types.StorageKey, block types.Hash)) *MockSubstrateApiI_QueryStorageAt_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].([]types.StorageKey), args[1].(types.Hash))
	})
	return _c
}

func (_c *MockSubstrateApiI_QueryStorageAt_Call) Return(_a0 []types.StorageChangeSet, _a1 error) *MockSubstrateApiI_QueryStorageAt_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSubstrateApiI_QueryStorageAt_Call) RunAndReturn(run func([]types.StorageKey, types.Hash) ([]types.StorageChangeSet, error)) *MockSubstrateApiI_QueryStorageAt_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockSubstrateApiI) QueryStorageAtLatest(keys []types.StorageKey) ([]types.StorageChangeSet, error) {
	ret := _m.Called(keys)

	if len(ret) == 0 {
		panic("no return value specified for QueryStorageAtLatest")
	}

	var r0 []types.StorageChangeSet
	var r1 error
	if rf, ok := ret.Get(0).(func([]types.StorageKey) ([]types.StorageChangeSet, error)); ok {
		return rf(keys)
	}
	if rf, ok := ret.Get(0).(func([]types.StorageKey) []types.StorageChangeSet); ok {
		r0 = rf(keys)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]types.StorageChangeSet)
		}
	}

	if rf, ok := ret.Get(1).(func([]types.StorageKey) error); ok {
		r1 = rf(keys)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_QueryStorageAtLatest_Call struct {
	*mock.Call
}



func (_e *MockSubstrateApiI_Expecter) QueryStorageAtLatest(keys interface{}) *MockSubstrateApiI_QueryStorageAtLatest_Call {
	return &MockSubstrateApiI_QueryStorageAtLatest_Call{Call: _e.mock.On("QueryStorageAtLatest", keys)}
}

func (_c *MockSubstrateApiI_QueryStorageAtLatest_Call) Run(run func(keys []types.StorageKey)) *MockSubstrateApiI_QueryStorageAtLatest_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].([]types.StorageKey))
	})
	return _c
}

func (_c *MockSubstrateApiI_QueryStorageAtLatest_Call) Return(_a0 []types.StorageChangeSet, _a1 error) *MockSubstrateApiI_QueryStorageAtLatest_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSubstrateApiI_QueryStorageAtLatest_Call) RunAndReturn(run func([]types.StorageKey) ([]types.StorageChangeSet, error)) *MockSubstrateApiI_QueryStorageAtLatest_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockSubstrateApiI) QueryStorageLatest(keys []types.StorageKey, startBlock types.Hash) ([]types.StorageChangeSet, error) {
	ret := _m.Called(keys, startBlock)

	if len(ret) == 0 {
		panic("no return value specified for QueryStorageLatest")
	}

	var r0 []types.StorageChangeSet
	var r1 error
	if rf, ok := ret.Get(0).(func([]types.StorageKey, types.Hash) ([]types.StorageChangeSet, error)); ok {
		return rf(keys, startBlock)
	}
	if rf, ok := ret.Get(0).(func([]types.StorageKey, types.Hash) []types.StorageChangeSet); ok {
		r0 = rf(keys, startBlock)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]types.StorageChangeSet)
		}
	}

	if rf, ok := ret.Get(1).(func([]types.StorageKey, types.Hash) error); ok {
		r1 = rf(keys, startBlock)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_QueryStorageLatest_Call struct {
	*mock.Call
}




func (_e *MockSubstrateApiI_Expecter) QueryStorageLatest(keys interface{}, startBlock interface{}) *MockSubstrateApiI_QueryStorageLatest_Call {
	return &MockSubstrateApiI_QueryStorageLatest_Call{Call: _e.mock.On("QueryStorageLatest", keys, startBlock)}
}

func (_c *MockSubstrateApiI_QueryStorageLatest_Call) Run(run func(keys []types.StorageKey, startBlock types.Hash)) *MockSubstrateApiI_QueryStorageLatest_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].([]types.StorageKey), args[1].(types.Hash))
	})
	return _c
}

func (_c *MockSubstrateApiI_QueryStorageLatest_Call) Return(_a0 []types.StorageChangeSet, _a1 error) *MockSubstrateApiI_QueryStorageLatest_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSubstrateApiI_QueryStorageLatest_Call) RunAndReturn(run func([]types.StorageKey, types.Hash) ([]types.StorageChangeSet, error)) *MockSubstrateApiI_QueryStorageLatest_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockSubstrateApiI) SubmitAndWatchExtrinsic(xt types.Extrinsic) (*author.ExtrinsicStatusSubscription, error) {
	ret := _m.Called(xt)

	if len(ret) == 0 {
		panic("no return value specified for SubmitAndWatchExtrinsic")
	}

	var r0 *author.ExtrinsicStatusSubscription
	var r1 error
	if rf, ok := ret.Get(0).(func(types.Extrinsic) (*author.ExtrinsicStatusSubscription, error)); ok {
		return rf(xt)
	}
	if rf, ok := ret.Get(0).(func(types.Extrinsic) *author.ExtrinsicStatusSubscription); ok {
		r0 = rf(xt)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*author.ExtrinsicStatusSubscription)
		}
	}

	if rf, ok := ret.Get(1).(func(types.Extrinsic) error); ok {
		r1 = rf(xt)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_SubmitAndWatchExtrinsic_Call struct {
	*mock.Call
}



func (_e *MockSubstrateApiI_Expecter) SubmitAndWatchExtrinsic(xt interface{}) *MockSubstrateApiI_SubmitAndWatchExtrinsic_Call {
	return &MockSubstrateApiI_SubmitAndWatchExtrinsic_Call{Call: _e.mock.On("SubmitAndWatchExtrinsic", xt)}
}

func (_c *MockSubstrateApiI_SubmitAndWatchExtrinsic_Call) Run(run func(xt types.Extrinsic)) *MockSubstrateApiI_SubmitAndWatchExtrinsic_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.Extrinsic))
	})
	return _c
}

func (_c *MockSubstrateApiI_SubmitAndWatchExtrinsic_Call) Return(_a0 *author.ExtrinsicStatusSubscription, _a1 error) *MockSubstrateApiI_SubmitAndWatchExtrinsic_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSubstrateApiI_SubmitAndWatchExtrinsic_Call) RunAndReturn(run func(types.Extrinsic) (*author.ExtrinsicStatusSubscription, error)) *MockSubstrateApiI_SubmitAndWatchExtrinsic_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockSubstrateApiI) SubmitExtrinsic(xt types.Extrinsic) (types.Hash, error) {
	ret := _m.Called(xt)

	if len(ret) == 0 {
		panic("no return value specified for SubmitExtrinsic")
	}

	var r0 types.Hash
	var r1 error
	if rf, ok := ret.Get(0).(func(types.Extrinsic) (types.Hash, error)); ok {
		return rf(xt)
	}
	if rf, ok := ret.Get(0).(func(types.Extrinsic) types.Hash); ok {
		r0 = rf(xt)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(types.Hash)
		}
	}

	if rf, ok := ret.Get(1).(func(types.Extrinsic) error); ok {
		r1 = rf(xt)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_SubmitExtrinsic_Call struct {
	*mock.Call
}



func (_e *MockSubstrateApiI_Expecter) SubmitExtrinsic(xt interface{}) *MockSubstrateApiI_SubmitExtrinsic_Call {
	return &MockSubstrateApiI_SubmitExtrinsic_Call{Call: _e.mock.On("SubmitExtrinsic", xt)}
}

func (_c *MockSubstrateApiI_SubmitExtrinsic_Call) Run(run func(xt types.Extrinsic)) *MockSubstrateApiI_SubmitExtrinsic_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.Extrinsic))
	})
	return _c
}

func (_c *MockSubstrateApiI_SubmitExtrinsic_Call) Return(_a0 types.Hash, _a1 error) *MockSubstrateApiI_SubmitExtrinsic_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSubstrateApiI_SubmitExtrinsic_Call) RunAndReturn(run func(types.Extrinsic) (types.Hash, error)) *MockSubstrateApiI_SubmitExtrinsic_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockSubstrateApiI) SubscribeFinalizedHeads() (*chain.FinalizedHeadsSubscription, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for SubscribeFinalizedHeads")
	}

	var r0 *chain.FinalizedHeadsSubscription
	var r1 error
	if rf, ok := ret.Get(0).(func() (*chain.FinalizedHeadsSubscription, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() *chain.FinalizedHeadsSubscription); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*chain.FinalizedHeadsSubscription)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_SubscribeFinalizedHeads_Call struct {
	*mock.Call
}


func (_e *MockSubstrateApiI_Expecter) SubscribeFinalizedHeads() *MockSubstrateApiI_SubscribeFinalizedHeads_Call {
	return &MockSubstrateApiI_SubscribeFinalizedHeads_Call{Call: _e.mock.On("SubscribeFinalizedHeads")}
}

func (_c *MockSubstrateApiI_SubscribeFinalizedHeads_Call) Run(run func()) *MockSubstrateApiI_SubscribeFinalizedHeads_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockSubstrateApiI_SubscribeFinalizedHeads_Call) Return(_a0 *chain.FinalizedHeadsSubscription, _a1 error) *MockSubstrateApiI_SubscribeFinalizedHeads_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSubstrateApiI_SubscribeFinalizedHeads_Call) RunAndReturn(run func() (*chain.FinalizedHeadsSubscription, error)) *MockSubstrateApiI_SubscribeFinalizedHeads_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockSubstrateApiI) SubscribeNewHeads() (*chain.NewHeadsSubscription, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for SubscribeNewHeads")
	}

	var r0 *chain.NewHeadsSubscription
	var r1 error
	if rf, ok := ret.Get(0).(func() (*chain.NewHeadsSubscription, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() *chain.NewHeadsSubscription); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*chain.NewHeadsSubscription)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_SubscribeNewHeads_Call struct {
	*mock.Call
}


func (_e *MockSubstrateApiI_Expecter) SubscribeNewHeads() *MockSubstrateApiI_SubscribeNewHeads_Call {
	return &MockSubstrateApiI_SubscribeNewHeads_Call{Call: _e.mock.On("SubscribeNewHeads")}
}

func (_c *MockSubstrateApiI_SubscribeNewHeads_Call) Run(run func()) *MockSubstrateApiI_SubscribeNewHeads_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockSubstrateApiI_SubscribeNewHeads_Call) Return(_a0 *chain.NewHeadsSubscription, _a1 error) *MockSubstrateApiI_SubscribeNewHeads_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSubstrateApiI_SubscribeNewHeads_Call) RunAndReturn(run func() (*chain.NewHeadsSubscription, error)) *MockSubstrateApiI_SubscribeNewHeads_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockSubstrateApiI) SubscribeRuntimeVersion() (*state.RuntimeVersionSubscription, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for SubscribeRuntimeVersion")
	}

	var r0 *state.RuntimeVersionSubscription
	var r1 error
	if rf, ok := ret.Get(0).(func() (*state.RuntimeVersionSubscription, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() *state.RuntimeVersionSubscription); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*state.RuntimeVersionSubscription)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_SubscribeRuntimeVersion_Call struct {
	*mock.Call
}


func (_e *MockSubstrateApiI_Expecter) SubscribeRuntimeVersion() *MockSubstrateApiI_SubscribeRuntimeVersion_Call {
	return &MockSubstrateApiI_SubscribeRuntimeVersion_Call{Call: _e.mock.On("SubscribeRuntimeVersion")}
}

func (_c *MockSubstrateApiI_SubscribeRuntimeVersion_Call) Run(run func()) *MockSubstrateApiI_SubscribeRuntimeVersion_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockSubstrateApiI_SubscribeRuntimeVersion_Call) Return(_a0 *state.RuntimeVersionSubscription, _a1 error) *MockSubstrateApiI_SubscribeRuntimeVersion_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSubstrateApiI_SubscribeRuntimeVersion_Call) RunAndReturn(run func() (*state.RuntimeVersionSubscription, error)) *MockSubstrateApiI_SubscribeRuntimeVersion_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockSubstrateApiI) SubscribeStorageRaw(keys []types.StorageKey) (*state.StorageSubscription, error) {
	ret := _m.Called(keys)

	if len(ret) == 0 {
		panic("no return value specified for SubscribeStorageRaw")
	}

	var r0 *state.StorageSubscription
	var r1 error
	if rf, ok := ret.Get(0).(func([]types.StorageKey) (*state.StorageSubscription, error)); ok {
		return rf(keys)
	}
	if rf, ok := ret.Get(0).(func([]types.StorageKey) *state.StorageSubscription); ok {
		r0 = rf(keys)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*state.StorageSubscription)
		}
	}

	if rf, ok := ret.Get(1).(func([]types.StorageKey) error); ok {
		r1 = rf(keys)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockSubstrateApiI_SubscribeStorageRaw_Call struct {
	*mock.Call
}



func (_e *MockSubstrateApiI_Expecter) SubscribeStorageRaw(keys interface{}) *MockSubstrateApiI_SubscribeStorageRaw_Call {
	return &MockSubstrateApiI_SubscribeStorageRaw_Call{Call: _e.mock.On("SubscribeStorageRaw", keys)}
}

func (_c *MockSubstrateApiI_SubscribeStorageRaw_Call) Run(run func(keys []types.StorageKey)) *MockSubstrateApiI_SubscribeStorageRaw_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].([]types.StorageKey))
	})
	return _c
}

func (_c *MockSubstrateApiI_SubscribeStorageRaw_Call) Return(_a0 *state.StorageSubscription, _a1 error) *MockSubstrateApiI_SubscribeStorageRaw_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSubstrateApiI_SubscribeStorageRaw_Call) RunAndReturn(run func([]types.StorageKey) (*state.StorageSubscription, error)) *MockSubstrateApiI_SubscribeStorageRaw_Call {
	_c.Call.Return(run)
	return _c
}



func NewMockSubstrateApiI(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockSubstrateApiI {
	mock := &MockSubstrateApiI{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
