

package rollapp

import (
	context "context"

	grpc "google.golang.org/grpc"

	mock "github.com/stretchr/testify/mock"

	rollapp "github.com/dymensionxyz/dymint/types/pb/dymensionxyz/dymension/rollapp"
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


func (_m *MockQueryClient) LatestHeight(ctx context.Context, in *rollapp.QueryGetLatestHeightRequest, opts ...grpc.CallOption) (*rollapp.QueryGetLatestHeightResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for LatestHeight")
	}

	var r0 *rollapp.QueryGetLatestHeightResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *rollapp.QueryGetLatestHeightRequest, ...grpc.CallOption) (*rollapp.QueryGetLatestHeightResponse, error)); ok {
		return rf(ctx, in, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *rollapp.QueryGetLatestHeightRequest, ...grpc.CallOption) *rollapp.QueryGetLatestHeightResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*rollapp.QueryGetLatestHeightResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *rollapp.QueryGetLatestHeightRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockQueryClient_LatestHeight_Call struct {
	*mock.Call
}





func (_e *MockQueryClient_Expecter) LatestHeight(ctx interface{}, in interface{}, opts ...interface{}) *MockQueryClient_LatestHeight_Call {
	return &MockQueryClient_LatestHeight_Call{Call: _e.mock.On("LatestHeight",
		append([]interface{}{ctx, in}, opts...)...)}
}

func (_c *MockQueryClient_LatestHeight_Call) Run(run func(ctx context.Context, in *rollapp.QueryGetLatestHeightRequest, opts ...grpc.CallOption)) *MockQueryClient_LatestHeight_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]grpc.CallOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(grpc.CallOption)
			}
		}
		run(args[0].(context.Context), args[1].(*rollapp.QueryGetLatestHeightRequest), variadicArgs...)
	})
	return _c
}

func (_c *MockQueryClient_LatestHeight_Call) Return(_a0 *rollapp.QueryGetLatestHeightResponse, _a1 error) *MockQueryClient_LatestHeight_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockQueryClient_LatestHeight_Call) RunAndReturn(run func(context.Context, *rollapp.QueryGetLatestHeightRequest, ...grpc.CallOption) (*rollapp.QueryGetLatestHeightResponse, error)) *MockQueryClient_LatestHeight_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockQueryClient) LatestStateIndex(ctx context.Context, in *rollapp.QueryGetLatestStateIndexRequest, opts ...grpc.CallOption) (*rollapp.QueryGetLatestStateIndexResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for LatestStateIndex")
	}

	var r0 *rollapp.QueryGetLatestStateIndexResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *rollapp.QueryGetLatestStateIndexRequest, ...grpc.CallOption) (*rollapp.QueryGetLatestStateIndexResponse, error)); ok {
		return rf(ctx, in, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *rollapp.QueryGetLatestStateIndexRequest, ...grpc.CallOption) *rollapp.QueryGetLatestStateIndexResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*rollapp.QueryGetLatestStateIndexResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *rollapp.QueryGetLatestStateIndexRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockQueryClient_LatestStateIndex_Call struct {
	*mock.Call
}





func (_e *MockQueryClient_Expecter) LatestStateIndex(ctx interface{}, in interface{}, opts ...interface{}) *MockQueryClient_LatestStateIndex_Call {
	return &MockQueryClient_LatestStateIndex_Call{Call: _e.mock.On("LatestStateIndex",
		append([]interface{}{ctx, in}, opts...)...)}
}

func (_c *MockQueryClient_LatestStateIndex_Call) Run(run func(ctx context.Context, in *rollapp.QueryGetLatestStateIndexRequest, opts ...grpc.CallOption)) *MockQueryClient_LatestStateIndex_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]grpc.CallOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(grpc.CallOption)
			}
		}
		run(args[0].(context.Context), args[1].(*rollapp.QueryGetLatestStateIndexRequest), variadicArgs...)
	})
	return _c
}

func (_c *MockQueryClient_LatestStateIndex_Call) Return(_a0 *rollapp.QueryGetLatestStateIndexResponse, _a1 error) *MockQueryClient_LatestStateIndex_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockQueryClient_LatestStateIndex_Call) RunAndReturn(run func(context.Context, *rollapp.QueryGetLatestStateIndexRequest, ...grpc.CallOption) (*rollapp.QueryGetLatestStateIndexResponse, error)) *MockQueryClient_LatestStateIndex_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockQueryClient) ObsoleteDRSVersions(ctx context.Context, in *rollapp.QueryObsoleteDRSVersionsRequest, opts ...grpc.CallOption) (*rollapp.QueryObsoleteDRSVersionsResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for ObsoleteDRSVersions")
	}

	var r0 *rollapp.QueryObsoleteDRSVersionsResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *rollapp.QueryObsoleteDRSVersionsRequest, ...grpc.CallOption) (*rollapp.QueryObsoleteDRSVersionsResponse, error)); ok {
		return rf(ctx, in, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *rollapp.QueryObsoleteDRSVersionsRequest, ...grpc.CallOption) *rollapp.QueryObsoleteDRSVersionsResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*rollapp.QueryObsoleteDRSVersionsResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *rollapp.QueryObsoleteDRSVersionsRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockQueryClient_ObsoleteDRSVersions_Call struct {
	*mock.Call
}





func (_e *MockQueryClient_Expecter) ObsoleteDRSVersions(ctx interface{}, in interface{}, opts ...interface{}) *MockQueryClient_ObsoleteDRSVersions_Call {
	return &MockQueryClient_ObsoleteDRSVersions_Call{Call: _e.mock.On("ObsoleteDRSVersions",
		append([]interface{}{ctx, in}, opts...)...)}
}

func (_c *MockQueryClient_ObsoleteDRSVersions_Call) Run(run func(ctx context.Context, in *rollapp.QueryObsoleteDRSVersionsRequest, opts ...grpc.CallOption)) *MockQueryClient_ObsoleteDRSVersions_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]grpc.CallOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(grpc.CallOption)
			}
		}
		run(args[0].(context.Context), args[1].(*rollapp.QueryObsoleteDRSVersionsRequest), variadicArgs...)
	})
	return _c
}

func (_c *MockQueryClient_ObsoleteDRSVersions_Call) Return(_a0 *rollapp.QueryObsoleteDRSVersionsResponse, _a1 error) *MockQueryClient_ObsoleteDRSVersions_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockQueryClient_ObsoleteDRSVersions_Call) RunAndReturn(run func(context.Context, *rollapp.QueryObsoleteDRSVersionsRequest, ...grpc.CallOption) (*rollapp.QueryObsoleteDRSVersionsResponse, error)) *MockQueryClient_ObsoleteDRSVersions_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockQueryClient) Params(ctx context.Context, in *rollapp.QueryParamsRequest, opts ...grpc.CallOption) (*rollapp.QueryParamsResponse, error) {
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

	var r0 *rollapp.QueryParamsResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *rollapp.QueryParamsRequest, ...grpc.CallOption) (*rollapp.QueryParamsResponse, error)); ok {
		return rf(ctx, in, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *rollapp.QueryParamsRequest, ...grpc.CallOption) *rollapp.QueryParamsResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*rollapp.QueryParamsResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *rollapp.QueryParamsRequest, ...grpc.CallOption) error); ok {
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

func (_c *MockQueryClient_Params_Call) Run(run func(ctx context.Context, in *rollapp.QueryParamsRequest, opts ...grpc.CallOption)) *MockQueryClient_Params_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]grpc.CallOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(grpc.CallOption)
			}
		}
		run(args[0].(context.Context), args[1].(*rollapp.QueryParamsRequest), variadicArgs...)
	})
	return _c
}

func (_c *MockQueryClient_Params_Call) Return(_a0 *rollapp.QueryParamsResponse, _a1 error) *MockQueryClient_Params_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockQueryClient_Params_Call) RunAndReturn(run func(context.Context, *rollapp.QueryParamsRequest, ...grpc.CallOption) (*rollapp.QueryParamsResponse, error)) *MockQueryClient_Params_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockQueryClient) RegisteredDenoms(ctx context.Context, in *rollapp.QueryRegisteredDenomsRequest, opts ...grpc.CallOption) (*rollapp.QueryRegisteredDenomsResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for RegisteredDenoms")
	}

	var r0 *rollapp.QueryRegisteredDenomsResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *rollapp.QueryRegisteredDenomsRequest, ...grpc.CallOption) (*rollapp.QueryRegisteredDenomsResponse, error)); ok {
		return rf(ctx, in, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *rollapp.QueryRegisteredDenomsRequest, ...grpc.CallOption) *rollapp.QueryRegisteredDenomsResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*rollapp.QueryRegisteredDenomsResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *rollapp.QueryRegisteredDenomsRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockQueryClient_RegisteredDenoms_Call struct {
	*mock.Call
}





func (_e *MockQueryClient_Expecter) RegisteredDenoms(ctx interface{}, in interface{}, opts ...interface{}) *MockQueryClient_RegisteredDenoms_Call {
	return &MockQueryClient_RegisteredDenoms_Call{Call: _e.mock.On("RegisteredDenoms",
		append([]interface{}{ctx, in}, opts...)...)}
}

func (_c *MockQueryClient_RegisteredDenoms_Call) Run(run func(ctx context.Context, in *rollapp.QueryRegisteredDenomsRequest, opts ...grpc.CallOption)) *MockQueryClient_RegisteredDenoms_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]grpc.CallOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(grpc.CallOption)
			}
		}
		run(args[0].(context.Context), args[1].(*rollapp.QueryRegisteredDenomsRequest), variadicArgs...)
	})
	return _c
}

func (_c *MockQueryClient_RegisteredDenoms_Call) Return(_a0 *rollapp.QueryRegisteredDenomsResponse, _a1 error) *MockQueryClient_RegisteredDenoms_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockQueryClient_RegisteredDenoms_Call) RunAndReturn(run func(context.Context, *rollapp.QueryRegisteredDenomsRequest, ...grpc.CallOption) (*rollapp.QueryRegisteredDenomsResponse, error)) *MockQueryClient_RegisteredDenoms_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockQueryClient) Rollapp(ctx context.Context, in *rollapp.QueryGetRollappRequest, opts ...grpc.CallOption) (*rollapp.QueryGetRollappResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Rollapp")
	}

	var r0 *rollapp.QueryGetRollappResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *rollapp.QueryGetRollappRequest, ...grpc.CallOption) (*rollapp.QueryGetRollappResponse, error)); ok {
		return rf(ctx, in, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *rollapp.QueryGetRollappRequest, ...grpc.CallOption) *rollapp.QueryGetRollappResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*rollapp.QueryGetRollappResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *rollapp.QueryGetRollappRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockQueryClient_Rollapp_Call struct {
	*mock.Call
}





func (_e *MockQueryClient_Expecter) Rollapp(ctx interface{}, in interface{}, opts ...interface{}) *MockQueryClient_Rollapp_Call {
	return &MockQueryClient_Rollapp_Call{Call: _e.mock.On("Rollapp",
		append([]interface{}{ctx, in}, opts...)...)}
}

func (_c *MockQueryClient_Rollapp_Call) Run(run func(ctx context.Context, in *rollapp.QueryGetRollappRequest, opts ...grpc.CallOption)) *MockQueryClient_Rollapp_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]grpc.CallOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(grpc.CallOption)
			}
		}
		run(args[0].(context.Context), args[1].(*rollapp.QueryGetRollappRequest), variadicArgs...)
	})
	return _c
}

func (_c *MockQueryClient_Rollapp_Call) Return(_a0 *rollapp.QueryGetRollappResponse, _a1 error) *MockQueryClient_Rollapp_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockQueryClient_Rollapp_Call) RunAndReturn(run func(context.Context, *rollapp.QueryGetRollappRequest, ...grpc.CallOption) (*rollapp.QueryGetRollappResponse, error)) *MockQueryClient_Rollapp_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockQueryClient) RollappAll(ctx context.Context, in *rollapp.QueryAllRollappRequest, opts ...grpc.CallOption) (*rollapp.QueryAllRollappResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for RollappAll")
	}

	var r0 *rollapp.QueryAllRollappResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *rollapp.QueryAllRollappRequest, ...grpc.CallOption) (*rollapp.QueryAllRollappResponse, error)); ok {
		return rf(ctx, in, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *rollapp.QueryAllRollappRequest, ...grpc.CallOption) *rollapp.QueryAllRollappResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*rollapp.QueryAllRollappResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *rollapp.QueryAllRollappRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockQueryClient_RollappAll_Call struct {
	*mock.Call
}





func (_e *MockQueryClient_Expecter) RollappAll(ctx interface{}, in interface{}, opts ...interface{}) *MockQueryClient_RollappAll_Call {
	return &MockQueryClient_RollappAll_Call{Call: _e.mock.On("RollappAll",
		append([]interface{}{ctx, in}, opts...)...)}
}

func (_c *MockQueryClient_RollappAll_Call) Run(run func(ctx context.Context, in *rollapp.QueryAllRollappRequest, opts ...grpc.CallOption)) *MockQueryClient_RollappAll_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]grpc.CallOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(grpc.CallOption)
			}
		}
		run(args[0].(context.Context), args[1].(*rollapp.QueryAllRollappRequest), variadicArgs...)
	})
	return _c
}

func (_c *MockQueryClient_RollappAll_Call) Return(_a0 *rollapp.QueryAllRollappResponse, _a1 error) *MockQueryClient_RollappAll_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockQueryClient_RollappAll_Call) RunAndReturn(run func(context.Context, *rollapp.QueryAllRollappRequest, ...grpc.CallOption) (*rollapp.QueryAllRollappResponse, error)) *MockQueryClient_RollappAll_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockQueryClient) RollappByEIP155(ctx context.Context, in *rollapp.QueryGetRollappByEIP155Request, opts ...grpc.CallOption) (*rollapp.QueryGetRollappResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for RollappByEIP155")
	}

	var r0 *rollapp.QueryGetRollappResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *rollapp.QueryGetRollappByEIP155Request, ...grpc.CallOption) (*rollapp.QueryGetRollappResponse, error)); ok {
		return rf(ctx, in, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *rollapp.QueryGetRollappByEIP155Request, ...grpc.CallOption) *rollapp.QueryGetRollappResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*rollapp.QueryGetRollappResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *rollapp.QueryGetRollappByEIP155Request, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockQueryClient_RollappByEIP155_Call struct {
	*mock.Call
}





func (_e *MockQueryClient_Expecter) RollappByEIP155(ctx interface{}, in interface{}, opts ...interface{}) *MockQueryClient_RollappByEIP155_Call {
	return &MockQueryClient_RollappByEIP155_Call{Call: _e.mock.On("RollappByEIP155",
		append([]interface{}{ctx, in}, opts...)...)}
}

func (_c *MockQueryClient_RollappByEIP155_Call) Run(run func(ctx context.Context, in *rollapp.QueryGetRollappByEIP155Request, opts ...grpc.CallOption)) *MockQueryClient_RollappByEIP155_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]grpc.CallOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(grpc.CallOption)
			}
		}
		run(args[0].(context.Context), args[1].(*rollapp.QueryGetRollappByEIP155Request), variadicArgs...)
	})
	return _c
}

func (_c *MockQueryClient_RollappByEIP155_Call) Return(_a0 *rollapp.QueryGetRollappResponse, _a1 error) *MockQueryClient_RollappByEIP155_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockQueryClient_RollappByEIP155_Call) RunAndReturn(run func(context.Context, *rollapp.QueryGetRollappByEIP155Request, ...grpc.CallOption) (*rollapp.QueryGetRollappResponse, error)) *MockQueryClient_RollappByEIP155_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockQueryClient) StateInfo(ctx context.Context, in *rollapp.QueryGetStateInfoRequest, opts ...grpc.CallOption) (*rollapp.QueryGetStateInfoResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for StateInfo")
	}

	var r0 *rollapp.QueryGetStateInfoResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *rollapp.QueryGetStateInfoRequest, ...grpc.CallOption) (*rollapp.QueryGetStateInfoResponse, error)); ok {
		return rf(ctx, in, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *rollapp.QueryGetStateInfoRequest, ...grpc.CallOption) *rollapp.QueryGetStateInfoResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*rollapp.QueryGetStateInfoResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *rollapp.QueryGetStateInfoRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockQueryClient_StateInfo_Call struct {
	*mock.Call
}





func (_e *MockQueryClient_Expecter) StateInfo(ctx interface{}, in interface{}, opts ...interface{}) *MockQueryClient_StateInfo_Call {
	return &MockQueryClient_StateInfo_Call{Call: _e.mock.On("StateInfo",
		append([]interface{}{ctx, in}, opts...)...)}
}

func (_c *MockQueryClient_StateInfo_Call) Run(run func(ctx context.Context, in *rollapp.QueryGetStateInfoRequest, opts ...grpc.CallOption)) *MockQueryClient_StateInfo_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]grpc.CallOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(grpc.CallOption)
			}
		}
		run(args[0].(context.Context), args[1].(*rollapp.QueryGetStateInfoRequest), variadicArgs...)
	})
	return _c
}

func (_c *MockQueryClient_StateInfo_Call) Return(_a0 *rollapp.QueryGetStateInfoResponse, _a1 error) *MockQueryClient_StateInfo_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockQueryClient_StateInfo_Call) RunAndReturn(run func(context.Context, *rollapp.QueryGetStateInfoRequest, ...grpc.CallOption) (*rollapp.QueryGetStateInfoResponse, error)) *MockQueryClient_StateInfo_Call {
	_c.Call.Return(run)
	return _c
}


func (_m *MockQueryClient) ValidateGenesisBridge(ctx context.Context, in *rollapp.QueryValidateGenesisBridgeRequest, opts ...grpc.CallOption) (*rollapp.QueryValidateGenesisBridgeResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for ValidateGenesisBridge")
	}

	var r0 *rollapp.QueryValidateGenesisBridgeResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *rollapp.QueryValidateGenesisBridgeRequest, ...grpc.CallOption) (*rollapp.QueryValidateGenesisBridgeResponse, error)); ok {
		return rf(ctx, in, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *rollapp.QueryValidateGenesisBridgeRequest, ...grpc.CallOption) *rollapp.QueryValidateGenesisBridgeResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*rollapp.QueryValidateGenesisBridgeResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *rollapp.QueryValidateGenesisBridgeRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}


type MockQueryClient_ValidateGenesisBridge_Call struct {
	*mock.Call
}





func (_e *MockQueryClient_Expecter) ValidateGenesisBridge(ctx interface{}, in interface{}, opts ...interface{}) *MockQueryClient_ValidateGenesisBridge_Call {
	return &MockQueryClient_ValidateGenesisBridge_Call{Call: _e.mock.On("ValidateGenesisBridge",
		append([]interface{}{ctx, in}, opts...)...)}
}

func (_c *MockQueryClient_ValidateGenesisBridge_Call) Run(run func(ctx context.Context, in *rollapp.QueryValidateGenesisBridgeRequest, opts ...grpc.CallOption)) *MockQueryClient_ValidateGenesisBridge_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]grpc.CallOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(grpc.CallOption)
			}
		}
		run(args[0].(context.Context), args[1].(*rollapp.QueryValidateGenesisBridgeRequest), variadicArgs...)
	})
	return _c
}

func (_c *MockQueryClient_ValidateGenesisBridge_Call) Return(_a0 *rollapp.QueryValidateGenesisBridgeResponse, _a1 error) *MockQueryClient_ValidateGenesisBridge_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockQueryClient_ValidateGenesisBridge_Call) RunAndReturn(run func(context.Context, *rollapp.QueryValidateGenesisBridgeRequest, ...grpc.CallOption) (*rollapp.QueryValidateGenesisBridgeResponse, error)) *MockQueryClient_ValidateGenesisBridge_Call {
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
