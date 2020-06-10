// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import (
	context "context"
	http "net/http"

	diff "go.skia.org/infra/golden/go/diff"

	mock "github.com/stretchr/testify/mock"

	types "go.skia.org/infra/golden/go/types"
)

// DiffStore is an autogenerated mock type for the DiffStore type
type DiffStore struct {
	mock.Mock
}

// Get provides a mock function with given fields: ctx, mainDigest, rightDigests
func (_m *DiffStore) Get(ctx context.Context, mainDigest types.Digest, rightDigests types.DigestSlice) (map[types.Digest]*diff.DiffMetrics, error) {
	ret := _m.Called(ctx, mainDigest, rightDigests)

	var r0 map[types.Digest]*diff.DiffMetrics
	if rf, ok := ret.Get(0).(func(context.Context, types.Digest, types.DigestSlice) map[types.Digest]*diff.DiffMetrics); ok {
		r0 = rf(ctx, mainDigest, rightDigests)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[types.Digest]*diff.DiffMetrics)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, types.Digest, types.DigestSlice) error); ok {
		r1 = rf(ctx, mainDigest, rightDigests)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ImageHandler provides a mock function with given fields: urlPrefix
func (_m *DiffStore) ImageHandler(urlPrefix string) (http.Handler, error) {
	ret := _m.Called(urlPrefix)

	var r0 http.Handler
	if rf, ok := ret.Get(0).(func(string) http.Handler); ok {
		r0 = rf(urlPrefix)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(http.Handler)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(urlPrefix)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// PurgeDigests provides a mock function with given fields: ctx, digests, purgeGCS
func (_m *DiffStore) PurgeDigests(ctx context.Context, digests types.DigestSlice, purgeGCS bool) error {
	ret := _m.Called(ctx, digests, purgeGCS)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, types.DigestSlice, bool) error); ok {
		r0 = rf(ctx, digests, purgeGCS)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
