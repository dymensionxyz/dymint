package p2p

import (
	mock "github.com/stretchr/testify/mock"
	crypto "github.com/tendermint/tendermint/crypto"
)

type MockProposerGetter struct {
	mock.Mock
}

type MockProposerGetter_Expecter struct {
	mock *mock.Mock
}

func (_m *MockProposerGetter) EXPECT() *MockProposerGetter_Expecter {
	return &MockProposerGetter_Expecter{mock: &_m.Mock}
}

func (_m *MockProposerGetter) GetProposerPubKey() crypto.PubKey {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetProposerPubKey")
	}

	var r0 crypto.PubKey
	if rf, ok := ret.Get(0).(func() crypto.PubKey); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(crypto.PubKey)
		}
	}

	return r0
}

type MockProposerGetter_GetProposerPubKey_Call struct {
	*mock.Call
}

func (_e *MockProposerGetter_Expecter) GetProposerPubKey() *MockProposerGetter_GetProposerPubKey_Call {
	return &MockProposerGetter_GetProposerPubKey_Call{Call: _e.mock.On("GetProposerPubKey")}
}

func (_c *MockProposerGetter_GetProposerPubKey_Call) Run(run func()) *MockProposerGetter_GetProposerPubKey_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockProposerGetter_GetProposerPubKey_Call) Return(_a0 crypto.PubKey) *MockProposerGetter_GetProposerPubKey_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockProposerGetter_GetProposerPubKey_Call) RunAndReturn(run func() crypto.PubKey) *MockProposerGetter_GetProposerPubKey_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockProposerGetter) GetRevision() uint64 {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetRevision")
	}

	var r0 uint64
	if rf, ok := ret.Get(0).(func() uint64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint64)
	}

	return r0
}

type MockProposerGetter_GetRevision_Call struct {
	*mock.Call
}

func (_e *MockProposerGetter_Expecter) GetRevision() *MockProposerGetter_GetRevision_Call {
	return &MockProposerGetter_GetRevision_Call{Call: _e.mock.On("GetRevision")}
}

func (_c *MockProposerGetter_GetRevision_Call) Run(run func()) *MockProposerGetter_GetRevision_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockProposerGetter_GetRevision_Call) Return(_a0 uint64) *MockProposerGetter_GetRevision_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockProposerGetter_GetRevision_Call) RunAndReturn(run func() uint64) *MockProposerGetter_GetRevision_Call {
	_c.Call.Return(run)
	return _c
}

func NewMockProposerGetter(t interface {
	mock.TestingT
	Cleanup(func())
},
) *MockProposerGetter {
	mock := &MockProposerGetter{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
