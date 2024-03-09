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
	e             executor
	w             inotifyWatcher
	isRecursive   bool
	waitQueueChan chan struct{}
	sMatchTimeout time.Duration
	statQueue     []statMatch
}

var lstat = os.Lstat // for testing

func newPathMonitor(ctx context.Context, exec executor, timeOut time.Duration, isRecursive bool) (*pTraverser, error) {
	mWatcher, err := newInotifyWatcher()
	if err != nil {
		return nil, err
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
		e:             exec,
		w:             mWatcher,
		isRecursive:   isRecursive,
		sMatchTimeout: timeOut,
	}, nil
}

func (r *pTraverser) Close() error {
	r.cancelFn()
	return r.w.Close()
}

func (r *pTraverser) GetMonitorPath(ino uint64, major uint32, minor uint32, name string) (MonitorPath, bool) {
	if r.ctx.Err() != nil {
		return MonitorPath{}, false
	}

	r.mtx.Lock()
	defer r.mtx.Unlock()

	if len(r.statQueue) == 0 {
		return MonitorPath{}, false
	}

	monitorPath := r.statQueue[0]
	if monitorPath.ino != ino ||
		monitorPath.major != major ||
		monitorPath.minor != minor ||
		monitorPath.fileName != name {
		return MonitorPath{}, false
	}

	r.statQueue = r.statQueue[1:]

	if len(r.statQueue) == 0 && r.waitQueueChan != nil {
		close(r.waitQueueChan)
		r.waitQueueChan = nil
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
		return nil, err
	}
	names, err := f.Readdirnames(-1)
	_ = f.Close()
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	return names, nil
}

func (r *pTraverser) ErrC() <-chan error {
	return r.errC
}

func (r *pTraverser) WalkAsync(path string, depth uint32, tid uint32) {
	if r.ctx.Err() != nil {
		return
	}

	go func() {
		walkErr := r.e.Run(func() error {
			return r.walk(r.ctx, path, depth, true, tid)
		})

		if walkErr == nil {
			return
		}

		select {
		case r.errC <- walkErr:
		case <-r.ctx.Done():
		}
	}()
}

func (r *pTraverser) walkRecursive(ctx context.Context, path string, mounts mountPoints, depth uint32, isFromMove bool, tid uint32) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if r.ctx.Err() != nil {
		return r.ctx.Err()
	}

	if !r.isRecursive && depth > 1 {
		return nil
	}

	// get the mountpoint associated to this path
	mnt := mounts.getMountByPath(path)
	if mnt == nil {
		return fmt.Errorf("could not find mount for %s", path)
	}

	// add the inotify watcher if it does not exist
	if _, err := r.w.Add(mnt.DeviceMajor, mnt.DeviceMinor, path); err != nil {
		return err
	}

	r.mtx.Lock()
	info, err := lstat(path)
	if err != nil {
		// maybe this path got deleted/moved in the meantime
		// return nil
		r.mtx.Unlock()
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

	r.statQueue = append(r.statQueue, statMatch{
		ino:        info.Sys().(*syscall.Stat_t).Ino,
		major:      mnt.DeviceMajor,
		minor:      mnt.DeviceMinor,
		depth:      depth,
		fileName:   matchFileName,
		isFromMove: isFromMove,
		tid:        tid,
		fullPath:   path,
	})
	r.mtx.Unlock()

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
		if err = r.walkRecursive(ctx, filename, mounts, depth+1, isFromMove, tid); err != nil {
			//lint:ignore nilerr no errors returned for readDirNames from walkRecursive
			return nil
		}
	}
	return nil
}

func (r *pTraverser) waitForWalk(ctx context.Context) error {
	r.mtx.Lock()

	// statQueue is already empty, return
	if len(r.statQueue) == 0 {
		r.mtx.Unlock()
		return nil
	}

	r.waitQueueChan = make(chan struct{})
	r.mtx.Unlock()

	select {
	// ctx of pTraverser is done
	case <-r.ctx.Done():
		return r.ctx.Err()
	// ctx of walk is done
	case <-ctx.Done():
		return ctx.Err()
	// statQueue is empty
	case <-r.waitQueueChan:
		return nil
	// timeout
	case <-time.After(r.sMatchTimeout):
		return ErrAckTimeout
	}
}

func (r *pTraverser) walk(ctx context.Context, path string, depth uint32, isFromMove bool, tid uint32) error {
	// get a snapshot of all mountpoints
	mounts, err := getAllMountPoints()
	if err != nil {
		return err
	}

	// start walking the given path
	if err := r.walkRecursive(ctx, path, mounts, depth, isFromMove, tid); err != nil {
		return err
	}

	// wait for the monitor queue to get empty
	return r.waitForWalk(ctx)
}

func (r *pTraverser) AddPathToMonitor(ctx context.Context, path string) error {
	if r.ctx.Err() != nil {
		return r.ctx.Err()
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	// we care about the existence of the path only in AddPathToMonitor
	// walk masks out all file existence errors
	_, err := lstat(path)
	if err != nil {
		return err
	}

	// paths from AddPathToMonitor are always starting with a depth of 0
	return r.e.Run(func() error {
		return r.walk(ctx, path, 0, false, 0)
	})
}
