// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build linux

package kprobes

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/elastic/beats/v7/auditbeat/tracing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/sys/unix"
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

func (p *monitorTestSuite) TestNew() {
	if runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		p.T().Skip("skipping on non-amd64/arm64")
		return
	}

	if os.Getuid() != 0 {
		p.T().Skip("skipping as non-root")
		return
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	m, err := New(true)
	p.Require().NoError(err)

	tmpDir, err := os.MkdirTemp("", "kprobe_bench_test")
	p.Require().NoError(err)
	defer os.RemoveAll(tmpDir)

	errChan := make(chan error)
	cancelChan := make(chan struct{})

	targetFile := filepath.Join(tmpDir, "file_kprobes.txt")
	tid := uint32(unix.Gettid())

	expectedEvents := []MonitorEvent{
		{
			Op:   uint32(unix.IN_CREATE),
			Path: targetFile,
			PID:  tid,
		},
		{
			Op:   uint32(unix.IN_MODIFY),
			Path: targetFile,
			PID:  tid,
		},
		{
			Op:   uint32(unix.IN_ATTRIB),
			Path: targetFile,
			PID:  tid,
		},
		{
			Op:   uint32(unix.IN_MODIFY),
			Path: targetFile,
			PID:  tid,
		},
		{
			Op:   uint32(unix.IN_MODIFY),
			Path: targetFile,
			PID:  tid,
		},
		{
			Op:   uint32(unix.IN_MODIFY),
			Path: targetFile,
			PID:  tid,
		},
	}

	var seenEvents []MonitorEvent
	go func() {
		defer close(errChan)
		for {
			select {
			case mErr := <-m.ErrorChannel():
				select {
				case errChan <- mErr:
				case <-cancelChan:
					return
				}
			case e, ok := <-m.EventChannel():
				if !ok {
					select {
					case errChan <- errors.New("closed event channel"):
					case <-cancelChan:
						return
					}
				}
				seenEvents = append(seenEvents, e)
				continue
			case <-cancelChan:
				return
			}
		}
	}()

	p.Require().NoError(m.Start())
	p.Require().NoError(m.Add(tmpDir))

	p.Require().NoError(os.WriteFile(targetFile, []byte("hello world!"), 0o644))
	p.Require().NoError(os.Chmod(targetFile, 0o777))
	p.Require().NoError(os.WriteFile(targetFile, []byte("data"), 0o644))
	p.Require().NoError(os.Truncate(targetFile, 0))

	time.Sleep(5 * time.Second)
	close(cancelChan)
	err = <-errChan
	if err != nil {
		p.Require().Fail(err.Error())
	}

	p.Require().Equal(expectedEvents, seenEvents)
}

const kernelURL string = "https://cdn.kernel.org/pub/linux/kernel/v6.x/linux-6.6.7.tar.xz"

func downloadKernel(filepath string) error {
	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	req, err := http.NewRequestWithContext(context.TODO(), http.MethodGet, kernelURL, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

func BenchmarkMonitor(b *testing.B) {
	if runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		b.Skip("skipping on non-amd64/arm64")
		return
	}

	if os.Getuid() != 0 {
		b.Skip("skipping as non-root")
		return
	}

	tmpDir, err := os.MkdirTemp("", "kprobe_bench_test")
	require.NoError(b, err)
	defer os.RemoveAll(tmpDir)

	tarFilePath := filepath.Join(tmpDir, "linux-6.6.7.tar.xz")

	m, err := New(true)
	require.NoError(b, err)

	errChan := make(chan error)
	cancelChan := make(chan struct{})

	seenEvents := uint64(0)
	go func() {
		defer close(errChan)
		for {
			select {
			case mErr := <-m.ErrorChannel():
				select {
				case errChan <- mErr:
				case <-cancelChan:
					return
				}
			case <-m.EventChannel():
				seenEvents += 1
				continue
			case <-cancelChan:
				return
			}
		}
	}()

	require.NoError(b, m.Start())
	require.NoError(b, m.Add(tmpDir))

	err = downloadKernel(tarFilePath)

	// decompress
	require.NoError(b, err)
	cmd := exec.Command("tar", "-xvf", "./linux-6.6.7.tar.xz")
	cmd.Dir = tmpDir
	err = cmd.Run()
	require.NoError(b, err)

	// re-decompress; causes deletions of previous files
	cmd = exec.Command("tar", "-xvf", "./linux-6.6.7.tar.xz")
	cmd.Dir = tmpDir
	err = cmd.Run()
	require.NoError(b, err)

	time.Sleep(2 * time.Second)
	close(cancelChan)
	err = <-errChan
	if err != nil {
		require.Fail(b, err.Error())
	}

	require.NoError(b, m.Close())

	// decompressing linux-6.6.7.tar.xz created 87082 files (includes created folder); measured with decompressing and
	//   running "find . | wc -l"
	// so the dcache entry should contain 1 (tmpDir) + 1 (linux-6.6.7.tar.xz archive)
	//   + 87082 (folder + archive contents) dentries
	require.Len(b, m.eProc.d.index, 87082+2)

	b.Logf("processed %d events", seenEvents)
}
