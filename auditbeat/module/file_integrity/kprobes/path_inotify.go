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
	"errors"
	"sync"

	"golang.org/x/sys/unix"
)

type mountID struct {
	major uint32
	minor uint32
}

type inotifyWatcher interface {
	Add(devMajor uint32, devMinor uint32, mountPath string) (bool, error)
	Close() error
}

type iWatcher struct {
	inotifyFD int
	mounts    map[mountID]struct{}
	uniqueFDs map[uint32]struct{}
	closed    bool
	mtx       sync.Mutex
}

var inotifyAddWatch = unix.InotifyAddWatch

// newInotifyWatcher creates a new inotifyWatcher object and initializes the inotify file descriptor.
//
// It returns a pointer to the newly created inotifyWatcher object and an error if there was any.
//
// Note: Having such a inotifyWatcher is necessary for Linux kernels v5.15+ (commit
// https://lore.kernel.org/all/20210810151220.285179-5-amir73il@gmail.com/). Essentially this commit adds
// a proactive check in the inline fsnotify helpers to avoid calling fsnotify() and __fsnotify_parent() (our
// kprobes) in case there are no marks of any type (inode/sb/mount) for an inode's super block. To bypass this check,
// and always make sure that our kprobes are triggered, we use the inotifyWatcher to add an inotify watch on the
// mountpoints that we are interested in (inotify IN_MOUNT doesn't interfere with our probes). Also, it keeps track
// of the mountpoints (referenced by devmajor and devminor) that have already had an inotify watch added and does not
// add them again.
func newInotifyWatcher() (*iWatcher, error) {
	fd, errno := unix.InotifyInit1(unix.IN_CLOEXEC | unix.IN_NONBLOCK)
	if fd == -1 {
		return nil, errno
	}

	return &iWatcher{
		inotifyFD: fd,
		mounts:    make(map[mountID]struct{}),
		uniqueFDs: make(map[uint32]struct{}),
	}, nil
}

// Add adds a mount to the inotifyWatcher.
//
// It takes in the device major number, device minor number, and mount as parameters.
// It returns false if the mount with the same device major number and minor number already
// has an inotify watch added. Also, it returns an error if there was any error.
func (w *iWatcher) Add(devMajor uint32, devMinor uint32, mountPath string) (bool, error) {
	w.mtx.Lock()
	defer w.mtx.Unlock()

	if w.closed {
		return false, errors.New("inotify watcher already closed")
	}

	id := mountID{
		major: devMajor,
		minor: devMinor,
	}

	if _, exists := w.mounts[id]; exists {
		return false, nil
	}

	wd, err := inotifyAddWatch(w.inotifyFD, mountPath, unix.IN_UNMOUNT)
	if err != nil {
		return false, err
	}

	_, fdExists := w.uniqueFDs[uint32(wd)]
	if fdExists {
		return false, nil
	}

	w.uniqueFDs[uint32(wd)] = struct{}{}
	w.mounts[id] = struct{}{}
	return true, nil
}

// Close closes the inotifyWatcher and releases any associated resources.
//
// It removes all inotify watches added. If any error occurs
// during the removal of watches, it will be accumulated and returned as a single
// error value. After removing all watches, it closes the inotify file descriptor.
func (w *iWatcher) Close() error {
	w.mtx.Lock()
	defer w.mtx.Unlock()

	var allErr error
	for fd := range w.uniqueFDs {
		if _, err := unix.InotifyRmWatch(w.inotifyFD, fd); err != nil {
			allErr = errors.Join(allErr, err)
		}
	}
	w.uniqueFDs = nil

	allErr = errors.Join(allErr, unix.Close(w.inotifyFD))

	w.mounts = nil

	return allErr
}
