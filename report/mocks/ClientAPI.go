// Code generated by mockery v2.36.0. DO NOT EDIT.

package mocks

import (
	api "github.com/bitrise-steplib/steps-deploy-to-bitrise-io/report/api"
	mock "github.com/stretchr/testify/mock"
)

// ClientAPI is an autogenerated mock type for the ClientAPI type
type ClientAPI struct {
	mock.Mock
}

// CreateReport provides a mock function with given fields: params
func (_m *ClientAPI) CreateReport(params api.CreateReportParameters) (api.CreateReportResponse, error) {
	ret := _m.Called(params)

	var r0 api.CreateReportResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(api.CreateReportParameters) (api.CreateReportResponse, error)); ok {
		return rf(params)
	}
	if rf, ok := ret.Get(0).(func(api.CreateReportParameters) api.CreateReportResponse); ok {
		r0 = rf(params)
	} else {
		r0 = ret.Get(0).(api.CreateReportResponse)
	}

	if rf, ok := ret.Get(1).(func(api.CreateReportParameters) error); ok {
		r1 = rf(params)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FinishReport provides a mock function with given fields: identifier, allAssetsUploaded
func (_m *ClientAPI) FinishReport(identifier string, allAssetsUploaded bool) error {
	ret := _m.Called(identifier, allAssetsUploaded)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, bool) error); ok {
		r0 = rf(identifier, allAssetsUploaded)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UploadAsset provides a mock function with given fields: url, path, contentType
func (_m *ClientAPI) UploadAsset(url string, path string, contentType string) error {
	ret := _m.Called(url, path, contentType)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, string) error); ok {
		r0 = rf(url, path, contentType)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewClientAPI creates a new instance of ClientAPI. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewClientAPI(t interface {
	mock.TestingT
	Cleanup(func())
}) *ClientAPI {
	mock := &ClientAPI{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
