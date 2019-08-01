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

	"github.com/pkg/errors"

	"github.com/elastic/fleet/x-pack/pkg/core/logger"
)

// Comparer receives a file and the saved information about that file from the previous scan,
// it's return true if the values are identical and will return the new state object to persist.
type Comparer func(string, interface{}) (bool, interface{}, error)

// DefaultComparer is the default comparer used by the watch.
var DefaultComparer = ContentComparer

// Watch allow to watch a set of file on disk and periodically scan if the files is different
// than the last time we have seen the files. NOTE: The Watch implementation is not threadsafe.
type Watch struct {
	log      *logger.Logger
	logbook  map[string]interface{}
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

	return &Watch{log: log, logbook: make(map[string]interface{}), comparer: f}, nil
}

// Watch add a new files to the list of files to watch on disk.
// NOTE: If we already know the file we will just keep the old record.
func (w *Watch) Watch(file string) {
	_, ok := w.logbook[file]
	if !ok {
		w.logbook[file] = nil
	}
}

// IsWatching returns true if the files is currently watch.
func (w *Watch) IsWatching(file string) bool {
	_, ok := w.logbook[file]
	return ok
}

// Watched returns the list of watched files.
func (w *Watch) Watched() []string {
	watched := make([]string, 0, len(w.logbook))
	for file := range w.logbook {
		watched = append(watched, file)
	}

	sort.Strings(watched)

	return watched
}

// Unwatch remove a files from the list of file to keep track of changes.
func (w *Watch) Unwatch(file string) {
	delete(w.logbook, file)
}

// Scan goes through the list of files and verifies if they have been modified since we last check
// for them.
func (w *Watch) Scan() ([]string, error) {
	var modifiedFiles []string
	for file, record := range w.logbook {
		change, newRecord, err := w.comparer(file, record)
		if err != nil {
			return []string{}, err
		}

		if change {
			w.logbook[file] = newRecord
			modifiedFiles = append(modifiedFiles, file)
		}
	}

	return modifiedFiles, nil
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
		return false, nil, errors.Wrap(err, "could not get information about the file")
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
		return nil, errors.Wrap(err, "could not open file")
	}

	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, errors.Wrap(err, "could not generate checksum")
	}
	return h.Sum(nil), nil
}
