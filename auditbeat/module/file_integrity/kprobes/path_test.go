package kprobes

import (
	"context"
	"github.com/stretchr/testify/mock"
)

type pathTraverserMock struct {
	mock.Mock
}

func (p *pathTraverserMock) AddPathToMonitor(ctx context.Context, path string) error {
	args := p.Called(ctx, path)
	return args.Error(0)
}

func (p *pathTraverserMock) GetMonitorPath(ino uint64, major uint32, minor uint32, name string) (MonitorPath, bool) {
	args := p.Called(ino, major, minor, name)
	return args.Get(0).(MonitorPath), args.Bool(1)
}

func (p *pathTraverserMock) WalkAsync(path string, depth uint32, tid uint32) {
	p.Called(path, depth, tid)
}

func (p *pathTraverserMock) ErrC() <-chan error {
	args := p.Called()
	return args.Get(0).(<-chan error)
}

func (p *pathTraverserMock) Close() error {
	args := p.Called()
	return args.Error(0)
}
