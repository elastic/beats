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
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func Test_PathTraverser_newPathMonitor(t *testing.T) {
	ctx := context.Background()

	pTrav, err := newPathMonitor(ctx, newFixedThreadExecutor(ctx), 0, true)
	require.NoError(t, err)
	require.Equal(t, pTrav.sMatchTimeout, 5*time.Second)
	require.NoError(t, pTrav.Close())

	pTrav, err = newPathMonitor(ctx, newFixedThreadExecutor(ctx), 2*time.Second, true)
	require.NoError(t, err)
	require.Equal(t, pTrav.sMatchTimeout, 2*time.Second)
	require.NoError(t, pTrav.Close())
}

type pathTestSuite struct {
	suite.Suite
}

func Test_PathTraverser(t *testing.T) {
	suite.Run(t, new(pathTestSuite))
}

func (p *pathTestSuite) TestContextCancelBeforeAdd() {
	// cancelled parent context
	ctx, cancelFn := context.WithCancel(context.Background())
	pTrav, err := newPathMonitor(ctx, newFixedThreadExecutor(ctx), 0, true)
	p.Require().NoError(err)
	cancelFn()
	err = pTrav.AddPathToMonitor(ctx, "not-existing-path")
	p.Require().ErrorIs(err, ctx.Err())
	p.Require().NoError(pTrav.Close())

	// cancelled traverser context
	ctx, cancelFn = context.WithCancel(context.Background())
	pTrav, err = newPathMonitor(ctx, newFixedThreadExecutor(ctx), 0, true)
	p.Require().NoError(err)
	pTrav.cancelFn()
	err = pTrav.AddPathToMonitor(ctx, "not-existing-path")
	p.Require().ErrorIs(err, pTrav.ctx.Err())
	p.Require().NoError(pTrav.Close())
	cancelFn()
}

func (p *pathTestSuite) TestAddParentContextDone() {
	ctx, cancelFn := context.WithCancel(context.Background())
	pTrav, err := newPathMonitor(ctx, newFixedThreadExecutor(ctx), 0, true)
	p.Require().NoError(err)
	cancelFn()
	err = pTrav.AddPathToMonitor(ctx, "not-existing-path")
	p.Require().ErrorIs(err, ctx.Err())
	p.Require().NoError(pTrav.Close())
}

func (p *pathTestSuite) TestRecursiveWalkAsync() {
	var createdPathsOrder []string
	createdPathsWithDepth := make(map[string]uint32)
	tmpDir, err := os.MkdirTemp("", "kprobe_unit_test")
	p.Require().NoError(err)
	defer os.RemoveAll(tmpDir)
	createdPathsWithDepth[tmpDir] = 1
	createdPathsOrder = append(createdPathsOrder, tmpDir)

	testDir := filepath.Join(tmpDir, "test_dir")
	err = os.Mkdir(testDir, 0o744)
	p.Require().NoError(err)
	createdPathsWithDepth[testDir] = 2
	createdPathsOrder = append(createdPathsOrder, testDir)

	testDirTestFile := filepath.Join(tmpDir, "test_dir", "test_file")
	f, err := os.Create(testDirTestFile)
	p.Require().NoError(err)
	p.Require().NoError(f.Close())
	createdPathsWithDepth[testDirTestFile] = 3
	createdPathsOrder = append(createdPathsOrder, testDirTestFile)

	testFile := filepath.Join(tmpDir, "test_file")
	f, err = os.Create(testFile)
	p.Require().NoError(err)
	p.Require().NoError(f.Close())
	createdPathsWithDepth[testFile] = 2
	createdPathsOrder = append(createdPathsOrder, testFile)

	mounts, err := getAllMountPoints()
	p.Require().NoError(err)

	p.Require().Equal(len(createdPathsOrder), len(createdPathsWithDepth))

	expectedStatQueue := make([]statMatch, 0, len(createdPathsOrder))
	for _, path := range createdPathsOrder {

		depth, exists := createdPathsWithDepth[path]
		p.Require().True(exists)

		info, err := os.Lstat(path)
		p.Require().NoError(err)
		mnt := mounts.getMountByPath(path)
		p.Require().NotNil(mnt)
		expectedStatQueue = append(expectedStatQueue, statMatch{
			ino:        info.Sys().(*syscall.Stat_t).Ino,
			major:      mnt.DeviceMajor,
			minor:      mnt.DeviceMinor,
			depth:      depth,
			fileName:   info.Name(),
			isFromMove: true,
			tid:        2,
			fullPath:   path,
		})
	}

	ctx := context.Background()
	pTrav, err := newPathMonitor(ctx, newFixedThreadExecutor(ctx), 0, true)
	p.Require().NoError(err)
	defer func() {
		p.Require().NoError(pTrav.Close())
	}()

	pTrav.WalkAsync(tmpDir, 1, 2)

	tries := 0
	for idx := 0; idx < len(expectedStatQueue); {
		mPath, match := pTrav.GetMonitorPath(
			expectedStatQueue[idx].ino,
			expectedStatQueue[idx].major,
			expectedStatQueue[idx].minor,
			expectedStatQueue[idx].fileName,
		)

		if match {
			p.Require().Equal(expectedStatQueue[idx].fullPath, mPath.fullPath)
			p.Require().Equal(expectedStatQueue[idx].isFromMove, mPath.isFromMove)
			p.Require().Equal(expectedStatQueue[idx].tid, mPath.tid)
			p.Require().Equal(expectedStatQueue[idx].depth, mPath.depth)

			tries = 0
			idx++
			continue
		}

		if tries >= 3 {
			p.Require().Fail("no match found")
			return
		}

		time.Sleep(300 * time.Millisecond)
		tries++
	}

	select {
	case err = <-pTrav.errC:
	default:
	}

	p.Require().NoError(err)
	p.Require().Empty(pTrav.statQueue)
}

func (p *pathTestSuite) TestWalkAsyncTimeoutErr() {
	tmpDir, err := os.MkdirTemp("", "kprobe_unit_test")
	p.Require().NoError(err)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()
	pTrav, err := newPathMonitor(ctx, newFixedThreadExecutor(ctx), 0, true)
	p.Require().NoError(err)
	defer func() {
		p.Require().NoError(pTrav.Close())
	}()

	pTrav.WalkAsync(tmpDir, 1, 2)

	select {
	case err = <-pTrav.errC:
	case <-time.After(10 * time.Second):
		p.Require().Fail("no timeout error received")
	}

	p.Require().ErrorIs(err, ErrAckTimeout)
}

func (p *pathTestSuite) TestNonRecursiveWalkAsync() {
	var createdPathsOrder []string
	createdPathsWithDepth := make(map[string]uint32)
	tmpDir, err := os.MkdirTemp("", "kprobe_unit_test")
	p.Require().NoError(err)
	defer os.RemoveAll(tmpDir)

	createdPathsWithDepth[tmpDir] = 1
	createdPathsOrder = append(createdPathsOrder, tmpDir)

	testDir := filepath.Join(tmpDir, "test_dir")
	err = os.Mkdir(testDir, 0o744)
	p.Require().NoError(err)

	testDirTestFile := filepath.Join(tmpDir, "test_dir", "test_file")
	f, err := os.Create(testDirTestFile)
	p.Require().NoError(err)
	p.Require().NoError(f.Close())

	testFile := filepath.Join(tmpDir, "test_file")
	f, err = os.Create(testFile)
	p.Require().NoError(err)
	p.Require().NoError(f.Close())

	mounts, err := getAllMountPoints()
	p.Require().NoError(err)

	p.Require().Equal(len(createdPathsOrder), len(createdPathsWithDepth))

	expectedStatQueue := make([]statMatch, 0, len(createdPathsOrder))
	for _, path := range createdPathsOrder {

		depth, exists := createdPathsWithDepth[path]
		p.Require().True(exists)

		info, err := os.Lstat(path)
		p.Require().NoError(err)
		mnt := mounts.getMountByPath(path)
		p.Require().NotNil(mnt)
		expectedStatQueue = append(expectedStatQueue, statMatch{
			ino:        info.Sys().(*syscall.Stat_t).Ino,
			major:      mnt.DeviceMajor,
			minor:      mnt.DeviceMinor,
			depth:      depth,
			fileName:   info.Name(),
			isFromMove: true,
			tid:        2,
			fullPath:   path,
		})
	}

	ctx := context.Background()
	pTrav, err := newPathMonitor(ctx, newFixedThreadExecutor(ctx), 0, false)
	p.Require().NoError(err)
	defer func() {
		p.Require().NoError(pTrav.Close())
	}()

	pTrav.WalkAsync(tmpDir, 1, 2)

	tries := 0
	for idx := 0; idx < len(expectedStatQueue); {
		mPath, match := pTrav.GetMonitorPath(
			expectedStatQueue[idx].ino,
			expectedStatQueue[idx].major,
			expectedStatQueue[idx].minor,
			expectedStatQueue[idx].fileName,
		)

		if match {
			p.Require().Equal(expectedStatQueue[idx].fullPath, mPath.fullPath)
			p.Require().Equal(expectedStatQueue[idx].isFromMove, mPath.isFromMove)
			p.Require().Equal(expectedStatQueue[idx].tid, mPath.tid)
			p.Require().Equal(expectedStatQueue[idx].depth, mPath.depth)

			tries = 0
			idx++
			continue
		}

		if tries >= 3 {
			p.Require().Fail("no match found")
			return
		}

		time.Sleep(300 * time.Millisecond)
		tries++
	}

	select {
	case err = <-pTrav.errC:
	default:
	}

	p.Require().NoError(err)
	p.Require().Empty(pTrav.statQueue)
}

func (p *pathTestSuite) TestAddTraverserContextCancel() {
	tmpDir, err := os.MkdirTemp("", "kprobe_unit_test")
	p.Require().NoError(err)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()
	pTrav, err := newPathMonitor(ctx, newFixedThreadExecutor(ctx), 10*time.Second, true)
	p.Require().NoError(err)
	defer func() {
		p.Require().NoError(pTrav.Close())
	}()

	errChan := make(chan error)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		errPath := pTrav.AddPathToMonitor(ctx, tmpDir)
		if errPath != nil {
			errChan <- errPath
		}
		close(errChan)
	}()

	tries := 0
	for {
		if tries >= 4 {
			p.Require().Fail("no path was added in 5 tries")
		}
		if len(pTrav.statQueue) == 0 {
			tries++
			time.Sleep(1 * time.Second)
			continue
		}
		break
	}
	pTrav.cancelFn()

	err = <-errChan
	p.Require().ErrorIs(err, pTrav.ctx.Err())
}

func (p *pathTestSuite) TestAddTimeout() {
	tmpDir, err := os.MkdirTemp("", "kprobe_unit_test")
	p.Require().NoError(err)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()
	pTrav, err := newPathMonitor(ctx, newFixedThreadExecutor(ctx), 5*time.Second, true)
	p.Require().NoError(err)
	defer func() {
		p.Require().NoError(pTrav.Close())
	}()

	errChan := make(chan error)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		errPath := pTrav.AddPathToMonitor(ctx, tmpDir)
		if errPath != nil {
			errChan <- errPath
		}
		close(errChan)
	}()

	select {
	case err = <-errChan:
	case <-time.After(10 * time.Second):
		p.Require().Fail("no path was added in 10 seconds")
	}
	p.Require().ErrorIs(err, ErrAckTimeout)
}

func (p *pathTestSuite) TestRecursiveAdd() {
	var createdPathsOrder []string
	createdPathsWithDepth := make(map[string]uint32)
	tmpDir, err := os.MkdirTemp("", "kprobe_unit_test")
	p.Require().NoError(err)
	defer os.RemoveAll(tmpDir)

	createdPathsWithDepth[tmpDir] = 0
	createdPathsOrder = append(createdPathsOrder, tmpDir)

	testDir := filepath.Join(tmpDir, "test_dir")
	err = os.Mkdir(testDir, 0o744)
	p.Require().NoError(err)
	createdPathsWithDepth[testDir] = 1
	createdPathsOrder = append(createdPathsOrder, testDir)

	testDirTestFile := filepath.Join(tmpDir, "test_dir", "test_file")
	f, err := os.Create(testDirTestFile)
	p.Require().NoError(err)
	p.Require().NoError(f.Close())
	createdPathsWithDepth[testDirTestFile] = 2
	createdPathsOrder = append(createdPathsOrder, testDirTestFile)

	testFile := filepath.Join(tmpDir, "test_file")
	f, err = os.Create(testFile)
	p.Require().NoError(err)
	p.Require().NoError(f.Close())
	createdPathsWithDepth[testFile] = 1
	createdPathsOrder = append(createdPathsOrder, testFile)

	mounts, err := getAllMountPoints()
	p.Require().NoError(err)

	p.Require().Equal(len(createdPathsOrder), len(createdPathsWithDepth))

	expectedStatQueue := make([]statMatch, 0, len(createdPathsOrder))
	for _, path := range createdPathsOrder {

		depth, exists := createdPathsWithDepth[path]
		p.Require().True(exists)

		info, err := os.Lstat(path)
		p.Require().NoError(err)
		mnt := mounts.getMountByPath(path)
		p.Require().NotNil(mnt)
		expectedStatQueue = append(expectedStatQueue, statMatch{
			ino:        info.Sys().(*syscall.Stat_t).Ino,
			major:      mnt.DeviceMajor,
			minor:      mnt.DeviceMinor,
			depth:      depth,
			fileName:   info.Name(),
			isFromMove: false,
			tid:        0,
			fullPath:   path,
		})
	}

	ctx := context.Background()
	pTrav, err := newPathMonitor(ctx, newFixedThreadExecutor(ctx), 0, true)
	p.Require().NoError(err)
	defer func() {
		p.Require().NoError(pTrav.Close())
	}()

	errChan := make(chan error)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		errPath := pTrav.AddPathToMonitor(ctx, tmpDir)
		if errPath != nil {
			errChan <- errPath
		}
		close(errChan)
	}()

	tries := 0
	for idx := 0; idx < len(expectedStatQueue); {
		mPath, match := pTrav.GetMonitorPath(
			expectedStatQueue[idx].ino,
			expectedStatQueue[idx].major,
			expectedStatQueue[idx].minor,
			expectedStatQueue[idx].fileName,
		)

		if match {
			p.Require().Equal(expectedStatQueue[idx].fullPath, mPath.fullPath)
			p.Require().Equal(expectedStatQueue[idx].isFromMove, mPath.isFromMove)
			p.Require().Equal(expectedStatQueue[idx].tid, mPath.tid)
			p.Require().Equal(expectedStatQueue[idx].depth, mPath.depth)

			tries = 0
			idx++
			continue
		}

		if tries >= 3 {
			p.Require().Fail("no match found")
		}

		time.Sleep(100 * time.Millisecond)
		tries++
	}

	err = <-errChan
	p.Require().NoError(err)
	p.Require().Empty(pTrav.statQueue)
}

func (p *pathTestSuite) TestNonRecursiveAdd() {
	var createdPathsOrder []string
	createdPathsWithDepth := make(map[string]uint32)
	tmpDir, err := os.MkdirTemp("", "kprobe_unit_test")
	p.Require().NoError(err)
	defer os.RemoveAll(tmpDir)

	createdPathsWithDepth[tmpDir] = 0
	createdPathsOrder = append(createdPathsOrder, tmpDir)

	testDir := filepath.Join(tmpDir, "test_dir")
	err = os.Mkdir(testDir, 0o744)
	p.Require().NoError(err)
	createdPathsWithDepth[testDir] = 1
	createdPathsOrder = append(createdPathsOrder, testDir)

	testDirTestFile := filepath.Join(tmpDir, "test_dir", "test_file")
	f, err := os.Create(testDirTestFile)
	p.Require().NoError(err)
	p.Require().NoError(f.Close())

	testFile := filepath.Join(tmpDir, "test_file")
	f, err = os.Create(testFile)
	p.Require().NoError(err)
	p.Require().NoError(f.Close())
	createdPathsWithDepth[testFile] = 1
	createdPathsOrder = append(createdPathsOrder, testFile)

	mounts, err := getAllMountPoints()
	p.Require().NoError(err)

	p.Require().Equal(len(createdPathsOrder), len(createdPathsWithDepth))

	expectedStatQueue := make([]statMatch, 0, len(createdPathsOrder))
	for _, path := range createdPathsOrder {

		depth, exists := createdPathsWithDepth[path]
		p.Require().True(exists)

		info, err := os.Lstat(path)
		p.Require().NoError(err)
		mnt := mounts.getMountByPath(path)
		p.Require().NotNil(mnt)
		expectedStatQueue = append(expectedStatQueue, statMatch{
			ino:        info.Sys().(*syscall.Stat_t).Ino,
			major:      mnt.DeviceMajor,
			minor:      mnt.DeviceMinor,
			depth:      depth,
			fileName:   info.Name(),
			isFromMove: false,
			tid:        0,
			fullPath:   path,
		})
	}

	ctx := context.Background()
	pTrav, err := newPathMonitor(ctx, newFixedThreadExecutor(ctx), 0, false)
	p.Require().NoError(err)
	defer func() {
		p.Require().NoError(pTrav.Close())
	}()

	errChan := make(chan error)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		errPath := pTrav.AddPathToMonitor(ctx, tmpDir)
		if errPath != nil {
			errChan <- errPath
		}
		close(errChan)
	}()

	tries := 0
	for idx := 0; idx < len(expectedStatQueue); {
		mPath, match := pTrav.GetMonitorPath(
			expectedStatQueue[idx].ino,
			expectedStatQueue[idx].major,
			expectedStatQueue[idx].minor,
			expectedStatQueue[idx].fileName,
		)

		if match {
			p.Require().Equal(expectedStatQueue[idx].fullPath, mPath.fullPath)
			p.Require().Equal(expectedStatQueue[idx].isFromMove, mPath.isFromMove)
			p.Require().Equal(expectedStatQueue[idx].tid, mPath.tid)
			p.Require().Equal(expectedStatQueue[idx].depth, mPath.depth)

			tries = 0
			idx++
			continue
		}

		if tries >= 3 {
			p.Require().Fail("no match found")
		}

		time.Sleep(100 * time.Millisecond)
		tries++
	}

	err = <-errChan
	p.Require().NoError(err)
	p.Require().Empty(pTrav.statQueue)
}

func (p *pathTestSuite) TestStatErrAtRootAdd() {
	defer func() {
		lstat = os.Lstat
	}()
	// lstat error at root path to monitor
	lstat = func(path string) (os.FileInfo, error) {
		return nil, os.ErrNotExist
	}
	ctx := context.Background()
	pTrav, err := newPathMonitor(ctx, newFixedThreadExecutor(ctx), 0, true)
	p.Require().NoError(err)
	err = pTrav.AddPathToMonitor(ctx, "not-existing-path")
	p.Require().ErrorIs(err, os.ErrNotExist)
	p.Require().NoError(pTrav.Close())
}

func (p *pathTestSuite) TestStatErrAtWalk() {
	defer func() {
		lstat = os.Lstat
	}()

	tmpDir, err := os.MkdirTemp("", "kprobe_unit_test")
	p.Require().NoError(err)
	defer os.RemoveAll(tmpDir)

	testDir := filepath.Join(tmpDir, "test_dir")
	err = os.Mkdir(testDir, 0o744)
	p.Require().NoError(err)

	testDirTestFile := filepath.Join(tmpDir, "test_dir", "test_file")
	f, err := os.Create(testDirTestFile)
	p.Require().NoError(err)
	p.Require().NoError(f.Close())

	testFile := filepath.Join(tmpDir, "test_file")
	f, err = os.Create(testFile)
	p.Require().NoError(err)
	p.Require().NoError(f.Close())

	// lstat error at root path to monitor
	lstat = func(path string) (os.FileInfo, error) {
		info, err := os.Lstat(path)
		lstat = func(name string) (os.FileInfo, error) {
			return nil, os.ErrNotExist
		}

		return info, err
	}
	ctx := context.Background()
	pTrav, err := newPathMonitor(ctx, newFixedThreadExecutor(ctx), 0, true)
	p.Require().NoError(err)
	err = pTrav.AddPathToMonitor(ctx, tmpDir)
	p.Require().NoError(err)
	p.Require().NoError(pTrav.Close())
}

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
