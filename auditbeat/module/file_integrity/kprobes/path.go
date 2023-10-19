package kprobes

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"syscall"
	"time"
)

type pathTraverser interface {
	AddPathToMonitor(ctx context.Context, path string) error
	Ack(ctx context.Context, ino uint64, major uint32, minor uint32, name string) (string, bool)
	Close() error
	WalkAsync(path string)
}

type statMatch struct {
	ctx      context.Context
	ack      chan struct{}
	ino      uint64
	major    uint32
	minor    uint32
	fileName string
	fullPath string
}

type pTraverser struct {
	sync.RWMutex
	ctx           context.Context
	cancelFn      context.CancelFunc
	e             executor
	w             inotifyWatcher
	sMatch        *statMatch
	isRecursive   bool
	sMatchTimeout time.Duration
}

func newPathMonitor(ctx context.Context, exec executor, timeOut time.Duration, isRecursive bool) (pathTraverser, error) {
	mWatcher, err := newInotifyWatcher()
	if err != nil {
		return nil, err
	}

	if timeOut == 0 {
		timeOut = 5 * time.Second
	}

	mCtx, cancelFn := context.WithCancel(ctx)

	return &pTraverser{
		RWMutex:       sync.RWMutex{},
		ctx:           mCtx,
		cancelFn:      cancelFn,
		e:             exec,
		w:             mWatcher,
		sMatch:        nil,
		isRecursive:   isRecursive,
		sMatchTimeout: timeOut,
	}, nil
}

func (r *pTraverser) Close() error {
	r.cancelFn()
	return r.w.Close()
}

func (r *pTraverser) Ack(ctx context.Context, ino uint64, major uint32, minor uint32, name string) (string, bool) {
	r.Lock()
	defer r.Unlock()

	if r.sMatch == nil {
		return "", false
	}

	if r.sMatch.ino != ino ||
		r.sMatch.major != major ||
		r.sMatch.minor != minor ||
		r.sMatch.fileName != name {
		return "", false
	}

	fullPath := r.sMatch.fullPath

	select {
	case r.sMatch.ack <- struct{}{}:
		return fullPath, true
	case <-ctx.Done():
		// context of the caller to Ack the match is done
		return "", false
	case <-r.sMatch.ctx.Done():
		// context of the wait for the stat to be acked is done
		return "", false
	case <-r.ctx.Done():
		// context of the path traverser is done
		return "", false
	}
}

func (r *pTraverser) statWithAckWait(ctx context.Context, path string, mnt *mount) (os.FileInfo, error) {

	// if we are about to stat the root of the mountpoint, and the subtree path has a different base
	// from the base name of the path (e.g. /watch [path] -> /etc/test [subtree])
	// the filename reported in the kprobe event will be "test" instead of "watch". Thus, we need to
	// construct the filename based on the base name of the subtree.
	matchFileName := filepath.Base(path)
	if path == mnt.Path && mnt.Subtree != "/" {
		subTreeBase := filepath.Base(mnt.Subtree)
		if matchFileName != subTreeBase {
			matchFileName = subTreeBase
		}
	}

	// Lock before lstat as the latter will cause a probe event to be emitted and
	// not holding the lock a priori can result in a race condition.
	r.Lock()
	info, err := os.Lstat(path)
	if err != nil {
		r.Unlock()
		return nil, err
	}

	match := &statMatch{
		ctx:      ctx, // ctx of the stat request
		ack:      make(chan struct{}),
		ino:      info.Sys().(*syscall.Stat_t).Ino,
		major:    mnt.DeviceMajor,
		minor:    mnt.DeviceMinor,
		fileName: matchFileName,
		fullPath: path,
	}
	r.sMatch = match
	r.Unlock()

	// wait for this stat syscall to be acked; look at r.Ack(...).
	select {
	case <-match.ack:
	case <-time.After(r.sMatchTimeout):
		err = fmt.Errorf("err at waiting probe match for path %s: %v", path, ErrAckTimeout)
	case <-ctx.Done():
		// context of the stat request is done
		err = ctx.Err()
	case <-r.ctx.Done():
		// context of the path traverser is done
		err = r.ctx.Err()
	}

	// clear the pending stat match
	r.Lock()
	r.sMatch = nil
	r.Unlock()

	if err != nil {
		return nil, err
	}

	return info, nil
}

type WalkFunc func(path string, info fs.FileInfo, err error) error

func readDirNames(dirname string) ([]string, error) {
	f, err := os.Open(dirname)
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

func (r *pTraverser) WalkAsync(path string) {
	go func() {
		fmt.Printf("[%v] walk async started: %s\n", time.Now(), path)
		defer func() {
			fmt.Printf("[%v] walk async ended: %s\n", time.Now(), path)
		}()
		mounts, err := getAllMountPoints()
		if err != nil {
			return
		}

		_ = r.e.Run(r.walk(r.ctx, path, mounts, false))
	}()
}

func (r *pTraverser) walkRecursive(path string, info fs.FileInfo, mounts mountPoints) error {

	//TODO(panosk): handle different mountpoints that are mounted inside the parent tree

	if !info.IsDir() {
		_, _ = os.Lstat(path)
		return nil
	} else if !r.isRecursive {
		return nil
	}

	names, err := readDirNames(path)
	if err != nil {
		return nil
	}

	for _, name := range names {
		filename := filepath.Join(path, name)
		fileInfo, err := os.Lstat(filename)
		if err != nil {
			return nil
		}

		if err = r.walkRecursive(filename, fileInfo, mounts); err != nil {
			return nil
		}
	}
	return nil
}

func (r *pTraverser) walk(ctx context.Context, path string, mounts mountPoints, isMonitor bool) func() error {
	return func() error {
		mnt := mounts.getMountByPath(path)
		if mnt == nil {
			return fmt.Errorf("could not find mount for %s", path)
		}

		if _, err := r.w.Add(mnt.DeviceMinor, mnt.DeviceMinor, path); err != nil {
			return err
		}

		var info os.FileInfo
		var err error

		if !isMonitor {
			info, err = os.Lstat(path)
		} else {
			info, err = r.statWithAckWait(ctx, path, mnt)
		}
		if err != nil {
			return nil
		}

		_ = r.walkRecursive(path, info, mounts)
		return nil
	}
}

func (r *pTraverser) AddPathToMonitor(ctx context.Context, path string) error {
	fmt.Printf("[%v] monitor started: %s\n", time.Now(), path)
	defer func() {
		fmt.Printf("[%v] monitor ended: %s\n", time.Now(), path)
	}()
	mounts, err := getAllMountPoints()
	if err != nil {
		return err
	}

	if err := r.e.Run(r.walk(ctx, path, mounts, true)); err != nil {
		return err
	}

	return nil
}
