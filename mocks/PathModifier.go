// Code generated by mockery v2.33.2. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"

// PathModifier is an autogenerated mock type for the PathModifier type
type PathModifier struct {
	mock.Mock
}

// AbsPath provides a mock function with given fields: pth
func (_m *PathModifier) AbsPath(pth string) (string, error) {
	ret := _m.Called(pth)

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (string, error)); ok {
		return rf(pth)
	}
	if rf, ok := ret.Get(0).(func(string) string); ok {
		r0 = rf(pth)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(pth)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewPathModifier creates a new instance of PathModifier. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewPathModifier(t interface {
	mock.TestingT
	Cleanup(func())
}) *PathModifier {
	mock := &PathModifier{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}