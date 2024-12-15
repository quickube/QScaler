// Code generated by mockery v2.45.1. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// Broker is an autogenerated mock type for the Broker type
type Broker struct {
	mock.Mock
}

// GetDeathQueue provides a mock function with given fields: topic
func (_m *Broker) GetDeathQueue(topic string) string {
	ret := _m.Called(topic)

	if len(ret) == 0 {
		panic("no return value specified for GetDeathQueue")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func(string) string); ok {
		r0 = rf(topic)
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetQueueLength provides a mock function with given fields: ctx, topic
func (_m *Broker) GetQueueLength(ctx *context.Context, topic string) (int, error) {
	ret := _m.Called(ctx, topic)

	if len(ret) == 0 {
		panic("no return value specified for GetQueueLength")
	}

	var r0 int
	var r1 error
	if rf, ok := ret.Get(0).(func(*context.Context, string) (int, error)); ok {
		return rf(ctx, topic)
	}
	if rf, ok := ret.Get(0).(func(*context.Context, string) int); ok {
		r0 = rf(ctx, topic)
	} else {
		r0 = ret.Get(0).(int)
	}

	if rf, ok := ret.Get(1).(func(*context.Context, string) error); ok {
		r1 = rf(ctx, topic)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// IsConnected provides a mock function with given fields: ctx
func (_m *Broker) IsConnected(ctx *context.Context) (bool, error) {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for IsConnected")
	}

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(*context.Context) (bool, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(*context.Context) bool); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(*context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// KillQueue provides a mock function with given fields: ctx, topic
func (_m *Broker) KillQueue(ctx *context.Context, topic string) error {
	ret := _m.Called(ctx, topic)

	if len(ret) == 0 {
		panic("no return value specified for KillQueue")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(*context.Context, string) error); ok {
		r0 = rf(ctx, topic)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewBroker creates a new instance of Broker. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewBroker(t interface {
	mock.TestingT
	Cleanup(func())
}) *Broker {
	mock := &Broker{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}