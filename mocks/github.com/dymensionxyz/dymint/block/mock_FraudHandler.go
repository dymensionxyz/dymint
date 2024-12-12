

package block

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)


type MockFraudHandler struct {
	mock.Mock
}

type MockFraudHandler_Expecter struct {
	mock *mock.Mock
}

func (_m *MockFraudHandler) EXPECT() *MockFraudHandler_Expecter {
	return &MockFraudHandler_Expecter{mock: &_m.Mock}
}


func (_m *MockFraudHandler) HandleFault(ctx context.Context, fault error) {
	_m.Called(ctx, fault)
}


type MockFraudHandler_HandleFault_Call struct {
	*mock.Call
}




func (_e *MockFraudHandler_Expecter) HandleFault(ctx interface{}, fault interface{}) *MockFraudHandler_HandleFault_Call {
	return &MockFraudHandler_HandleFault_Call{Call: _e.mock.On("HandleFault", ctx, fault)}
}

func (_c *MockFraudHandler_HandleFault_Call) Run(run func(ctx context.Context, fault error)) *MockFraudHandler_HandleFault_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(error))
	})
	return _c
}

func (_c *MockFraudHandler_HandleFault_Call) Return() *MockFraudHandler_HandleFault_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockFraudHandler_HandleFault_Call) RunAndReturn(run func(context.Context, error)) *MockFraudHandler_HandleFault_Call {
	_c.Call.Return(run)
	return _c
}



func NewMockFraudHandler(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockFraudHandler {
	mock := &MockFraudHandler{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
