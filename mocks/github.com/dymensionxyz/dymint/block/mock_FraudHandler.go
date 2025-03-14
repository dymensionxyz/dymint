// Code generated by mockery v2.50.2. DO NOT EDIT.

package block

import mock "github.com/stretchr/testify/mock"

// MockFraudHandler is an autogenerated mock type for the FraudHandler type
type MockFraudHandler struct {
	mock.Mock
}

type MockFraudHandler_Expecter struct {
	mock *mock.Mock
}

func (_m *MockFraudHandler) EXPECT() *MockFraudHandler_Expecter {
	return &MockFraudHandler_Expecter{mock: &_m.Mock}
}

// HandleFault provides a mock function with given fields: fault
func (_m *MockFraudHandler) HandleFault(fault error) {
	_m.Called(fault)
}

// MockFraudHandler_HandleFault_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'HandleFault'
type MockFraudHandler_HandleFault_Call struct {
	*mock.Call
}

// HandleFault is a helper method to define mock.On call
//   - fault error
func (_e *MockFraudHandler_Expecter) HandleFault(fault interface{}) *MockFraudHandler_HandleFault_Call {
	return &MockFraudHandler_HandleFault_Call{Call: _e.mock.On("HandleFault", fault)}
}

func (_c *MockFraudHandler_HandleFault_Call) Run(run func(fault error)) *MockFraudHandler_HandleFault_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(error))
	})
	return _c
}

func (_c *MockFraudHandler_HandleFault_Call) Return() *MockFraudHandler_HandleFault_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockFraudHandler_HandleFault_Call) RunAndReturn(run func(error)) *MockFraudHandler_HandleFault_Call {
	_c.Run(run)
	return _c
}

// NewMockFraudHandler creates a new instance of MockFraudHandler. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockFraudHandler(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockFraudHandler {
	mock := &MockFraudHandler{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
