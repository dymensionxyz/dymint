// Code generated by mockery v2.42.3. DO NOT EDIT.

package dymension

import (
	context "context"

	client "github.com/cosmos/cosmos-sdk/client"

	coretypes "github.com/tendermint/tendermint/rpc/core/types"

	cosmosaccount "github.com/ignite/cli/ignite/pkg/cosmosaccount"

	cosmosclient "github.com/dymensionxyz/cosmosclient/cosmosclient"

	mock "github.com/stretchr/testify/mock"

	rollapp "github.com/dymensionxyz/dymint/types/pb/dymensionxyz/dymension/rollapp"

	sequencer "github.com/dymensionxyz/dymint/types/pb/dymensionxyz/dymension/sequencer"

	types "github.com/cosmos/cosmos-sdk/types"
)

// MockCosmosClient is an autogenerated mock type for the CosmosClient type
type MockCosmosClient struct {
	mock.Mock
}

type MockCosmosClient_Expecter struct {
	mock *mock.Mock
}

func (_m *MockCosmosClient) EXPECT() *MockCosmosClient_Expecter {
	return &MockCosmosClient_Expecter{mock: &_m.Mock}
}

// BroadcastTx provides a mock function with given fields: accountName, msgs
func (_m *MockCosmosClient) BroadcastTx(accountName string, msgs ...types.Msg) (cosmosclient.Response, error) {
	_va := make([]interface{}, len(msgs))
	for _i := range msgs {
		_va[_i] = msgs[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, accountName)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for BroadcastTx")
	}

	var r0 cosmosclient.Response
	var r1 error
	if rf, ok := ret.Get(0).(func(string, ...types.Msg) (cosmosclient.Response, error)); ok {
		return rf(accountName, msgs...)
	}
	if rf, ok := ret.Get(0).(func(string, ...types.Msg) cosmosclient.Response); ok {
		r0 = rf(accountName, msgs...)
	} else {
		r0 = ret.Get(0).(cosmosclient.Response)
	}

	if rf, ok := ret.Get(1).(func(string, ...types.Msg) error); ok {
		r1 = rf(accountName, msgs...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockCosmosClient_BroadcastTx_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'BroadcastTx'
type MockCosmosClient_BroadcastTx_Call struct {
	*mock.Call
}

// BroadcastTx is a helper method to define mock.On call
//   - accountName string
//   - msgs ...types.Msg
func (_e *MockCosmosClient_Expecter) BroadcastTx(accountName interface{}, msgs ...interface{}) *MockCosmosClient_BroadcastTx_Call {
	return &MockCosmosClient_BroadcastTx_Call{Call: _e.mock.On("BroadcastTx",
		append([]interface{}{accountName}, msgs...)...)}
}

func (_c *MockCosmosClient_BroadcastTx_Call) Run(run func(accountName string, msgs ...types.Msg)) *MockCosmosClient_BroadcastTx_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]types.Msg, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.(types.Msg)
			}
		}
		run(args[0].(string), variadicArgs...)
	})
	return _c
}

func (_c *MockCosmosClient_BroadcastTx_Call) Return(_a0 cosmosclient.Response, _a1 error) *MockCosmosClient_BroadcastTx_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockCosmosClient_BroadcastTx_Call) RunAndReturn(run func(string, ...types.Msg) (cosmosclient.Response, error)) *MockCosmosClient_BroadcastTx_Call {
	_c.Call.Return(run)
	return _c
}

// Context provides a mock function with given fields:
func (_m *MockCosmosClient) Context() client.Context {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Context")
	}

	var r0 client.Context
	if rf, ok := ret.Get(0).(func() client.Context); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(client.Context)
	}

	return r0
}

// MockCosmosClient_Context_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Context'
type MockCosmosClient_Context_Call struct {
	*mock.Call
}

// Context is a helper method to define mock.On call
func (_e *MockCosmosClient_Expecter) Context() *MockCosmosClient_Context_Call {
	return &MockCosmosClient_Context_Call{Call: _e.mock.On("Context")}
}

func (_c *MockCosmosClient_Context_Call) Run(run func()) *MockCosmosClient_Context_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockCosmosClient_Context_Call) Return(_a0 client.Context) *MockCosmosClient_Context_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockCosmosClient_Context_Call) RunAndReturn(run func() client.Context) *MockCosmosClient_Context_Call {
	_c.Call.Return(run)
	return _c
}

// EventListenerQuit provides a mock function with given fields:
func (_m *MockCosmosClient) EventListenerQuit() <-chan struct{} {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for EventListenerQuit")
	}

	var r0 <-chan struct{}
	if rf, ok := ret.Get(0).(func() <-chan struct{}); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan struct{})
		}
	}

	return r0
}

// MockCosmosClient_EventListenerQuit_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'EventListenerQuit'
type MockCosmosClient_EventListenerQuit_Call struct {
	*mock.Call
}

// EventListenerQuit is a helper method to define mock.On call
func (_e *MockCosmosClient_Expecter) EventListenerQuit() *MockCosmosClient_EventListenerQuit_Call {
	return &MockCosmosClient_EventListenerQuit_Call{Call: _e.mock.On("EventListenerQuit")}
}

func (_c *MockCosmosClient_EventListenerQuit_Call) Run(run func()) *MockCosmosClient_EventListenerQuit_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockCosmosClient_EventListenerQuit_Call) Return(_a0 <-chan struct{}) *MockCosmosClient_EventListenerQuit_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockCosmosClient_EventListenerQuit_Call) RunAndReturn(run func() <-chan struct{}) *MockCosmosClient_EventListenerQuit_Call {
	_c.Call.Return(run)
	return _c
}

// GetAccount provides a mock function with given fields: accountName
func (_m *MockCosmosClient) GetAccount(accountName string) (cosmosaccount.Account, error) {
	ret := _m.Called(accountName)

	if len(ret) == 0 {
		panic("no return value specified for GetAccount")
	}

	var r0 cosmosaccount.Account
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (cosmosaccount.Account, error)); ok {
		return rf(accountName)
	}
	if rf, ok := ret.Get(0).(func(string) cosmosaccount.Account); ok {
		r0 = rf(accountName)
	} else {
		r0 = ret.Get(0).(cosmosaccount.Account)
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(accountName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockCosmosClient_GetAccount_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetAccount'
type MockCosmosClient_GetAccount_Call struct {
	*mock.Call
}

// GetAccount is a helper method to define mock.On call
//   - accountName string
func (_e *MockCosmosClient_Expecter) GetAccount(accountName interface{}) *MockCosmosClient_GetAccount_Call {
	return &MockCosmosClient_GetAccount_Call{Call: _e.mock.On("GetAccount", accountName)}
}

func (_c *MockCosmosClient_GetAccount_Call) Run(run func(accountName string)) *MockCosmosClient_GetAccount_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *MockCosmosClient_GetAccount_Call) Return(_a0 cosmosaccount.Account, _a1 error) *MockCosmosClient_GetAccount_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockCosmosClient_GetAccount_Call) RunAndReturn(run func(string) (cosmosaccount.Account, error)) *MockCosmosClient_GetAccount_Call {
	_c.Call.Return(run)
	return _c
}

// GetBalance provides a mock function with given fields: ctx, accountName, denom
func (_m *MockCosmosClient) GetBalance(ctx context.Context, accountName string, denom string) (*types.Coin, error) {
	ret := _m.Called(ctx, accountName, denom)

	if len(ret) == 0 {
		panic("no return value specified for GetBalance")
	}

	var r0 *types.Coin
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) (*types.Coin, error)); ok {
		return rf(ctx, accountName, denom)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string) *types.Coin); ok {
		r0 = rf(ctx, accountName, denom)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Coin)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, accountName, denom)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockCosmosClient_GetBalance_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetBalance'
type MockCosmosClient_GetBalance_Call struct {
	*mock.Call
}

// GetBalance is a helper method to define mock.On call
//   - ctx context.Context
//   - accountName string
//   - denom string
func (_e *MockCosmosClient_Expecter) GetBalance(ctx interface{}, accountName interface{}, denom interface{}) *MockCosmosClient_GetBalance_Call {
	return &MockCosmosClient_GetBalance_Call{Call: _e.mock.On("GetBalance", ctx, accountName, denom)}
}

func (_c *MockCosmosClient_GetBalance_Call) Run(run func(ctx context.Context, accountName string, denom string)) *MockCosmosClient_GetBalance_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string))
	})
	return _c
}

func (_c *MockCosmosClient_GetBalance_Call) Return(_a0 *types.Coin, _a1 error) *MockCosmosClient_GetBalance_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockCosmosClient_GetBalance_Call) RunAndReturn(run func(context.Context, string, string) (*types.Coin, error)) *MockCosmosClient_GetBalance_Call {
	_c.Call.Return(run)
	return _c
}

// GetRollappClient provides a mock function with given fields:
func (_m *MockCosmosClient) GetRollappClient() rollapp.QueryClient {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetRollappClient")
	}

	var r0 rollapp.QueryClient
	if rf, ok := ret.Get(0).(func() rollapp.QueryClient); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(rollapp.QueryClient)
		}
	}

	return r0
}

// MockCosmosClient_GetRollappClient_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetRollappClient'
type MockCosmosClient_GetRollappClient_Call struct {
	*mock.Call
}

// GetRollappClient is a helper method to define mock.On call
func (_e *MockCosmosClient_Expecter) GetRollappClient() *MockCosmosClient_GetRollappClient_Call {
	return &MockCosmosClient_GetRollappClient_Call{Call: _e.mock.On("GetRollappClient")}
}

func (_c *MockCosmosClient_GetRollappClient_Call) Run(run func()) *MockCosmosClient_GetRollappClient_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockCosmosClient_GetRollappClient_Call) Return(_a0 rollapp.QueryClient) *MockCosmosClient_GetRollappClient_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockCosmosClient_GetRollappClient_Call) RunAndReturn(run func() rollapp.QueryClient) *MockCosmosClient_GetRollappClient_Call {
	_c.Call.Return(run)
	return _c
}

// GetSequencerClient provides a mock function with given fields:
func (_m *MockCosmosClient) GetSequencerClient() sequencer.QueryClient {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetSequencerClient")
	}

	var r0 sequencer.QueryClient
	if rf, ok := ret.Get(0).(func() sequencer.QueryClient); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(sequencer.QueryClient)
		}
	}

	return r0
}

// MockCosmosClient_GetSequencerClient_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetSequencerClient'
type MockCosmosClient_GetSequencerClient_Call struct {
	*mock.Call
}

// GetSequencerClient is a helper method to define mock.On call
func (_e *MockCosmosClient_Expecter) GetSequencerClient() *MockCosmosClient_GetSequencerClient_Call {
	return &MockCosmosClient_GetSequencerClient_Call{Call: _e.mock.On("GetSequencerClient")}
}

func (_c *MockCosmosClient_GetSequencerClient_Call) Run(run func()) *MockCosmosClient_GetSequencerClient_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockCosmosClient_GetSequencerClient_Call) Return(_a0 sequencer.QueryClient) *MockCosmosClient_GetSequencerClient_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockCosmosClient_GetSequencerClient_Call) RunAndReturn(run func() sequencer.QueryClient) *MockCosmosClient_GetSequencerClient_Call {
	_c.Call.Return(run)
	return _c
}

// StartEventListener provides a mock function with given fields:
func (_m *MockCosmosClient) StartEventListener() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for StartEventListener")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockCosmosClient_StartEventListener_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'StartEventListener'
type MockCosmosClient_StartEventListener_Call struct {
	*mock.Call
}

// StartEventListener is a helper method to define mock.On call
func (_e *MockCosmosClient_Expecter) StartEventListener() *MockCosmosClient_StartEventListener_Call {
	return &MockCosmosClient_StartEventListener_Call{Call: _e.mock.On("StartEventListener")}
}

func (_c *MockCosmosClient_StartEventListener_Call) Run(run func()) *MockCosmosClient_StartEventListener_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockCosmosClient_StartEventListener_Call) Return(_a0 error) *MockCosmosClient_StartEventListener_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockCosmosClient_StartEventListener_Call) RunAndReturn(run func() error) *MockCosmosClient_StartEventListener_Call {
	_c.Call.Return(run)
	return _c
}

// StopEventListener provides a mock function with given fields:
func (_m *MockCosmosClient) StopEventListener() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for StopEventListener")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockCosmosClient_StopEventListener_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'StopEventListener'
type MockCosmosClient_StopEventListener_Call struct {
	*mock.Call
}

// StopEventListener is a helper method to define mock.On call
func (_e *MockCosmosClient_Expecter) StopEventListener() *MockCosmosClient_StopEventListener_Call {
	return &MockCosmosClient_StopEventListener_Call{Call: _e.mock.On("StopEventListener")}
}

func (_c *MockCosmosClient_StopEventListener_Call) Run(run func()) *MockCosmosClient_StopEventListener_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockCosmosClient_StopEventListener_Call) Return(_a0 error) *MockCosmosClient_StopEventListener_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockCosmosClient_StopEventListener_Call) RunAndReturn(run func() error) *MockCosmosClient_StopEventListener_Call {
	_c.Call.Return(run)
	return _c
}

// SubscribeToEvents provides a mock function with given fields: ctx, subscriber, query, outCapacity
func (_m *MockCosmosClient) SubscribeToEvents(ctx context.Context, subscriber string, query string, outCapacity ...int) (<-chan coretypes.ResultEvent, error) {
	_va := make([]interface{}, len(outCapacity))
	for _i := range outCapacity {
		_va[_i] = outCapacity[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, subscriber, query)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for SubscribeToEvents")
	}

	var r0 <-chan coretypes.ResultEvent
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, ...int) (<-chan coretypes.ResultEvent, error)); ok {
		return rf(ctx, subscriber, query, outCapacity...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string, ...int) <-chan coretypes.ResultEvent); ok {
		r0 = rf(ctx, subscriber, query, outCapacity...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan coretypes.ResultEvent)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string, ...int) error); ok {
		r1 = rf(ctx, subscriber, query, outCapacity...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockCosmosClient_SubscribeToEvents_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SubscribeToEvents'
type MockCosmosClient_SubscribeToEvents_Call struct {
	*mock.Call
}

// SubscribeToEvents is a helper method to define mock.On call
//   - ctx context.Context
//   - subscriber string
//   - query string
//   - outCapacity ...int
func (_e *MockCosmosClient_Expecter) SubscribeToEvents(ctx interface{}, subscriber interface{}, query interface{}, outCapacity ...interface{}) *MockCosmosClient_SubscribeToEvents_Call {
	return &MockCosmosClient_SubscribeToEvents_Call{Call: _e.mock.On("SubscribeToEvents",
		append([]interface{}{ctx, subscriber, query}, outCapacity...)...)}
}

func (_c *MockCosmosClient_SubscribeToEvents_Call) Run(run func(ctx context.Context, subscriber string, query string, outCapacity ...int)) *MockCosmosClient_SubscribeToEvents_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]int, len(args)-3)
		for i, a := range args[3:] {
			if a != nil {
				variadicArgs[i] = a.(int)
			}
		}
		run(args[0].(context.Context), args[1].(string), args[2].(string), variadicArgs...)
	})
	return _c
}

func (_c *MockCosmosClient_SubscribeToEvents_Call) Return(out <-chan coretypes.ResultEvent, err error) *MockCosmosClient_SubscribeToEvents_Call {
	_c.Call.Return(out, err)
	return _c
}

func (_c *MockCosmosClient_SubscribeToEvents_Call) RunAndReturn(run func(context.Context, string, string, ...int) (<-chan coretypes.ResultEvent, error)) *MockCosmosClient_SubscribeToEvents_Call {
	_c.Call.Return(run)
	return _c
}

// UnsubscribeAll provides a mock function with given fields: ctx, subscriber
func (_m *MockCosmosClient) UnsubscribeAll(ctx context.Context, subscriber string) error {
	ret := _m.Called(ctx, subscriber)

	if len(ret) == 0 {
		panic("no return value specified for UnsubscribeAll")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, subscriber)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockCosmosClient_UnsubscribeAll_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UnsubscribeAll'
type MockCosmosClient_UnsubscribeAll_Call struct {
	*mock.Call
}

// UnsubscribeAll is a helper method to define mock.On call
//   - ctx context.Context
//   - subscriber string
func (_e *MockCosmosClient_Expecter) UnsubscribeAll(ctx interface{}, subscriber interface{}) *MockCosmosClient_UnsubscribeAll_Call {
	return &MockCosmosClient_UnsubscribeAll_Call{Call: _e.mock.On("UnsubscribeAll", ctx, subscriber)}
}

func (_c *MockCosmosClient_UnsubscribeAll_Call) Run(run func(ctx context.Context, subscriber string)) *MockCosmosClient_UnsubscribeAll_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *MockCosmosClient_UnsubscribeAll_Call) Return(_a0 error) *MockCosmosClient_UnsubscribeAll_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockCosmosClient_UnsubscribeAll_Call) RunAndReturn(run func(context.Context, string) error) *MockCosmosClient_UnsubscribeAll_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockCosmosClient creates a new instance of MockCosmosClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockCosmosClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockCosmosClient {
	mock := &MockCosmosClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
