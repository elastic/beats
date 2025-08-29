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
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

type MonitorPath struct {
	fullPath   string
	depth      uint32
	isFromMove bool
	tid        uint32
}

type pathTraverser interface {
	AddPathToMonitor(ctx context.Context, path string) error
	GetMonitorPath(ino uint64, major uint32, minor uint32, name string) (MonitorPath, bool)
	WalkAsync(path string, depth uint32, tid uint32)
	ErrC() <-chan error
	Close() error
}

type statMatch struct {
	ino        uint64
	major      uint32
	minor      uint32
	depth      uint32
	fileName   string
	isFromMove bool
	tid        uint32
	fullPath   string
}

type pTraverser struct {
	mtx           sync.RWMutex
	errC          chan error
	ctx           context.Context
	cancelFn      context.CancelFunc
	exec          executor
	watcher       inotifyWatcher
	isRecursive   bool
	waitQueueChan chan struct{}
	sMatchTimeout time.Duration
	statQueue     []statMatch
}

var lstat = os.Lstat // for testing

func newPathMonitor(ctx context.Context, exec executor, timeOut time.Duration, isRecursive bool) (*pTraverser, error) {
	mWatcher, err := newInotifyWatcher()
	if err != nil {
		return nil, fmt.Errorf("error creating new inotify watcher: %w", err)
	}

	if timeOut == 0 {
		timeOut = 5 * time.Second
	}

	mCtx, cancelFn := context.WithCancel(ctx)

	return &pTraverser{
		mtx:           sync.RWMutex{},
		ctx:           mCtx,
		errC:          make(chan error),
		cancelFn:      cancelFn,
		exec:          exec,
		watcher:       mWatcher,
		isRecursive:   isRecursive,
		sMatchTimeout: timeOut,
	}, nil
}

func (traverser *pTraverser) Close() error {
	traverser.cancelFn()
	return traverser.watcher.Close()
}

func (traverser *pTraverser) GetMonitorPath(ino uint64, major uint32, minor uint32, name string) (MonitorPath, bool) {
	if traverser.ctx.Err() != nil {
		return MonitorPath{}, false
	}

	traverser.mtx.Lock()
	defer traverser.mtx.Unlock()

	if len(traverser.statQueue) == 0 {
		return MonitorPath{}, false
	}

	monitorPath := traverser.statQueue[0]
	if monitorPath.ino != ino ||
		monitorPath.major != major ||
		monitorPath.minor != minor ||
		monitorPath.fileName != name {
		return MonitorPath{}, false
	}

	traverser.statQueue = traverser.statQueue[1:]

	if len(traverser.statQueue) == 0 && traverser.waitQueueChan != nil {
		close(traverser.waitQueueChan)
		traverser.waitQueueChan = nil
	}

	return MonitorPath{
		fullPath:   monitorPath.fullPath,
		depth:      monitorPath.depth,
		isFromMove: monitorPath.isFromMove,
		tid:        monitorPath.tid,
	}, true
}

func readDirNames(dirName string) ([]string, error) {
	f, err := os.Open(dirName)
	if err != nil {
		return nil, fmt.Errorf("error opening directory %s: %w", dirName, err)
	}
	names, err := f.Readdirnames(-1)
	_ = f.Close()
	if err != nil {
		return nil, fmt.Errorf("error reading %s: %w", dirName, err)
	}
	sort.Strings(names)
	return names, nil
}

func (traverser *pTraverser) ErrC() <-chan error {
	return traverser.errC
}

func (traverser *pTraverser) WalkAsync(path string, depth uint32, tid uint32) {
	if traverser.ctx.Err() != nil {
		return
	}

	go func() {
		walkErr := traverser.exec.Run(func() error {
			return traverser.walk(traverser.ctx, path, depth, true, tid)
		})

		if walkErr == nil {
			return
		}

		select {
		case traverser.errC <- walkErr:
		case <-traverser.ctx.Done():
		}
	}()
}

func (traverser *pTraverser) walkRecursive(ctx context.Context, path string, mounts mountPoints, depth uint32, isFromMove bool, tid uint32) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if traverser.ctx.Err() != nil {
		return traverser.ctx.Err()
	}

	if !traverser.isRecursive && depth > 1 {
		return nil
	}

	// get the mountpoint associated to this path
	mnt := mounts.getMountByPath(path)
	if mnt == nil {
		return fmt.Errorf("could not find mount for %s", path)
	}

	// add the inotify watcher if it does not exist
	if _, err := traverser.watcher.Add(mnt.DeviceMajor, mnt.DeviceMinor, path); err != nil {
		return fmt.Errorf("error adding inotify watch for %s: %w", path, err)
	}

	traverser.mtx.Lock()
	info, err := lstat(path)
	if err != nil {
		// maybe this path got deleted/moved in the meantime
		// return nil
		traverser.mtx.Unlock()
		//lint:ignore nilerr no errors returned for lstat from walkRecursive
		return nil
	}

	// if we are about to stat the root of the mountpoint, and the subtree has a different base
	// from the base of the path (e.g. /watch [path] -> /etc/test [subtree])
	// the filename reported in the kprobe event will be "test" instead of "watch". Thus, we need to
	// construct the filename based on the base name of the subtree.
	mntPath := strings.Replace(path, mnt.Path, "", 1)
	if !strings.HasPrefix(mntPath, mnt.Subtree) {
		mntPath = filepath.Join(mnt.Subtree, mntPath)
	}

	matchFileName := filepath.Base(mntPath)

	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("file info is %T, not a stat_t object", info.Sys())
	}

	traverser.statQueue = append(traverser.statQueue, statMatch{
		ino:        stat.Ino,
		major:      mnt.DeviceMajor,
		minor:      mnt.DeviceMinor,
		depth:      depth,
		fileName:   matchFileName,
		isFromMove: isFromMove,
		tid:        tid,
		fullPath:   path,
	})
	traverser.mtx.Unlock()

	if !info.IsDir() {
		return nil
	}

	names, err := readDirNames(path)
	if err != nil {
		// maybe this dir got deleted/moved in the meantime
		// return nil
		//lint:ignore nilerr no errors returned for readDirNames from walkRecursive
		return nil
	}

	for _, name := range names {
		filename := filepath.Join(path, name)
		if err = traverser.walkRecursive(ctx, filename, mounts, depth+1, isFromMove, tid); err != nil {
			//lint:ignore nilerr no errors returned for readDirNames from walkRecursive
			return nil
		}
	}
	return nil
}

func (traverser *pTraverser) waitForWalk(ctx context.Context) error {
	traverser.mtx.Lock()

	// statQueue is already empty, return
	if len(traverser.statQueue) == 0 {
		traverser.mtx.Unlock()
		return nil
	}

	traverser.waitQueueChan = make(chan struct{})
	traverser.mtx.Unlock()

	select {
	// ctx of pTraverser is done
	case <-traverser.ctx.Done():
		return traverser.ctx.Err()
	// ctx of walk is done
	case <-ctx.Done():
		return ctx.Err()
	// statQueue is empty
	case <-traverser.waitQueueChan:
		return nil
	// timeout
	case <-time.After(traverser.sMatchTimeout):
		return ErrAckTimeout
	}
}

func (traverser *pTraverser) walk(ctx context.Context, path string, depth uint32, isFromMove bool, tid uint32) error {
	// get a snapshot of all mountpoints
	mounts, err := getAllMountPoints()
	if err != nil {
		return fmt.Errorf("error getting mount points: %w", err)
	}

	// start walking the given path
	if err := traverser.walkRecursive(ctx, path, mounts, depth, isFromMove, tid); err != nil {
		return fmt.Errorf("error walking path %s: %w", path, err)
	}

	// wait for the monitor queue to get empty
	return traverser.waitForWalk(ctx)
}

func (traverser *pTraverser) AddPathToMonitor(ctx context.Context, path string) error {
	if traverser.ctx.Err() != nil {
		return traverser.ctx.Err()
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	// we care about the existence of the path only in AddPathToMonitor
	// walk masks out all file existence errors
	_, err := lstat(path)
	if err != nil {
		return fmt.Errorf("error stating path %s: %w", path, err)
	}

	// paths from AddPathToMonitor are always starting with a depth of 0
	return traverser.exec.Run(func() error {
		return traverser.walk(ctx, path, 0, false, 0)
	})
}
