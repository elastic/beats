// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !windows

package proc

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

var (
	ErrInvalidProcNsPidStatContent       = errors.New("invalid /proc/ns/pid stat content")
	ErrInvalidProcNsPidStatParsedContent = errors.New("invalid /proc/ns/pid stat parsed content")
)

type NamespaceInfo struct {
	Ino uint64
}

// ReadNamespace reads process namespace information from /proc/<pid>/ns/pid.
func ReadNamespace(root string, pid string) (nsInfo NamespaceInfo, err error) {
	return ReadNamespaceFS(os.DirFS(root), pid)
}

func ReadNamespaceFS(fsys fs.FS, pid string) (nsInfo NamespaceInfo, err error) {
	// Get the namespace stat
	nsStat, err := getNamespaceStat(fsys, pid)
	if err != nil {
		return
	}

	// Set the namespace ino
	nsInfo.Ino = nsStat.Ino

	return nsInfo, nil
}

func getNamespaceStat(fsys fs.FS, pid string) (*syscall.Stat_t, error) {
	// Path for the ns pid file
	fn := filepath.Join("proc", pid, filepath.Join("ns", "pid"))

	// Calling stat on the ns pid file
	stat, err := fs.Stat(fsys, fn)
	if err != nil {
		return nil, err
	}

	// Pull stat data
	dataSource := stat.Sys()
	if dataSource == nil {
		return nil, ErrInvalidProcNsPidStatContent
	}

	// Convert pulled stat data into stat structure
	dsStat, ok := dataSource.(*syscall.Stat_t)
	if !ok {
		return nil, ErrInvalidProcNsPidStatParsedContent
	}

	return dsStat, nil
}
