// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package kprobes

import (
	"sync"
	"syscall"

	"github.com/pkg/errors"
	"golang.org/x/sys/unix"

	"github.com/elastic/beats/v7/x-pack/auditbeat/tracing"
)

type mountPoint struct {
	fsType string
	path   string
}

type unmounter struct {
	sync.Mutex
	paths map[string]int
}

var defaultMounts = []*mountPoint{
	{fsType: "tracefs", path: "/sys/kernel/tracing"},
	{fsType: "debugfs", path: "/sys/kernel/debug"},
}

var fsTracker unmounter

func (m mountPoint) mount() error {
	return unix.Mount(m.fsType, m.path, m.fsType, 0, "")
}

func (m mountPoint) unmount() error {
	return syscall.Unmount(m.path, 0)
}

func (m *mountPoint) String() string {
	return m.fsType + " at " + m.path
}

func (fs *unmounter) Mount(m mountPoint) error {
	fs.Lock()
	defer fs.Unlock()
	if _, exists := fs.paths[m.path]; exists {
		return errors.Errorf("path '%s' already mounted", m.path)
	}
	if err := m.mount(); err != nil {
		return err
	}
	if fs.paths == nil {
		fs.paths = make(map[string]int)
	}
	fs.paths[m.path] = 1
	return nil
}

func (fs *unmounter) Use(path string) {
	fs.Lock()
	defer fs.Unlock()
	if _, exists := fs.paths[path]; exists {
		fs.paths[path]++
	}
}

func (fs *unmounter) Release(path string) error {
	fs.Lock()
	defer fs.Unlock()
	cur, exists := fs.paths[path]
	if !exists {
		return nil
	}
	if cur == 1 {
		delete(fs.paths, path)
		m := mountPoint{path: path}
		if err := m.unmount(); err != nil {
			return errors.Wrapf(err, "failed to unmount '%s'", path)
		}
	} else {
		fs.paths[path]--
	}
	return nil
}

func (e *Engine) openTraceFS() (err error) {
	if e.traceFSpath != nil {
		fsTracker.Use(*e.traceFSpath)
		e.traceFS, err = tracing.NewTraceFSWithPath(*e.traceFSpath)
		return err
	}
	if e.autoMount {
		for _, mount := range defaultMounts {
			if err = fsTracker.Mount(*mount); err != nil {
				e.log.Debugf("Mount %s failed: %v", mount, err)
				continue
			}
			if tracing.IsTraceFSAvailable() != nil {
				e.log.Warnf("Mounted %s but no kprobes available", mount, err)
				fsTracker.Release(mount.path)
				continue
			}
			e.log.Debugf("Mounted %s", mount)
			break
		}
	}
	e.traceFS, err = tracing.NewTraceFS()
	return err
}
