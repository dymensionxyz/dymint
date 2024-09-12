<<<<<<< HEAD
// Code generated by mockery v2.46.0. DO NOT EDIT.
=======
// Code generated by mockery v2.45.1. DO NOT EDIT.
>>>>>>> c7939c7 (first check on mockFraud Handler)

package types

import (
	context "context"

	blob "github.com/celestiaorg/celestia-openrpc/types/blob"

	header "github.com/celestiaorg/celestia-openrpc/types/header"

	mock "github.com/stretchr/testify/mock"

	share "github.com/celestiaorg/celestia-openrpc/types/share"
)

// MockCelestiaRPCClient is an autogenerated mock type for the CelestiaRPCClient type
type MockCelestiaRPCClient struct {
	mock.Mock
}

type MockCelestiaRPCClient_Expecter struct {
	mock *mock.Mock
}

func (_m *MockCelestiaRPCClient) EXPECT() *MockCelestiaRPCClient_Expecter {
	return &MockCelestiaRPCClient_Expecter{mock: &_m.Mock}
}

// Get provides a mock function with given fields: ctx, height, namespace, commitment
func (_m *MockCelestiaRPCClient) Get(ctx context.Context, height uint64, namespace share.Namespace, commitment blob.Commitment) (*blob.Blob, error) {
	ret := _m.Called(ctx, height, namespace, commitment)

	if len(ret) == 0 {
		panic("no return value specified for Get")
	}

	var r0 *blob.Blob
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, uint64, share.Namespace, blob.Commitment) (*blob.Blob, error)); ok {
		return rf(ctx, height, namespace, commitment)
	}
	if rf, ok := ret.Get(0).(func(context.Context, uint64, share.Namespace, blob.Commitment) *blob.Blob); ok {
		r0 = rf(ctx, height, namespace, commitment)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*blob.Blob)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, uint64, share.Namespace, blob.Commitment) error); ok {
		r1 = rf(ctx, height, namespace, commitment)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockCelestiaRPCClient_Get_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Get'
type MockCelestiaRPCClient_Get_Call struct {
	*mock.Call
}

// Get is a helper method to define mock.On call
//   - ctx context.Context
//   - height uint64
//   - namespace share.Namespace
//   - commitment blob.Commitment
func (_e *MockCelestiaRPCClient_Expecter) Get(ctx interface{}, height interface{}, namespace interface{}, commitment interface{}) *MockCelestiaRPCClient_Get_Call {
	return &MockCelestiaRPCClient_Get_Call{Call: _e.mock.On("Get", ctx, height, namespace, commitment)}
}

func (_c *MockCelestiaRPCClient_Get_Call) Run(run func(ctx context.Context, height uint64, namespace share.Namespace, commitment blob.Commitment)) *MockCelestiaRPCClient_Get_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(uint64), args[2].(share.Namespace), args[3].(blob.Commitment))
	})
	return _c
}

func (_c *MockCelestiaRPCClient_Get_Call) Return(_a0 *blob.Blob, _a1 error) *MockCelestiaRPCClient_Get_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockCelestiaRPCClient_Get_Call) RunAndReturn(run func(context.Context, uint64, share.Namespace, blob.Commitment) (*blob.Blob, error)) *MockCelestiaRPCClient_Get_Call {
	_c.Call.Return(run)
	return _c
}

// GetAll provides a mock function with given fields: _a0, _a1, _a2
func (_m *MockCelestiaRPCClient) GetAll(_a0 context.Context, _a1 uint64, _a2 []share.Namespace) ([]*blob.Blob, error) {
	ret := _m.Called(_a0, _a1, _a2)

	if len(ret) == 0 {
		panic("no return value specified for GetAll")
	}

	var r0 []*blob.Blob
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, uint64, []share.Namespace) ([]*blob.Blob, error)); ok {
		return rf(_a0, _a1, _a2)
	}
	if rf, ok := ret.Get(0).(func(context.Context, uint64, []share.Namespace) []*blob.Blob); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*blob.Blob)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, uint64, []share.Namespace) error); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockCelestiaRPCClient_GetAll_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetAll'
type MockCelestiaRPCClient_GetAll_Call struct {
	*mock.Call
}

// GetAll is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 uint64
//   - _a2 []share.Namespace
func (_e *MockCelestiaRPCClient_Expecter) GetAll(_a0 interface{}, _a1 interface{}, _a2 interface{}) *MockCelestiaRPCClient_GetAll_Call {
	return &MockCelestiaRPCClient_GetAll_Call{Call: _e.mock.On("GetAll", _a0, _a1, _a2)}
}

func (_c *MockCelestiaRPCClient_GetAll_Call) Run(run func(_a0 context.Context, _a1 uint64, _a2 []share.Namespace)) *MockCelestiaRPCClient_GetAll_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(uint64), args[2].([]share.Namespace))
	})
	return _c
}

func (_c *MockCelestiaRPCClient_GetAll_Call) Return(_a0 []*blob.Blob, _a1 error) *MockCelestiaRPCClient_GetAll_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockCelestiaRPCClient_GetAll_Call) RunAndReturn(run func(context.Context, uint64, []share.Namespace) ([]*blob.Blob, error)) *MockCelestiaRPCClient_GetAll_Call {
	_c.Call.Return(run)
	return _c
}

// GetByHeight provides a mock function with given fields: ctx, height
func (_m *MockCelestiaRPCClient) GetByHeight(ctx context.Context, height uint64) (*header.ExtendedHeader, error) {
	ret := _m.Called(ctx, height)

	if len(ret) == 0 {
		panic("no return value specified for GetByHeight")
	}

	var r0 *header.ExtendedHeader
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, uint64) (*header.ExtendedHeader, error)); ok {
		return rf(ctx, height)
	}
	if rf, ok := ret.Get(0).(func(context.Context, uint64) *header.ExtendedHeader); ok {
		r0 = rf(ctx, height)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*header.ExtendedHeader)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, uint64) error); ok {
		r1 = rf(ctx, height)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockCelestiaRPCClient_GetByHeight_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetByHeight'
type MockCelestiaRPCClient_GetByHeight_Call struct {
	*mock.Call
}

// GetByHeight is a helper method to define mock.On call
//   - ctx context.Context
//   - height uint64
func (_e *MockCelestiaRPCClient_Expecter) GetByHeight(ctx interface{}, height interface{}) *MockCelestiaRPCClient_GetByHeight_Call {
	return &MockCelestiaRPCClient_GetByHeight_Call{Call: _e.mock.On("GetByHeight", ctx, height)}
}

func (_c *MockCelestiaRPCClient_GetByHeight_Call) Run(run func(ctx context.Context, height uint64)) *MockCelestiaRPCClient_GetByHeight_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(uint64))
	})
	return _c
}

func (_c *MockCelestiaRPCClient_GetByHeight_Call) Return(_a0 *header.ExtendedHeader, _a1 error) *MockCelestiaRPCClient_GetByHeight_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockCelestiaRPCClient_GetByHeight_Call) RunAndReturn(run func(context.Context, uint64) (*header.ExtendedHeader, error)) *MockCelestiaRPCClient_GetByHeight_Call {
	_c.Call.Return(run)
	return _c
}

// GetProof provides a mock function with given fields: ctx, height, namespace, commitment
func (_m *MockCelestiaRPCClient) GetProof(ctx context.Context, height uint64, namespace share.Namespace, commitment blob.Commitment) (*blob.Proof, error) {
	ret := _m.Called(ctx, height, namespace, commitment)

	if len(ret) == 0 {
		panic("no return value specified for GetProof")
	}

	var r0 *blob.Proof
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, uint64, share.Namespace, blob.Commitment) (*blob.Proof, error)); ok {
		return rf(ctx, height, namespace, commitment)
	}
	if rf, ok := ret.Get(0).(func(context.Context, uint64, share.Namespace, blob.Commitment) *blob.Proof); ok {
		r0 = rf(ctx, height, namespace, commitment)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*blob.Proof)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, uint64, share.Namespace, blob.Commitment) error); ok {
		r1 = rf(ctx, height, namespace, commitment)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockCelestiaRPCClient_GetProof_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetProof'
type MockCelestiaRPCClient_GetProof_Call struct {
	*mock.Call
}

// GetProof is a helper method to define mock.On call
//   - ctx context.Context
//   - height uint64
//   - namespace share.Namespace
//   - commitment blob.Commitment
func (_e *MockCelestiaRPCClient_Expecter) GetProof(ctx interface{}, height interface{}, namespace interface{}, commitment interface{}) *MockCelestiaRPCClient_GetProof_Call {
	return &MockCelestiaRPCClient_GetProof_Call{Call: _e.mock.On("GetProof", ctx, height, namespace, commitment)}
}

func (_c *MockCelestiaRPCClient_GetProof_Call) Run(run func(ctx context.Context, height uint64, namespace share.Namespace, commitment blob.Commitment)) *MockCelestiaRPCClient_GetProof_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(uint64), args[2].(share.Namespace), args[3].(blob.Commitment))
	})
	return _c
}

func (_c *MockCelestiaRPCClient_GetProof_Call) Return(_a0 *blob.Proof, _a1 error) *MockCelestiaRPCClient_GetProof_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockCelestiaRPCClient_GetProof_Call) RunAndReturn(run func(context.Context, uint64, share.Namespace, blob.Commitment) (*blob.Proof, error)) *MockCelestiaRPCClient_GetProof_Call {
	_c.Call.Return(run)
	return _c
}

// Included provides a mock function with given fields: ctx, height, namespace, proof, commitment
func (_m *MockCelestiaRPCClient) Included(ctx context.Context, height uint64, namespace share.Namespace, proof *blob.Proof, commitment blob.Commitment) (bool, error) {
	ret := _m.Called(ctx, height, namespace, proof, commitment)

	if len(ret) == 0 {
		panic("no return value specified for Included")
	}

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, uint64, share.Namespace, *blob.Proof, blob.Commitment) (bool, error)); ok {
		return rf(ctx, height, namespace, proof, commitment)
	}
	if rf, ok := ret.Get(0).(func(context.Context, uint64, share.Namespace, *blob.Proof, blob.Commitment) bool); ok {
		r0 = rf(ctx, height, namespace, proof, commitment)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(context.Context, uint64, share.Namespace, *blob.Proof, blob.Commitment) error); ok {
		r1 = rf(ctx, height, namespace, proof, commitment)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockCelestiaRPCClient_Included_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Included'
type MockCelestiaRPCClient_Included_Call struct {
	*mock.Call
}

// Included is a helper method to define mock.On call
//   - ctx context.Context
//   - height uint64
//   - namespace share.Namespace
//   - proof *blob.Proof
//   - commitment blob.Commitment
func (_e *MockCelestiaRPCClient_Expecter) Included(ctx interface{}, height interface{}, namespace interface{}, proof interface{}, commitment interface{}) *MockCelestiaRPCClient_Included_Call {
	return &MockCelestiaRPCClient_Included_Call{Call: _e.mock.On("Included", ctx, height, namespace, proof, commitment)}
}

func (_c *MockCelestiaRPCClient_Included_Call) Run(run func(ctx context.Context, height uint64, namespace share.Namespace, proof *blob.Proof, commitment blob.Commitment)) *MockCelestiaRPCClient_Included_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(uint64), args[2].(share.Namespace), args[3].(*blob.Proof), args[4].(blob.Commitment))
	})
	return _c
}

func (_c *MockCelestiaRPCClient_Included_Call) Return(_a0 bool, _a1 error) *MockCelestiaRPCClient_Included_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockCelestiaRPCClient_Included_Call) RunAndReturn(run func(context.Context, uint64, share.Namespace, *blob.Proof, blob.Commitment) (bool, error)) *MockCelestiaRPCClient_Included_Call {
	_c.Call.Return(run)
	return _c
}

// Submit provides a mock function with given fields: ctx, blobs, options
func (_m *MockCelestiaRPCClient) Submit(ctx context.Context, blobs []*blob.Blob, options *blob.SubmitOptions) (uint64, error) {
	ret := _m.Called(ctx, blobs, options)

	if len(ret) == 0 {
		panic("no return value specified for Submit")
	}

	var r0 uint64
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, []*blob.Blob, *blob.SubmitOptions) (uint64, error)); ok {
		return rf(ctx, blobs, options)
	}
	if rf, ok := ret.Get(0).(func(context.Context, []*blob.Blob, *blob.SubmitOptions) uint64); ok {
		r0 = rf(ctx, blobs, options)
	} else {
		r0 = ret.Get(0).(uint64)
	}

	if rf, ok := ret.Get(1).(func(context.Context, []*blob.Blob, *blob.SubmitOptions) error); ok {
		r1 = rf(ctx, blobs, options)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockCelestiaRPCClient_Submit_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Submit'
type MockCelestiaRPCClient_Submit_Call struct {
	*mock.Call
}

// Submit is a helper method to define mock.On call
//   - ctx context.Context
//   - blobs []*blob.Blob
//   - options *blob.SubmitOptions
func (_e *MockCelestiaRPCClient_Expecter) Submit(ctx interface{}, blobs interface{}, options interface{}) *MockCelestiaRPCClient_Submit_Call {
	return &MockCelestiaRPCClient_Submit_Call{Call: _e.mock.On("Submit", ctx, blobs, options)}
}

func (_c *MockCelestiaRPCClient_Submit_Call) Run(run func(ctx context.Context, blobs []*blob.Blob, options *blob.SubmitOptions)) *MockCelestiaRPCClient_Submit_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].([]*blob.Blob), args[2].(*blob.SubmitOptions))
	})
	return _c
}

func (_c *MockCelestiaRPCClient_Submit_Call) Return(_a0 uint64, _a1 error) *MockCelestiaRPCClient_Submit_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockCelestiaRPCClient_Submit_Call) RunAndReturn(run func(context.Context, []*blob.Blob, *blob.SubmitOptions) (uint64, error)) *MockCelestiaRPCClient_Submit_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockCelestiaRPCClient creates a new instance of MockCelestiaRPCClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockCelestiaRPCClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockCelestiaRPCClient {
	mock := &MockCelestiaRPCClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
