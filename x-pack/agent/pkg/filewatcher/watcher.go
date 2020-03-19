// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package filewatcher

import (
	"bytes"
	"crypto/sha256"
	"io"
	"os"
	"sort"
	"sync"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/core/logger"
)

// Comparer receives a file and the saved information about that file from the previous scan,
// it's return true if the values are identical and will return the new state object to persist.
type Comparer func(string, interface{}) (bool, interface{}, error)

// DefaultComparer is the default comparer used by the watch.
var DefaultComparer = ContentComparer

type fileinfo struct {
	watched bool
	record  interface{}
}

// Status is returned when you call Update() on the filewatched and will contains informations about
// the unwatched files, the unchanged files, the updated files and if an update is required.
type Status struct {
	Updated    []string
	Unchanged  []string
	Unwatched  []string
	NeedUpdate bool
}

// Watch allow to watch a set of file on disk and periodically scan if the files is different
// than the last time we have seen the files. NOTE: The Watch implementation is not threadsafe.
type Watch struct {
	mu       sync.Mutex
	log      *logger.Logger
	logbook  map[string]fileinfo
	comparer Comparer
}

// New returns a new Watch that will watch for file changes.
func New(log *logger.Logger, f Comparer) (*Watch, error) {
	var err error
	if log == nil {
		log, err = logger.New()
		if err != nil {
			return nil, err
		}
	}

	return &Watch{log: log, logbook: make(map[string]fileinfo), comparer: f}, nil
}

// Watch add a new files to the list of files to watch on disk.
// NOTE: If we already know the file we will just keep the old record.
func (w *Watch) Watch(file string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	v, ok := w.logbook[file]
	if !ok {
		w.logbook[file] = fileinfo{watched: true}
	}
	v.watched = true
	w.logbook[file] = v
}

// Cleanup removed all unwatched files from the logbook.
func (w *Watch) Cleanup() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	var removed []string
	for file, info := range w.logbook {
		if !info.watched {
			removed = append(removed, file)
			delete(w.logbook, file)
		}
	}
	return removed
}

// Update returns multiples list, updated file, unchanged files and unwatched files.
// - Check if we are watching new files.
// - Check if we unwatch some files.
// - Check if watched files have changed.
func (w *Watch) Update() (Status, error) {
	var (
		updated    []string
		unchanged  []string
		unwatched  []string
		needUpdate bool
		err        error
	)

	unwatched = w.Cleanup()

	if len(unwatched) > 0 {
		needUpdate = true
	}

	updated, unchanged, err = w.scan()
	if err != nil {
		return Status{}, err
	}

	if len(updated) > 0 {
		needUpdate = true
	}

	return Status{
		Updated:    updated,
		Unchanged:  unchanged,
		Unwatched:  unwatched,
		NeedUpdate: needUpdate,
	}, nil
}

// Reset mark all the files in the logbook as unwatched.
func (w *Watch) Reset() {
	w.mu.Lock()
	defer w.mu.Unlock()
	for file, info := range w.logbook {
		info.watched = false
		w.logbook[file] = info
	}
}

// IsWatching returns true if the files is currently watch.
func (w *Watch) IsWatching(file string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	_, ok := w.logbook[file]
	return ok
}

// Watched returns the list of watched files.
func (w *Watch) Watched() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	watched := make([]string, 0, len(w.logbook))
	for file := range w.logbook {
		watched = append(watched, file)
	}

	sort.Strings(watched)
	return watched
}

// Unwatch remove a files from the list of file to keep track of changes.
func (w *Watch) Unwatch(file string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.logbook, file)
}

// scan goes through the list of files and verifies if they have been modified since we last check
// for them.
func (w *Watch) scan() (modifiedFiles []string, unchanged []string, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for file, info := range w.logbook {
		change, newRecord, err := w.comparer(file, info.record)
		if err != nil {
			return []string{}, []string{}, err
		}

		if change {
			info := w.logbook[file]
			info.record = newRecord
			w.logbook[file] = info
			modifiedFiles = append(modifiedFiles, file)
		} else {
			unchanged = append(unchanged, file)
		}
	}

	return modifiedFiles, unchanged, nil
}

// Invalidate invalidates the current cache.
func (w *Watch) Invalidate() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.logbook = make(map[string]fileinfo)
}

type record struct {
	info     os.FileInfo
	checksum []byte
}

// ContentComparer uses the last modified date of the file and a checksum of the content of the file
// to know if the files are differents and must be processed again.
func ContentComparer(file string, r interface{}) (bool, interface{}, error) {
	stat, err := os.Stat(file)
	if err != nil {
		return false, nil, errors.New(err,
			"could not get information about the file",
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, file))
	}

	// We never saw the file before.
	if r == nil {
		checksum, err := checksum(file)
		if err != nil {
			return false, nil, err
		}
		return true, record{info: stat, checksum: checksum}, nil
	}

	// We already saw the file.
	fileRecord := r.(record)

	// If the modification time is the same, we assume nothing was changed on disk.
	if stat.ModTime().Sub(fileRecord.info.ModTime()) == 0 {
		return false, fileRecord, nil
	}

	checksum, err := checksum(file)
	if err != nil {
		return false, nil, err
	}

	// content is the same, no change.
	if bytes.Equal(checksum, fileRecord.checksum) {
		return false, fileRecord, nil
	}

	return true, record{info: stat, checksum: checksum}, nil
}

func checksum(file string) ([]byte, error) {
	// Mod time was changed on on the file, now lets looks at the content of the file.
	f, err := os.Open(file)
	if err != nil {
		return nil, errors.New(err,
			"could not open file",
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, file))
	}

	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, errors.New(err,
			"could not generate checksum",
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, file))
	}
	return h.Sum(nil), nil
}
