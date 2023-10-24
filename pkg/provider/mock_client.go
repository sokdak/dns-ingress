// Code generated by mockery v2.36.0. DO NOT EDIT.

package provider

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// MockClient is an autogenerated mock type for the Client type
type MockClient struct {
	mock.Mock
}

// Create provides a mock function with given fields: ctx, name, zoneId, recordType, records, ttl
func (_m *MockClient) Create(ctx context.Context, name string, zoneId string, recordType string, records []string, ttl int) (*Domain, error) {
	ret := _m.Called(ctx, name, zoneId, recordType, records, ttl)

	var r0 *Domain
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string, []string, int) (*Domain, error)); ok {
		return rf(ctx, name, zoneId, recordType, records, ttl)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string, []string, int) *Domain); ok {
		r0 = rf(ctx, name, zoneId, recordType, records, ttl)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*Domain)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string, string, []string, int) error); ok {
		r1 = rf(ctx, name, zoneId, recordType, records, ttl)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Delete provides a mock function with given fields: ctx, id, zoneId
func (_m *MockClient) Delete(ctx context.Context, id string, zoneId string) error {
	ret := _m.Called(ctx, id, zoneId)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) error); ok {
		r0 = rf(ctx, id, zoneId)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Get provides a mock function with given fields: ctx, id, zoneId
func (_m *MockClient) Get(ctx context.Context, id string, zoneId string) (*Domain, error) {
	ret := _m.Called(ctx, id, zoneId)

	var r0 *Domain
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) (*Domain, error)); ok {
		return rf(ctx, id, zoneId)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string) *Domain); ok {
		r0 = rf(ctx, id, zoneId)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*Domain)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, id, zoneId)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetByName provides a mock function with given fields: ctx, name, zoneId
func (_m *MockClient) GetByName(ctx context.Context, name string, zoneId string) (*Domain, error) {
	ret := _m.Called(ctx, name, zoneId)

	var r0 *Domain
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) (*Domain, error)); ok {
		return rf(ctx, name, zoneId)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string) *Domain); ok {
		r0 = rf(ctx, name, zoneId)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*Domain)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, name, zoneId)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetZone provides a mock function with given fields: ctx, zoneName
func (_m *MockClient) GetZone(ctx context.Context, zoneName string) (*Zone, error) {
	ret := _m.Called(ctx, zoneName)

	var r0 *Zone
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*Zone, error)); ok {
		return rf(ctx, zoneName)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *Zone); ok {
		r0 = rf(ctx, zoneName)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*Zone)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, zoneName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Update provides a mock function with given fields: ctx, id, zoneId, recordType, records, ttl
func (_m *MockClient) Update(ctx context.Context, id string, zoneId string, recordType string, records []string, ttl int) (*Domain, error) {
	ret := _m.Called(ctx, id, zoneId, recordType, records, ttl)

	var r0 *Domain
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string, []string, int) (*Domain, error)); ok {
		return rf(ctx, id, zoneId, recordType, records, ttl)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string, []string, int) *Domain); ok {
		r0 = rf(ctx, id, zoneId, recordType, records, ttl)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*Domain)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string, string, []string, int) error); ok {
		r1 = rf(ctx, id, zoneId, recordType, records, ttl)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewMockClient creates a new instance of MockClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockClient {
	mock := &MockClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
