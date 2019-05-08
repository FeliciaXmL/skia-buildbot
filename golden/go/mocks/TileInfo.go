// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"
import tiling "go.skia.org/infra/go/tiling"

// TileInfo is an autogenerated mock type for the TileInfo type
type TileInfo struct {
	mock.Mock
}

// AllCommits provides a mock function with given fields:
func (_m *TileInfo) AllCommits() []*tiling.Commit {
	ret := _m.Called()

	var r0 []*tiling.Commit
	if rf, ok := ret.Get(0).(func() []*tiling.Commit); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*tiling.Commit)
		}
	}

	return r0
}

// DataCommits provides a mock function with given fields:
func (_m *TileInfo) DataCommits() []*tiling.Commit {
	ret := _m.Called()

	var r0 []*tiling.Commit
	if rf, ok := ret.Get(0).(func() []*tiling.Commit); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*tiling.Commit)
		}
	}

	return r0
}

// GetTile provides a mock function with given fields: includeIgnores
func (_m *TileInfo) GetTile(includeIgnores bool) *tiling.Tile {
	ret := _m.Called(includeIgnores)

	var r0 *tiling.Tile
	if rf, ok := ret.Get(0).(func(bool) *tiling.Tile); ok {
		r0 = rf(includeIgnores)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*tiling.Tile)
		}
	}

	return r0
}