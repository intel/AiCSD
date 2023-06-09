// Code generated by mockery v2.27.1. DO NOT EDIT.

package mocks

import (
	types "aicsd/pkg/types"

	mock "github.com/stretchr/testify/mock"
)

// Client is an autogenerated mock type for the Client type
type Client struct {
	mock.Mock
}

// TransmitFile provides a mock function with given fields: id, entry
func (_m *Client) TransmitFile(id string, entry types.FileInfo) (int, error) {
	ret := _m.Called(id, entry)

	var r0 int
	var r1 error
	if rf, ok := ret.Get(0).(func(string, types.FileInfo) (int, error)); ok {
		return rf(id, entry)
	}
	if rf, ok := ret.Get(0).(func(string, types.FileInfo) int); ok {
		r0 = rf(id, entry)
	} else {
		r0 = ret.Get(0).(int)
	}

	if rf, ok := ret.Get(1).(func(string, types.FileInfo) error); ok {
		r1 = rf(id, entry)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// TransmitJob provides a mock function with given fields: entry
func (_m *Client) TransmitJob(entry types.Job) error {
	ret := _m.Called(entry)

	var r0 error
	if rf, ok := ret.Get(0).(func(types.Job) error); ok {
		r0 = rf(entry)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type mockConstructorTestingTNewClient interface {
	mock.TestingT
	Cleanup(func())
}

// NewClient creates a new instance of Client. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewClient(t mockConstructorTestingTNewClient) *Client {
	mock := &Client{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
