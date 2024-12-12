

package sequencer

import (
	context "context"

	grpc "google.golang.org/grpc"

	mock "github.com/stretchr/testify/mock"

	sequencer "github.com/dymensionxyz/dymint/types/pb/dymensionxyz/dymension/sequencer"
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


func (_m *MockQueryClient) GetNextProposerByRollapp(ctx context.Context, in *sequencer.QueryGetNextProposerByRollappRequest, opts ...grpc.CallOption) (*sequencer.QueryGetNextProposerByRollappResponse, error) {
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

	var r0 *sequencer.QueryGetNextProposerByRollappResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *sequencer.QueryGetNextProposerByRollappRequest, ...grpc.CallOption) (*sequencer.QueryGetNextProposerByRollappResponse, error)); ok {
		return rf(ctx, in, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *sequencer.QueryGetNextProposerByRollappRequest, ...grpc.CallOption) *sequencer.QueryGetNextProposerByRollappResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*sequencer.QueryGetNextProposerByRollappResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *sequencer.QueryGetNextProposerByRollappRequest, ...grpc.CallOption) error); ok {
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

func (_c *MockQueryClient_GetNextProposerByRollapp_Call) Run(run func(ctx context.Context, in *sequencer.QueryGetNextProposerByRollappRequest, opts ...grpc.CallOption)) *MockQueryClient_GetNextProposerByRollapp_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]grpc.CallOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(grpc.CallOption)
			}
		}
		run(args[0].(context.Context), args[1].(*sequencer.QueryGetNextProposerByRollappRequest), variadicArgs...)
	})
	return _c
}

func (_c *MockQueryClient_GetNextProposerByRollapp_Call) Return(_a0 *sequencer.QueryGetNextProposerByRollappResponse, _a1 error) *MockQueryClient_GetNextProposerByRollapp_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockQueryClient_GetNextProposerByRollapp_Call) RunAndReturn(run func(context.Context, *sequencer.QueryGetNextProposerByRollappRequest, ...grpc.CallOption) (*sequencer.QueryGetNextProposerByRollappResponse, error)) *MockQueryClient_GetNextProposerByRollapp_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockQueryClient) GetProposerByRollapp(ctx context.Context, in *sequencer.QueryGetProposerByRollappRequest, opts ...grpc.CallOption) (*sequencer.QueryGetProposerByRollappResponse, error) {
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

	var r0 *sequencer.QueryGetProposerByRollappResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *sequencer.QueryGetProposerByRollappRequest, ...grpc.CallOption) (*sequencer.QueryGetProposerByRollappResponse, error)); ok {
		return rf(ctx, in, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *sequencer.QueryGetProposerByRollappRequest, ...grpc.CallOption) *sequencer.QueryGetProposerByRollappResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*sequencer.QueryGetProposerByRollappResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *sequencer.QueryGetProposerByRollappRequest, ...grpc.CallOption) error); ok {
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

func (_c *MockQueryClient_GetProposerByRollapp_Call) Run(run func(ctx context.Context, in *sequencer.QueryGetProposerByRollappRequest, opts ...grpc.CallOption)) *MockQueryClient_GetProposerByRollapp_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]grpc.CallOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(grpc.CallOption)
			}
		}
		run(args[0].(context.Context), args[1].(*sequencer.QueryGetProposerByRollappRequest), variadicArgs...)
	})
	return _c
}

func (_c *MockQueryClient_GetProposerByRollapp_Call) Return(_a0 *sequencer.QueryGetProposerByRollappResponse, _a1 error) *MockQueryClient_GetProposerByRollapp_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockQueryClient_GetProposerByRollapp_Call) RunAndReturn(run func(context.Context, *sequencer.QueryGetProposerByRollappRequest, ...grpc.CallOption) (*sequencer.QueryGetProposerByRollappResponse, error)) *MockQueryClient_GetProposerByRollapp_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockQueryClient) Params(ctx context.Context, in *sequencer.QueryParamsRequest, opts ...grpc.CallOption) (*sequencer.QueryParamsResponse, error) {
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

	var r0 *sequencer.QueryParamsResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *sequencer.QueryParamsRequest, ...grpc.CallOption) (*sequencer.QueryParamsResponse, error)); ok {
		return rf(ctx, in, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *sequencer.QueryParamsRequest, ...grpc.CallOption) *sequencer.QueryParamsResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*sequencer.QueryParamsResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *sequencer.QueryParamsRequest, ...grpc.CallOption) error); ok {
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

func (_c *MockQueryClient_Params_Call) Run(run func(ctx context.Context, in *sequencer.QueryParamsRequest, opts ...grpc.CallOption)) *MockQueryClient_Params_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]grpc.CallOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(grpc.CallOption)
			}
		}
		run(args[0].(context.Context), args[1].(*sequencer.QueryParamsRequest), variadicArgs...)
	})
	return _c
}

func (_c *MockQueryClient_Params_Call) Return(_a0 *sequencer.QueryParamsResponse, _a1 error) *MockQueryClient_Params_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockQueryClient_Params_Call) RunAndReturn(run func(context.Context, *sequencer.QueryParamsRequest, ...grpc.CallOption) (*sequencer.QueryParamsResponse, error)) *MockQueryClient_Params_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockQueryClient) Proposers(ctx context.Context, in *sequencer.QueryProposersRequest, opts ...grpc.CallOption) (*sequencer.QueryProposersResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Proposers")
	}

	var r0 *sequencer.QueryProposersResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *sequencer.QueryProposersRequest, ...grpc.CallOption) (*sequencer.QueryProposersResponse, error)); ok {
		return rf(ctx, in, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *sequencer.QueryProposersRequest, ...grpc.CallOption) *sequencer.QueryProposersResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*sequencer.QueryProposersResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *sequencer.QueryProposersRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockQueryClient_Proposers_Call struct {
	*mock.Call
}





func (_e *MockQueryClient_Expecter) Proposers(ctx interface{}, in interface{}, opts ...interface{}) *MockQueryClient_Proposers_Call {
	return &MockQueryClient_Proposers_Call{Call: _e.mock.On("Proposers",
		append([]interface{}{ctx, in}, opts...)...)}
}

func (_c *MockQueryClient_Proposers_Call) Run(run func(ctx context.Context, in *sequencer.QueryProposersRequest, opts ...grpc.CallOption)) *MockQueryClient_Proposers_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]grpc.CallOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(grpc.CallOption)
			}
		}
		run(args[0].(context.Context), args[1].(*sequencer.QueryProposersRequest), variadicArgs...)
	})
	return _c
}

func (_c *MockQueryClient_Proposers_Call) Return(_a0 *sequencer.QueryProposersResponse, _a1 error) *MockQueryClient_Proposers_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockQueryClient_Proposers_Call) RunAndReturn(run func(context.Context, *sequencer.QueryProposersRequest, ...grpc.CallOption) (*sequencer.QueryProposersResponse, error)) *MockQueryClient_Proposers_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockQueryClient) Sequencer(ctx context.Context, in *sequencer.QueryGetSequencerRequest, opts ...grpc.CallOption) (*sequencer.QueryGetSequencerResponse, error) {
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

	var r0 *sequencer.QueryGetSequencerResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *sequencer.QueryGetSequencerRequest, ...grpc.CallOption) (*sequencer.QueryGetSequencerResponse, error)); ok {
		return rf(ctx, in, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *sequencer.QueryGetSequencerRequest, ...grpc.CallOption) *sequencer.QueryGetSequencerResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*sequencer.QueryGetSequencerResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *sequencer.QueryGetSequencerRequest, ...grpc.CallOption) error); ok {
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

func (_c *MockQueryClient_Sequencer_Call) Run(run func(ctx context.Context, in *sequencer.QueryGetSequencerRequest, opts ...grpc.CallOption)) *MockQueryClient_Sequencer_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]grpc.CallOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(grpc.CallOption)
			}
		}
		run(args[0].(context.Context), args[1].(*sequencer.QueryGetSequencerRequest), variadicArgs...)
	})
	return _c
}

func (_c *MockQueryClient_Sequencer_Call) Return(_a0 *sequencer.QueryGetSequencerResponse, _a1 error) *MockQueryClient_Sequencer_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockQueryClient_Sequencer_Call) RunAndReturn(run func(context.Context, *sequencer.QueryGetSequencerRequest, ...grpc.CallOption) (*sequencer.QueryGetSequencerResponse, error)) *MockQueryClient_Sequencer_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockQueryClient) Sequencers(ctx context.Context, in *sequencer.QuerySequencersRequest, opts ...grpc.CallOption) (*sequencer.QuerySequencersResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Sequencers")
	}

	var r0 *sequencer.QuerySequencersResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *sequencer.QuerySequencersRequest, ...grpc.CallOption) (*sequencer.QuerySequencersResponse, error)); ok {
		return rf(ctx, in, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *sequencer.QuerySequencersRequest, ...grpc.CallOption) *sequencer.QuerySequencersResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*sequencer.QuerySequencersResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *sequencer.QuerySequencersRequest, ...grpc.CallOption) error); ok {
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
	return &MockQueryClient_Sequencers_Call{Call: _e.mock.On("Sequencers",
		append([]interface{}{ctx, in}, opts...)...)}
}

func (_c *MockQueryClient_Sequencers_Call) Run(run func(ctx context.Context, in *sequencer.QuerySequencersRequest, opts ...grpc.CallOption)) *MockQueryClient_Sequencers_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]grpc.CallOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(grpc.CallOption)
			}
		}
		run(args[0].(context.Context), args[1].(*sequencer.QuerySequencersRequest), variadicArgs...)
	})
	return _c
}

func (_c *MockQueryClient_Sequencers_Call) Return(_a0 *sequencer.QuerySequencersResponse, _a1 error) *MockQueryClient_Sequencers_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockQueryClient_Sequencers_Call) RunAndReturn(run func(context.Context, *sequencer.QuerySequencersRequest, ...grpc.CallOption) (*sequencer.QuerySequencersResponse, error)) *MockQueryClient_Sequencers_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockQueryClient) SequencersByRollapp(ctx context.Context, in *sequencer.QueryGetSequencersByRollappRequest, opts ...grpc.CallOption) (*sequencer.QueryGetSequencersByRollappResponse, error) {
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

	var r0 *sequencer.QueryGetSequencersByRollappResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *sequencer.QueryGetSequencersByRollappRequest, ...grpc.CallOption) (*sequencer.QueryGetSequencersByRollappResponse, error)); ok {
		return rf(ctx, in, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *sequencer.QueryGetSequencersByRollappRequest, ...grpc.CallOption) *sequencer.QueryGetSequencersByRollappResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*sequencer.QueryGetSequencersByRollappResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *sequencer.QueryGetSequencersByRollappRequest, ...grpc.CallOption) error); ok {
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

func (_c *MockQueryClient_SequencersByRollapp_Call) Run(run func(ctx context.Context, in *sequencer.QueryGetSequencersByRollappRequest, opts ...grpc.CallOption)) *MockQueryClient_SequencersByRollapp_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]grpc.CallOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(grpc.CallOption)
			}
		}
		run(args[0].(context.Context), args[1].(*sequencer.QueryGetSequencersByRollappRequest), variadicArgs...)
	})
	return _c
}

func (_c *MockQueryClient_SequencersByRollapp_Call) Return(_a0 *sequencer.QueryGetSequencersByRollappResponse, _a1 error) *MockQueryClient_SequencersByRollapp_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockQueryClient_SequencersByRollapp_Call) RunAndReturn(run func(context.Context, *sequencer.QueryGetSequencersByRollappRequest, ...grpc.CallOption) (*sequencer.QueryGetSequencersByRollappResponse, error)) *MockQueryClient_SequencersByRollapp_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockQueryClient) SequencersByRollappByStatus(ctx context.Context, in *sequencer.QueryGetSequencersByRollappByStatusRequest, opts ...grpc.CallOption) (*sequencer.QueryGetSequencersByRollappByStatusResponse, error) {
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

	var r0 *sequencer.QueryGetSequencersByRollappByStatusResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *sequencer.QueryGetSequencersByRollappByStatusRequest, ...grpc.CallOption) (*sequencer.QueryGetSequencersByRollappByStatusResponse, error)); ok {
		return rf(ctx, in, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *sequencer.QueryGetSequencersByRollappByStatusRequest, ...grpc.CallOption) *sequencer.QueryGetSequencersByRollappByStatusResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*sequencer.QueryGetSequencersByRollappByStatusResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *sequencer.QueryGetSequencersByRollappByStatusRequest, ...grpc.CallOption) error); ok {
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

func (_c *MockQueryClient_SequencersByRollappByStatus_Call) Run(run func(ctx context.Context, in *sequencer.QueryGetSequencersByRollappByStatusRequest, opts ...grpc.CallOption)) *MockQueryClient_SequencersByRollappByStatus_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]grpc.CallOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(grpc.CallOption)
			}
		}
		run(args[0].(context.Context), args[1].(*sequencer.QueryGetSequencersByRollappByStatusRequest), variadicArgs...)
	})
	return _c
}

func (_c *MockQueryClient_SequencersByRollappByStatus_Call) Return(_a0 *sequencer.QueryGetSequencersByRollappByStatusResponse, _a1 error) *MockQueryClient_SequencersByRollappByStatus_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockQueryClient_SequencersByRollappByStatus_Call) RunAndReturn(run func(context.Context, *sequencer.QueryGetSequencersByRollappByStatusRequest, ...grpc.CallOption) (*sequencer.QueryGetSequencersByRollappByStatusResponse, error)) *MockQueryClient_SequencersByRollappByStatus_Call {
	_c.Call.Return(run)
	return _c
}



func NewMockQueryClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockQueryClient {
	mock := &MockQueryClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
