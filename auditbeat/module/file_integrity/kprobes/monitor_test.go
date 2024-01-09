package kprobes

import (
	"context"
	"errors"
	"github.com/elastic/beats/v7/auditbeat/module/file_integrity/kprobes/tracing"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"golang.org/x/sys/unix"
	"testing"
	"time"
)

type monitorTestSuite struct {
	suite.Suite
}

func Test_Monitor(t *testing.T) {
	suite.Run(t, new(monitorTestSuite))
}

func (p *monitorTestSuite) TestDoubleClose() {
	ctx := context.Background()
	mockPerfChannel := &perfChannelMock{}
	mockPerfChannel.On("Close").Return(nil)
	exec := newFixedThreadExecutor(ctx)
	m, err := newMonitor(ctx, true, mockPerfChannel, exec)
	p.Require().NoError(err)
	err = m.Close()
	p.Require().NoError(err)
	err = m.Close()
	p.Require().NoError(err)
}

func (p *monitorTestSuite) TestPerfChannelClose() {
	ctx := context.Background()
	mockPerfChannel := &perfChannelMock{}
	closeErr := errors.New("error closing perf channel")
	mockPerfChannel.On("Close").Return(closeErr)
	exec := newFixedThreadExecutor(ctx)
	m, err := newMonitor(ctx, true, mockPerfChannel, exec)
	p.Require().NoError(err)
	err = m.Close()
	p.Require().ErrorIs(err, closeErr)
}

func (p *monitorTestSuite) TestPerfChannelRunErr() {
	ctx := context.Background()
	mockPerfChannel := &perfChannelMock{}
	runErr := errors.New("perf channel run err")
	mockPerfChannel.On("Run").Return(runErr)
	mockPerfChannel.On("Close").Return(nil)

	exec := newFixedThreadExecutor(ctx)
	m, err := newMonitor(ctx, true, mockPerfChannel, exec)
	p.Require().NoError(err)

	err = m.Start()
	p.Require().Error(err, runErr)

	p.Require().NoError(m.Close())
}

func (p *monitorTestSuite) TestRunPerfChannelLost() {
	ctx := context.Background()

	perfLost := make(chan uint64)
	perfEvent := make(chan interface{})
	perfErr := make(chan error)

	mockPerfChannel := &perfChannelMock{}
	mockPerfChannel.On("Run").Return(nil)
	mockPerfChannel.On("Close").Return(nil)
	mockPerfChannel.On("C").Return(perfEvent)
	mockPerfChannel.On("ErrC").Return(perfErr)
	mockPerfChannel.On("LostC").Return(perfLost)

	exec := newFixedThreadExecutor(ctx)
	m, err := newMonitor(ctx, true, mockPerfChannel, exec)
	p.Require().NoError(err)

	err = m.Start()
	p.Require().NoError(err)

	select {
	case perfLost <- 10:
	case <-time.After(5 * time.Second):
		p.Fail("timeout at writing perf lost")
	}

	select {
	case err = <-m.ErrorChannel():
		p.Require().Error(err)
	case <-time.After(5 * time.Second):
		p.Fail("no error received")
	}

	p.Require().NoError(m.Close())
}

func (p *monitorTestSuite) TestRunPerfChannelErr() {
	ctx := context.Background()

	perfLost := make(chan uint64)
	perfEvent := make(chan interface{})
	perfErr := make(chan error)

	mockPerfChannel := &perfChannelMock{}
	mockPerfChannel.On("Run").Return(nil)
	mockPerfChannel.On("Close").Return(nil)
	mockPerfChannel.On("C").Return(perfEvent)
	mockPerfChannel.On("ErrC").Return(perfErr)
	mockPerfChannel.On("LostC").Return(perfLost)

	exec := newFixedThreadExecutor(ctx)
	m, err := newMonitor(ctx, true, mockPerfChannel, exec)
	p.Require().NoError(err)

	err = m.Start()
	p.Require().NoError(err)

	runErr := errors.New("perf channel run err")
	select {
	case perfErr <- runErr:
	case <-time.After(5 * time.Second):
		p.Fail("timeout at writing perf err")
	}

	select {
	case err = <-m.ErrorChannel():
		p.Require().ErrorIs(err, runErr)
	case <-time.After(5 * time.Second):
		p.Fail("no error received")
	}

	p.Require().NoError(m.Close())
}

func (p *monitorTestSuite) TestRunPathErr() {
	ctx := context.Background()

	perfLost := make(chan uint64)
	perfEvent := make(chan interface{})
	perfErr := make(chan error)

	mockPerfChannel := &perfChannelMock{}
	mockPerfChannel.On("Run").Return(nil)
	mockPerfChannel.On("Close").Return(nil)
	mockPerfChannel.On("C").Return(perfEvent)
	mockPerfChannel.On("ErrC").Return(perfErr)
	mockPerfChannel.On("LostC").Return(perfLost)

	exec := newFixedThreadExecutor(ctx)
	m, err := newMonitor(ctx, true, mockPerfChannel, exec)
	p.Require().NoError(err)

	err = m.Start()
	p.Require().NoError(err)

	runErr := errors.New("path channel run err")
	select {
	case m.pathMonitor.errC <- runErr:
	case <-time.After(5 * time.Second):
		p.Fail("timeout at writing path err")
	}

	select {
	case err = <-m.ErrorChannel():
		p.Require().ErrorIs(err, runErr)
	case <-time.After(5 * time.Second):
		p.Fail("no error received")
	}

	p.Require().NoError(m.Close())
}

func (p *monitorTestSuite) TestRunUnknownEventType() {
	ctx := context.Background()

	type Unknown struct{}

	perfLost := make(chan uint64)
	perfEvent := make(chan interface{})
	perfErr := make(chan error)

	mockPerfChannel := &perfChannelMock{}
	mockPerfChannel.On("Run").Return(nil)
	mockPerfChannel.On("Close").Return(nil)
	mockPerfChannel.On("C").Return(perfEvent)
	mockPerfChannel.On("ErrC").Return(perfErr)
	mockPerfChannel.On("LostC").Return(perfLost)

	exec := newFixedThreadExecutor(ctx)
	m, err := newMonitor(ctx, true, mockPerfChannel, exec)
	p.Require().NoError(err)

	err = m.Start()
	p.Require().NoError(err)

	select {
	case perfEvent <- &Unknown{}:
	case <-time.After(5 * time.Second):
		p.Fail("timeout at writing perf event")
	}

	select {
	case err = <-m.ErrorChannel():
		p.Require().Error(err)
	case <-time.After(5 * time.Second):
		p.Fail("no error received")
	}

	p.Require().NoError(m.Close())
}

func (p *monitorTestSuite) TestRunPerfCloseEventChan() {
	ctx := context.Background()

	type Unknown struct{}

	perfLost := make(chan uint64)
	perfEvent := make(chan interface{})
	perfErr := make(chan error)

	mockPerfChannel := &perfChannelMock{}
	mockPerfChannel.On("Run").Return(nil)
	mockPerfChannel.On("Close").Return(nil)
	mockPerfChannel.On("C").Return(perfEvent)
	mockPerfChannel.On("ErrC").Return(perfErr)
	mockPerfChannel.On("LostC").Return(perfLost)

	exec := newFixedThreadExecutor(ctx)
	m, err := newMonitor(ctx, true, mockPerfChannel, exec)
	p.Require().NoError(err)

	err = m.Start()
	p.Require().NoError(err)

	close(perfEvent)

	select {
	case err = <-m.ErrorChannel():
		p.Require().Error(err)
	case <-time.After(5 * time.Second):
		p.Fail("no error received")
	}

	p.Require().NoError(m.Close())
}

func (p *monitorTestSuite) TestDoubleStart() {
	ctx := context.Background()

	perfLost := make(chan uint64)
	perfEvent := make(chan interface{})
	perfErr := make(chan error)

	mockPerfChannel := &perfChannelMock{}
	mockPerfChannel.On("Run").Return(nil)
	mockPerfChannel.On("Close").Return(nil)
	mockPerfChannel.On("C").Return(perfEvent)
	mockPerfChannel.On("ErrC").Return(perfErr)
	mockPerfChannel.On("LostC").Return(perfLost)

	exec := newFixedThreadExecutor(ctx)
	m, err := newMonitor(ctx, true, mockPerfChannel, exec)
	p.Require().NoError(err)
	err = m.Start()
	p.Require().NoError(err)
	err = m.Start()
	p.Require().Error(err)
	p.Require().NoError(m.Close())
}

func (p *monitorTestSuite) TestAddPathNotStarted() {
	ctx := context.Background()
	mockPerfChannel := &perfChannelMock{}
	mockPerfChannel.On("Close").Return(nil)
	exec := newFixedThreadExecutor(ctx)
	m, err := newMonitor(ctx, true, mockPerfChannel, exec)
	p.Require().NoError(err)
	err = m.Add("not-exist")
	p.Require().Error(err)

	p.Require().NoError(m.Close())
}

func (p *monitorTestSuite) TestAddPathNotClosed() {
	ctx := context.Background()

	perfLost := make(chan uint64)
	perfEvent := make(chan interface{})
	perfErr := make(chan error)

	mockPerfChannel := &perfChannelMock{}
	mockPerfChannel.On("Run").Return(nil)
	mockPerfChannel.On("Close").Return(nil)
	mockPerfChannel.On("C").Return(perfEvent)
	mockPerfChannel.On("ErrC").Return(perfErr)
	mockPerfChannel.On("LostC").Return(perfLost)

	exec := newFixedThreadExecutor(ctx)
	m, err := newMonitor(ctx, true, mockPerfChannel, exec)
	p.Require().NoError(err)
	err = m.Start()
	p.Require().NoError(err)

	p.Require().NoError(m.Close())

	p.Require().Error(m.Add("not-exist"))
}

func (p *monitorTestSuite) TestRunNoError() {
	ctx := context.Background()

	perfLost := make(chan uint64)
	perfEvent := make(chan interface{})
	perfErr := make(chan error)

	mockPerfChannel := &perfChannelMock{}
	mockPerfChannel.On("Run").Return(nil)
	mockPerfChannel.On("Close").Return(nil)
	mockPerfChannel.On("C").Return(perfEvent)
	mockPerfChannel.On("ErrC").Return(perfErr)
	mockPerfChannel.On("LostC").Return(perfLost)

	exec := newFixedThreadExecutor(ctx)
	m, err := newMonitor(ctx, true, mockPerfChannel, exec)
	p.Require().NoError(err)
	m.eProc.d.Add(&dEntry{
		Parent:   nil,
		Depth:    0,
		Children: nil,
		Name:     "/test/test",
		Ino:      1,
		DevMajor: 1,
		DevMinor: 1,
	}, nil)

	err = m.Start()
	p.Require().NoError(err)

	probeEvent := &ProbeEvent{
		Meta: tracing.Metadata{
			TID: 1,
			PID: 1,
		},
		MaskModify:   1,
		FileIno:      1,
		FileDevMajor: 1,
		FileDevMinor: 1,
		FileName:     "test",
	}

	select {
	case perfEvent <- probeEvent:
	case <-time.After(5 * time.Second):
		p.Fail("timeout on writing event to perf channel")
	}

	select {
	case emittedEvent := <-m.EventChannel():
		p.Require().Equal(uint32(unix.IN_MODIFY), emittedEvent.Op)
		p.Require().Equal("/test/test", emittedEvent.Path)
		p.Require().Equal(uint32(1), emittedEvent.PID)
	case <-time.After(5 * time.Second):
		p.Fail("timeout on waiting event from monitor")
	}

	p.Require().NoError(m.Close())
}

type emitterMock struct {
	mock.Mock
}

func (e *emitterMock) Emit(ePath string, pid uint32, op uint32) error {
	args := e.Called(ePath, pid, op)
	return args.Error(0)
}

func (p *monitorTestSuite) TestRunEmitError() {
	ctx := context.Background()

	perfLost := make(chan uint64)
	perfEvent := make(chan interface{})
	perfErr := make(chan error)

	mockPerfChannel := &perfChannelMock{}
	mockPerfChannel.On("Run").Return(nil)
	mockPerfChannel.On("Close").Return(nil)
	mockPerfChannel.On("C").Return(perfEvent)
	mockPerfChannel.On("ErrC").Return(perfErr)
	mockPerfChannel.On("LostC").Return(perfLost)

	emitErr := errors.New("emit error")
	mockEmitter := &emitterMock{}
	mockEmitter.On("Emit", mock.Anything, mock.Anything, mock.Anything).Return(emitErr)

	exec := newFixedThreadExecutor(ctx)
	m, err := newMonitor(ctx, true, mockPerfChannel, exec)
	p.Require().NoError(err)

	m.eProc.e = mockEmitter
	m.eProc.d.Add(&dEntry{
		Parent:   nil,
		Depth:    0,
		Children: nil,
		Name:     "/test/test",
		Ino:      1,
		DevMajor: 1,
		DevMinor: 1,
	}, nil)

	err = m.Start()
	p.Require().NoError(err)

	probeEvent := &ProbeEvent{
		Meta: tracing.Metadata{
			TID: 1,
			PID: 1,
		},
		MaskModify:   1,
		FileIno:      1,
		FileDevMajor: 1,
		FileDevMinor: 1,
		FileName:     "test",
	}

	select {
	case perfEvent <- probeEvent:
	case <-time.After(5 * time.Second):
		p.Fail("timeout on writing event to perf channel")
	}

	select {
	case err = <-m.ErrorChannel():
		p.Require().ErrorIs(err, emitErr)
	case <-time.After(5 * time.Second):
		p.Fail("timeout on waiting err from monitor")
	}

	p.Require().NoError(m.Close())
}
