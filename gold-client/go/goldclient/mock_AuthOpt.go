// Code generated by mockery v1.0.0. DO NOT EDIT.

package goldclient

import mock "github.com/stretchr/testify/mock"

// MockAuthOpt is an autogenerated mock type for the AuthOpt type
type MockAuthOpt struct {
	mock.Mock
}

// GetGoldUploader provides a mock function with given fields:
func (_m *MockAuthOpt) GetGoldUploader() (GoldUploader, error) {
	ret := _m.Called()

	var r0 GoldUploader
	if rf, ok := ret.Get(0).(func() GoldUploader); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(GoldUploader)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetHTTPClient provides a mock function with given fields:
func (_m *MockAuthOpt) GetHTTPClient() (HTTPClient, error) {
	ret := _m.Called()

	var r0 HTTPClient
	if rf, ok := ret.Get(0).(func() HTTPClient); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(HTTPClient)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SetDryRun provides a mock function with given fields: isDryRun
func (_m *MockAuthOpt) SetDryRun(isDryRun bool) {
	_m.Called(isDryRun)
}

// Validate provides a mock function with given fields:
func (_m *MockAuthOpt) Validate() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
