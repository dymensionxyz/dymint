package types

import (
	context "context"

	grpc "google.golang.org/grpc"

	mock "github.com/stretchr/testify/mock"

	types "github.com/dymensionxyz/dymint/types/pb/dymensionxyz/dymension/sequencer"
)

type MockQueryClient struct {
	mock.Mock
}

type MockQueryClient_Expecter struct {
	mock *mock.Mock
}

func (_m *MockQueryClient) EXPECT() *MockQueryClient_Expecter {
	return &MockQueryClient_Expecter{mock: &_m.Mock}
}

func (_m *MockQueryClient) GetNextProposerByRollapp(ctx context.Context, in *types.QueryGetNextProposerByRollappRequest, opts ...grpc.CallOption) (*types.QueryGetNextProposerByRollappResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for GetNextProposerByRollapp")
	}

	var r0 *types.QueryGetNextProposerByRollappResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *types.QueryGetNextProposerByRollappRequest, ...grpc.CallOption) (*types.QueryGetNextProposerByRollappResponse, error)); ok {
		return rf(ctx, in, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *types.QueryGetNextProposerByRollappRequest, ...grpc.CallOption) *types.QueryGetNextProposerByRollappResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.QueryGetNextProposerByRollappResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *types.QueryGetNextProposerByRollappRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type MockQueryClient_GetNextProposerByRollapp_Call struct {
	*mock.Call
}

func (_e *MockQueryClient_Expecter) GetNextProposerByRollapp(ctx interface{}, in interface{}, opts ...interface{}) *MockQueryClient_GetNextProposerByRollapp_Call {
	return &MockQueryClient_GetNextProposerByRollapp_Call{Call: _e.mock.On("GetNextProposerByRollapp",
		append([]interface{}{ctx, in}, opts...)...)}
}

func (_c *MockQueryClient_GetNextProposerByRollapp_Call) Run(run func(ctx context.Context, in *types.QueryGetNextProposerByRollappRequest, opts ...grpc.CallOption)) *MockQueryClient_GetNextProposerByRollapp_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]grpc.CallOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(grpc.CallOption)
			}
		}
		run(args[0].(context.Context), args[1].(*types.QueryGetNextProposerByRollappRequest), variadicArgs...)
	})
	return _c
}

func (_c *MockQueryClient_GetNextProposerByRollapp_Call) Return(_a0 *types.QueryGetNextProposerByRollappResponse, _a1 error) *MockQueryClient_GetNextProposerByRollapp_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockQueryClient_GetNextProposerByRollapp_Call) RunAndReturn(run func(context.Context, *types.QueryGetNextProposerByRollappRequest, ...grpc.CallOption) (*types.QueryGetNextProposerByRollappResponse, error)) *MockQueryClient_GetNextProposerByRollapp_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockQueryClient) GetProposerByRollapp(ctx context.Context, in *types.QueryGetProposerByRollappRequest, opts ...grpc.CallOption) (*types.QueryGetProposerByRollappResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for GetProposerByRollapp")
	}

	var r0 *types.QueryGetProposerByRollappResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *types.QueryGetProposerByRollappRequest, ...grpc.CallOption) (*types.QueryGetProposerByRollappResponse, error)); ok {
		return rf(ctx, in, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *types.QueryGetProposerByRollappRequest, ...grpc.CallOption) *types.QueryGetProposerByRollappResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.QueryGetProposerByRollappResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *types.QueryGetProposerByRollappRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type MockQueryClient_GetProposerByRollapp_Call struct {
	*mock.Call
}

func (_e *MockQueryClient_Expecter) GetProposerByRollapp(ctx interface{}, in interface{}, opts ...interface{}) *MockQueryClient_GetProposerByRollapp_Call {
	return &MockQueryClient_GetProposerByRollapp_Call{Call: _e.mock.On("GetProposerByRollapp",
		append([]interface{}{ctx, in}, opts...)...)}
}

func (_c *MockQueryClient_GetProposerByRollapp_Call) Run(run func(ctx context.Context, in *types.QueryGetProposerByRollappRequest, opts ...grpc.CallOption)) *MockQueryClient_GetProposerByRollapp_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]grpc.CallOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(grpc.CallOption)
			}
		}
		run(args[0].(context.Context), args[1].(*types.QueryGetProposerByRollappRequest), variadicArgs...)
	})
	return _c
}

func (_c *MockQueryClient_GetProposerByRollapp_Call) Return(_a0 *types.QueryGetProposerByRollappResponse, _a1 error) *MockQueryClient_GetProposerByRollapp_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockQueryClient_GetProposerByRollapp_Call) RunAndReturn(run func(context.Context, *types.QueryGetProposerByRollappRequest, ...grpc.CallOption) (*types.QueryGetProposerByRollappResponse, error)) *MockQueryClient_GetProposerByRollapp_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockQueryClient) Params(ctx context.Context, in *types.QueryParamsRequest, opts ...grpc.CallOption) (*types.QueryParamsResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Params")
	}

	var r0 *types.QueryParamsResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *types.QueryParamsRequest, ...grpc.CallOption) (*types.QueryParamsResponse, error)); ok {
		return rf(ctx, in, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *types.QueryParamsRequest, ...grpc.CallOption) *types.QueryParamsResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.QueryParamsResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *types.QueryParamsRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type MockQueryClient_Params_Call struct {
	*mock.Call
}

func (_e *MockQueryClient_Expecter) Params(ctx interface{}, in interface{}, opts ...interface{}) *MockQueryClient_Params_Call {
	return &MockQueryClient_Params_Call{Call: _e.mock.On("Params",
		append([]interface{}{ctx, in}, opts...)...)}
}

func (_c *MockQueryClient_Params_Call) Run(run func(ctx context.Context, in *types.QueryParamsRequest, opts ...grpc.CallOption)) *MockQueryClient_Params_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]grpc.CallOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(grpc.CallOption)
			}
		}
		run(args[0].(context.Context), args[1].(*types.QueryParamsRequest), variadicArgs...)
	})
	return _c
}

func (_c *MockQueryClient_Params_Call) Return(_a0 *types.QueryParamsResponse, _a1 error) *MockQueryClient_Params_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockQueryClient_Params_Call) RunAndReturn(run func(context.Context, *types.QueryParamsRequest, ...grpc.CallOption) (*types.QueryParamsResponse, error)) *MockQueryClient_Params_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockQueryClient) Sequencer(ctx context.Context, in *types.QueryGetSequencerRequest, opts ...grpc.CallOption) (*types.QueryGetSequencerResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Sequencer")
	}

	var r0 *types.QueryGetSequencerResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *types.QueryGetSequencerRequest, ...grpc.CallOption) (*types.QueryGetSequencerResponse, error)); ok {
		return rf(ctx, in, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *types.QueryGetSequencerRequest, ...grpc.CallOption) *types.QueryGetSequencerResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.QueryGetSequencerResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *types.QueryGetSequencerRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type MockQueryClient_Sequencer_Call struct {
	*mock.Call
}

func (_e *MockQueryClient_Expecter) Sequencer(ctx interface{}, in interface{}, opts ...interface{}) *MockQueryClient_Sequencer_Call {
	return &MockQueryClient_Sequencer_Call{Call: _e.mock.On("Sequencer",
		append([]interface{}{ctx, in}, opts...)...)}
}

func (_c *MockQueryClient_Sequencer_Call) Run(run func(ctx context.Context, in *types.QueryGetSequencerRequest, opts ...grpc.CallOption)) *MockQueryClient_Sequencer_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]grpc.CallOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(grpc.CallOption)
			}
		}
		run(args[0].(context.Context), args[1].(*types.QueryGetSequencerRequest), variadicArgs...)
	})
	return _c
}

func (_c *MockQueryClient_Sequencer_Call) Return(_a0 *types.QueryGetSequencerResponse, _a1 error) *MockQueryClient_Sequencer_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockQueryClient_Sequencer_Call) RunAndReturn(run func(context.Context, *types.QueryGetSequencerRequest, ...grpc.CallOption) (*types.QueryGetSequencerResponse, error)) *MockQueryClient_Sequencer_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockQueryClient) Sequencers(ctx context.Context, in *types.QuerySequencersRequest, opts ...grpc.CallOption) (*types.QuerySequencersResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for sequencers")
	}

	var r0 *types.QuerySequencersResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *types.QuerySequencersRequest, ...grpc.CallOption) (*types.QuerySequencersResponse, error)); ok {
		return rf(ctx, in, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *types.QuerySequencersRequest, ...grpc.CallOption) *types.QuerySequencersResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.QuerySequencersResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *types.QuerySequencersRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type MockQueryClient_Sequencers_Call struct {
	*mock.Call
}

func (_e *MockQueryClient_Expecter) Sequencers(ctx interface{}, in interface{}, opts ...interface{}) *MockQueryClient_Sequencers_Call {
	return &MockQueryClient_Sequencers_Call{Call: _e.mock.On("sequencers",
		append([]interface{}{ctx, in}, opts...)...)}
}

func (_c *MockQueryClient_Sequencers_Call) Run(run func(ctx context.Context, in *types.QuerySequencersRequest, opts ...grpc.CallOption)) *MockQueryClient_Sequencers_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]grpc.CallOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(grpc.CallOption)
			}
		}
		run(args[0].(context.Context), args[1].(*types.QuerySequencersRequest), variadicArgs...)
	})
	return _c
}

func (_c *MockQueryClient_Sequencers_Call) Return(_a0 *types.QuerySequencersResponse, _a1 error) *MockQueryClient_Sequencers_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockQueryClient_Sequencers_Call) RunAndReturn(run func(context.Context, *types.QuerySequencersRequest, ...grpc.CallOption) (*types.QuerySequencersResponse, error)) *MockQueryClient_Sequencers_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockQueryClient) SequencersByRollapp(ctx context.Context, in *types.QueryGetSequencersByRollappRequest, opts ...grpc.CallOption) (*types.QueryGetSequencersByRollappResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for SequencersByRollapp")
	}

	var r0 *types.QueryGetSequencersByRollappResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *types.QueryGetSequencersByRollappRequest, ...grpc.CallOption) (*types.QueryGetSequencersByRollappResponse, error)); ok {
		return rf(ctx, in, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *types.QueryGetSequencersByRollappRequest, ...grpc.CallOption) *types.QueryGetSequencersByRollappResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.QueryGetSequencersByRollappResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *types.QueryGetSequencersByRollappRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type MockQueryClient_SequencersByRollapp_Call struct {
	*mock.Call
}

func (_e *MockQueryClient_Expecter) SequencersByRollapp(ctx interface{}, in interface{}, opts ...interface{}) *MockQueryClient_SequencersByRollapp_Call {
	return &MockQueryClient_SequencersByRollapp_Call{Call: _e.mock.On("SequencersByRollapp",
		append([]interface{}{ctx, in}, opts...)...)}
}

func (_c *MockQueryClient_SequencersByRollapp_Call) Run(run func(ctx context.Context, in *types.QueryGetSequencersByRollappRequest, opts ...grpc.CallOption)) *MockQueryClient_SequencersByRollapp_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]grpc.CallOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(grpc.CallOption)
			}
		}
		run(args[0].(context.Context), args[1].(*types.QueryGetSequencersByRollappRequest), variadicArgs...)
	})
	return _c
}

func (_c *MockQueryClient_SequencersByRollapp_Call) Return(_a0 *types.QueryGetSequencersByRollappResponse, _a1 error) *MockQueryClient_SequencersByRollapp_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockQueryClient_SequencersByRollapp_Call) RunAndReturn(run func(context.Context, *types.QueryGetSequencersByRollappRequest, ...grpc.CallOption) (*types.QueryGetSequencersByRollappResponse, error)) *MockQueryClient_SequencersByRollapp_Call {
	_c.Call.Return(run)
	return _c
}

func (_m *MockQueryClient) SequencersByRollappByStatus(ctx context.Context, in *types.QueryGetSequencersByRollappByStatusRequest, opts ...grpc.CallOption) (*types.QueryGetSequencersByRollappByStatusResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for SequencersByRollappByStatus")
	}

	var r0 *types.QueryGetSequencersByRollappByStatusResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *types.QueryGetSequencersByRollappByStatusRequest, ...grpc.CallOption) (*types.QueryGetSequencersByRollappByStatusResponse, error)); ok {
		return rf(ctx, in, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *types.QueryGetSequencersByRollappByStatusRequest, ...grpc.CallOption) *types.QueryGetSequencersByRollappByStatusResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.QueryGetSequencersByRollappByStatusResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *types.QueryGetSequencersByRollappByStatusRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type MockQueryClient_SequencersByRollappByStatus_Call struct {
	*mock.Call
}

func (_e *MockQueryClient_Expecter) SequencersByRollappByStatus(ctx interface{}, in interface{}, opts ...interface{}) *MockQueryClient_SequencersByRollappByStatus_Call {
	return &MockQueryClient_SequencersByRollappByStatus_Call{Call: _e.mock.On("SequencersByRollappByStatus",
		append([]interface{}{ctx, in}, opts...)...)}
}

func (_c *MockQueryClient_SequencersByRollappByStatus_Call) Run(run func(ctx context.Context, in *types.QueryGetSequencersByRollappByStatusRequest, opts ...grpc.CallOption)) *MockQueryClient_SequencersByRollappByStatus_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]grpc.CallOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(grpc.CallOption)
			}
		}
		run(args[0].(context.Context), args[1].(*types.QueryGetSequencersByRollappByStatusRequest), variadicArgs...)
	})
	return _c
}

func (_c *MockQueryClient_SequencersByRollappByStatus_Call) Return(_a0 *types.QueryGetSequencersByRollappByStatusResponse, _a1 error) *MockQueryClient_SequencersByRollappByStatus_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockQueryClient_SequencersByRollappByStatus_Call) RunAndReturn(run func(context.Context, *types.QueryGetSequencersByRollappByStatusRequest, ...grpc.CallOption) (*types.QueryGetSequencersByRollappByStatusResponse, error)) *MockQueryClient_SequencersByRollappByStatus_Call {
	_c.Call.Return(run)
	return _c
}

func NewMockQueryClient(t interface {
	mock.TestingT
	Cleanup(func())
},
) *MockQueryClient {
	mock := &MockQueryClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
