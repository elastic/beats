package kprobes

import "github.com/stretchr/testify/mock"

type perfChannelMock struct {
	mock.Mock
}

func (p *perfChannelMock) C() <-chan interface{} {
	args := p.Called()
	return args.Get(0).(chan interface{})
}

func (p *perfChannelMock) ErrC() <-chan error {
	args := p.Called()
	return args.Get(0).(chan error)
}

func (p *perfChannelMock) LostC() <-chan uint64 {
	args := p.Called()
	return args.Get(0).(chan uint64)
}

func (p *perfChannelMock) Run() error {
	args := p.Called()
	return args.Error(0)
}

func (p *perfChannelMock) Close() error {
	args := p.Called()
	return args.Error(0)
}
