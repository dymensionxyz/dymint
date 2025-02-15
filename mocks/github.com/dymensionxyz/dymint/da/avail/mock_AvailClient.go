// Code generated by mockery v2.50.2. DO NOT EDIT.

package avail

import (
	sdk "github.com/availproject/avail-go-sdk/sdk"
	mock "github.com/stretchr/testify/mock"
)

// MockAvailClient is an autogenerated mock type for the AvailClient type
type MockAvailClient struct {
	mock.Mock
}

type MockAvailClient_Expecter struct {
	mock *mock.Mock
}

func (_m *MockAvailClient) EXPECT() *MockAvailClient_Expecter {
	return &MockAvailClient_Expecter{mock: &_m.Mock}
}

// GetAccountAddress provides a mock function with no fields
func (_m *MockAvailClient) GetAccountAddress() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetAccountAddress")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// MockAvailClient_GetAccountAddress_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetAccountAddress'
type MockAvailClient_GetAccountAddress_Call struct {
	*mock.Call
}

// GetAccountAddress is a helper method to define mock.On call
func (_e *MockAvailClient_Expecter) GetAccountAddress() *MockAvailClient_GetAccountAddress_Call {
	return &MockAvailClient_GetAccountAddress_Call{Call: _e.mock.On("GetAccountAddress")}
}

func (_c *MockAvailClient_GetAccountAddress_Call) Run(run func()) *MockAvailClient_GetAccountAddress_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockAvailClient_GetAccountAddress_Call) Return(_a0 string) *MockAvailClient_GetAccountAddress_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockAvailClient_GetAccountAddress_Call) RunAndReturn(run func() string) *MockAvailClient_GetAccountAddress_Call {
	_c.Call.Return(run)
	return _c
}

// GetBlobsBySigner provides a mock function with given fields: blockHash, accountAddress
func (_m *MockAvailClient) GetBlobsBySigner(blockHash string, accountAddress string) ([]sdk.DataSubmission, error) {
	ret := _m.Called(blockHash, accountAddress)

	if len(ret) == 0 {
		panic("no return value specified for GetBlobsBySigner")
	}

	var r0 []sdk.DataSubmission
	var r1 error
	if rf, ok := ret.Get(0).(func(string, string) ([]sdk.DataSubmission, error)); ok {
		return rf(blockHash, accountAddress)
	}
	if rf, ok := ret.Get(0).(func(string, string) []sdk.DataSubmission); ok {
		r0 = rf(blockHash, accountAddress)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]sdk.DataSubmission)
		}
	}

	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(blockHash, accountAddress)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockAvailClient_GetBlobsBySigner_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetBlobsBySigner'
type MockAvailClient_GetBlobsBySigner_Call struct {
	*mock.Call
}

// GetBlobsBySigner is a helper method to define mock.On call
//   - blockHash string
//   - accountAddress string
func (_e *MockAvailClient_Expecter) GetBlobsBySigner(blockHash interface{}, accountAddress interface{}) *MockAvailClient_GetBlobsBySigner_Call {
	return &MockAvailClient_GetBlobsBySigner_Call{Call: _e.mock.On("GetBlobsBySigner", blockHash, accountAddress)}
}

func (_c *MockAvailClient_GetBlobsBySigner_Call) Run(run func(blockHash string, accountAddress string)) *MockAvailClient_GetBlobsBySigner_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string), args[1].(string))
	})
	return _c
}

func (_c *MockAvailClient_GetBlobsBySigner_Call) Return(_a0 []sdk.DataSubmission, _a1 error) *MockAvailClient_GetBlobsBySigner_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockAvailClient_GetBlobsBySigner_Call) RunAndReturn(run func(string, string) ([]sdk.DataSubmission, error)) *MockAvailClient_GetBlobsBySigner_Call {
	_c.Call.Return(run)
	return _c
}

// GetBlock provides a mock function with given fields: blockHash
func (_m *MockAvailClient) GetBlock(blockHash string) (sdk.Block, error) {
	ret := _m.Called(blockHash)

	if len(ret) == 0 {
		panic("no return value specified for GetBlock")
	}

	var r0 sdk.Block
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (sdk.Block, error)); ok {
		return rf(blockHash)
	}
	if rf, ok := ret.Get(0).(func(string) sdk.Block); ok {
		r0 = rf(blockHash)
	} else {
		r0 = ret.Get(0).(sdk.Block)
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(blockHash)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockAvailClient_GetBlock_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetBlock'
type MockAvailClient_GetBlock_Call struct {
	*mock.Call
}

// GetBlock is a helper method to define mock.On call
//   - blockHash string
func (_e *MockAvailClient_Expecter) GetBlock(blockHash interface{}) *MockAvailClient_GetBlock_Call {
	return &MockAvailClient_GetBlock_Call{Call: _e.mock.On("GetBlock", blockHash)}
}

func (_c *MockAvailClient_GetBlock_Call) Run(run func(blockHash string)) *MockAvailClient_GetBlock_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *MockAvailClient_GetBlock_Call) Return(_a0 sdk.Block, _a1 error) *MockAvailClient_GetBlock_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockAvailClient_GetBlock_Call) RunAndReturn(run func(string) (sdk.Block, error)) *MockAvailClient_GetBlock_Call {
	_c.Call.Return(run)
	return _c
}

// IsSyncing provides a mock function with no fields
func (_m *MockAvailClient) IsSyncing() (bool, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for IsSyncing")
	}

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func() (bool, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockAvailClient_IsSyncing_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'IsSyncing'
type MockAvailClient_IsSyncing_Call struct {
	*mock.Call
}

// IsSyncing is a helper method to define mock.On call
func (_e *MockAvailClient_Expecter) IsSyncing() *MockAvailClient_IsSyncing_Call {
	return &MockAvailClient_IsSyncing_Call{Call: _e.mock.On("IsSyncing")}
}

func (_c *MockAvailClient_IsSyncing_Call) Run(run func()) *MockAvailClient_IsSyncing_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockAvailClient_IsSyncing_Call) Return(_a0 bool, _a1 error) *MockAvailClient_IsSyncing_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockAvailClient_IsSyncing_Call) RunAndReturn(run func() (bool, error)) *MockAvailClient_IsSyncing_Call {
	_c.Call.Return(run)
	return _c
}

// SubmitData provides a mock function with given fields: data
func (_m *MockAvailClient) SubmitData(data []byte) (string, error) {
	ret := _m.Called(data)

	if len(ret) == 0 {
		panic("no return value specified for SubmitData")
	}

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func([]byte) (string, error)); ok {
		return rf(data)
	}
	if rf, ok := ret.Get(0).(func([]byte) string); ok {
		r0 = rf(data)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func([]byte) error); ok {
		r1 = rf(data)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockAvailClient_SubmitData_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SubmitData'
type MockAvailClient_SubmitData_Call struct {
	*mock.Call
}

// SubmitData is a helper method to define mock.On call
//   - data []byte
func (_e *MockAvailClient_Expecter) SubmitData(data interface{}) *MockAvailClient_SubmitData_Call {
	return &MockAvailClient_SubmitData_Call{Call: _e.mock.On("SubmitData", data)}
}

func (_c *MockAvailClient_SubmitData_Call) Run(run func(data []byte)) *MockAvailClient_SubmitData_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].([]byte))
	})
	return _c
}

func (_c *MockAvailClient_SubmitData_Call) Return(_a0 string, _a1 error) *MockAvailClient_SubmitData_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockAvailClient_SubmitData_Call) RunAndReturn(run func([]byte) (string, error)) *MockAvailClient_SubmitData_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockAvailClient creates a new instance of MockAvailClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockAvailClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockAvailClient {
	mock := &MockAvailClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
