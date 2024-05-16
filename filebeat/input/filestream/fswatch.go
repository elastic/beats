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

package filestream

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/elastic/go-concert/timed"
	"github.com/elastic/go-concert/unison"

	"github.com/elastic/beats/v7/filebeat/input/file"
	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	commonfile "github.com/elastic/beats/v7/libbeat/common/file"
	"github.com/elastic/beats/v7/libbeat/common/match"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	RecursiveGlobDepth           = 8
	DefaultFingerprintSize int64 = 1024 // 1KB
	scannerDebugKey              = "scanner"
	watcherDebugKey              = "file_watcher"
)

var (
	errFileTooSmall = errors.New("file size is too small for ingestion")
)

type fileWatcherConfig struct {
	// Interval is the time between two scans.
	Interval time.Duration `config:"check_interval"`
	// ResendOnModTime  if a file has been changed according to modtime but the size is the same
	// it is still considered truncation.
	ResendOnModTime bool `config:"resend_on_touch"`
	// Scanner is the configuration of the scanner.
	Scanner fileScannerConfig `config:",inline"`
}

// fileWatcher gets the list of files from a FSWatcher and creates events by
// comparing the files between its last two runs.
type fileWatcher struct {
	cfg     fileWatcherConfig
	prev    map[string]loginp.FileDescriptor
	scanner loginp.FSScanner
	log     *logp.Logger
	events  chan loginp.FSEvent
}

func newFileWatcher(paths []string, ns *conf.Namespace) (loginp.FSWatcher, error) {
	var config *conf.C
	if ns == nil {
		config = conf.NewConfig()
	} else {
		config = ns.Config()
	}

	return newScannerWatcher(paths, config)
}

func newScannerWatcher(paths []string, c *conf.C) (loginp.FSWatcher, error) {
	config := defaultFileWatcherConfig()
	err := c.Unpack(&config)
	if err != nil {
		return nil, err
	}
	scanner, err := newFileScanner(paths, config.Scanner)
	if err != nil {
		return nil, err
	}
	return &fileWatcher{
		log:     logp.NewLogger(watcherDebugKey),
		cfg:     config,
		prev:    make(map[string]loginp.FileDescriptor, 0),
		scanner: scanner,
		events:  make(chan loginp.FSEvent),
	}, nil
}

func defaultFileWatcherConfig() fileWatcherConfig {
	return fileWatcherConfig{
		Interval:        10 * time.Second,
		ResendOnModTime: false,
		Scanner:         defaultFileScannerConfig(),
	}
}

func (w *fileWatcher) Run(ctx unison.Canceler) {
	defer close(w.events)

	// run initial scan before starting regular
	w.watch(ctx)

	_ = timed.Periodic(ctx, w.cfg.Interval, func() error {
		w.watch(ctx)

		return nil
	})
}

func (w *fileWatcher) watch(ctx unison.Canceler) {
	w.log.Debug("Start next scan")

	paths := w.scanner.GetFiles()

	// for debugging purposes
	writtenCount := 0
	truncatedCount := 0
	renamedCount := 0
	removedCount := 0
	createdCount := 0

	newFilesByName := make(map[string]*loginp.FileDescriptor)
	newFilesByID := make(map[string]*loginp.FileDescriptor)

	for path, fd := range paths {
		// if the scanner found a new path or an existing path
		// with a different file, it is a new file
		prevDesc, ok := w.prev[path]
		sfd := fd // to avoid memory aliasing
		if !ok || !loginp.SameFile(&prevDesc, &sfd) {
			newFilesByName[path] = &sfd
			newFilesByID[fd.FileID()] = &sfd
			continue
		}

		var e loginp.FSEvent
		switch {

		// the new size is smaller, the file was truncated
		case prevDesc.Info.Size() > fd.Info.Size():
			e = truncateEvent(path, fd)
			truncatedCount++

		// the size is the same, timestamps are different, the file was touched
		case prevDesc.Info.Size() == fd.Info.Size() && prevDesc.Info.ModTime() != fd.Info.ModTime():
			if w.cfg.ResendOnModTime {
				e = truncateEvent(path, fd)
				truncatedCount++
			}

		// the new size is larger, something was written
		case prevDesc.Info.Size() < fd.Info.Size():
			e = writeEvent(path, fd)
			writtenCount++
		}

		// if none of the conditions were true, the file remained unchanged and we don't need to create an event
		if e.Op != loginp.OpDone {
			select {
			case <-ctx.Done():
				return
			case w.events <- e:
			}
		}

		// delete from previous state to mark that we've seen the existing file again
		delete(w.prev, path)
	}

	// remaining files in the prev map are the ones that are missing
	// either because they have been deleted or renamed
	for remainingPath, remainingDesc := range w.prev {
		var e loginp.FSEvent

		id := remainingDesc.FileID()
		if newDesc, renamed := newFilesByID[id]; renamed {
			e = renamedEvent(remainingPath, newDesc.Filename, *newDesc)
			delete(newFilesByName, newDesc.Filename)
			delete(newFilesByID, id)
			renamedCount++
		} else {
			e = deleteEvent(remainingPath, remainingDesc)
			removedCount++
		}
		select {
		case <-ctx.Done():
			return
		case w.events <- e:
		}
	}

	// remaining files in newFiles are newly created files
	for path, fd := range newFilesByName {
		// no need to react on empty new files
		if fd.Info.Size() == 0 {
			w.log.Debugf("file %q has no content yet, skipping", fd.Filename)
			delete(paths, path)
			continue
		}
		select {
		case <-ctx.Done():
			return
		case w.events <- createEvent(path, *fd):
			createdCount++
		}
	}

	w.log.With(
		"total", len(paths),
		"written", writtenCount,
		"truncated", truncatedCount,
		"renamed", renamedCount,
		"removed", removedCount,
		"created", createdCount,
	).Debugf("File scan complete")

	w.prev = paths
}

func createEvent(path string, fd loginp.FileDescriptor) loginp.FSEvent {
	return loginp.FSEvent{Op: loginp.OpCreate, OldPath: "", NewPath: path, Descriptor: fd}
}

func writeEvent(path string, fd loginp.FileDescriptor) loginp.FSEvent {
	return loginp.FSEvent{Op: loginp.OpWrite, OldPath: path, NewPath: path, Descriptor: fd}
}

func truncateEvent(path string, fd loginp.FileDescriptor) loginp.FSEvent {
	return loginp.FSEvent{Op: loginp.OpTruncate, OldPath: path, NewPath: path, Descriptor: fd}
}

func renamedEvent(oldPath, path string, fd loginp.FileDescriptor) loginp.FSEvent {
	return loginp.FSEvent{Op: loginp.OpRename, OldPath: oldPath, NewPath: path, Descriptor: fd}
}

func deleteEvent(path string, fd loginp.FileDescriptor) loginp.FSEvent {
	return loginp.FSEvent{Op: loginp.OpDelete, OldPath: path, NewPath: "", Descriptor: fd}
}

func (w *fileWatcher) Event() loginp.FSEvent {
	return <-w.events
}

func (w *fileWatcher) GetFiles() map[string]loginp.FileDescriptor {
	return w.scanner.GetFiles()
}

type fingerprintConfig struct {
	Enabled bool  `config:"enabled"`
	Offset  int64 `config:"offset"`
	Length  int64 `config:"length"`
}

type fileScannerConfig struct {
	ExcludedFiles []match.Matcher   `config:"exclude_files"`
	IncludedFiles []match.Matcher   `config:"include_files"`
	Symlinks      bool              `config:"symlinks"`
	RecursiveGlob bool              `config:"recursive_glob"`
	Fingerprint   fingerprintConfig `config:"fingerprint"`
}

func defaultFileScannerConfig() fileScannerConfig {
	return fileScannerConfig{
		Symlinks:      false,
		RecursiveGlob: true,
		Fingerprint: fingerprintConfig{
			Enabled: false,
			Offset:  0,
			Length:  DefaultFingerprintSize,
		},
	}
}

// fileScanner looks for files which match the patterns in paths.
// It is able to exclude files and symlinks.
type fileScanner struct {
	paths      []string
	cfg        fileScannerConfig
	log        *logp.Logger
	hasher     hash.Hash
	readBuffer []byte
}

func newFileScanner(paths []string, config fileScannerConfig) (*fileScanner, error) {
	s := fileScanner{
		paths:  paths,
		cfg:    config,
		log:    logp.NewLogger(scannerDebugKey),
		hasher: sha256.New(),
	}

	if s.cfg.Fingerprint.Enabled {
		if s.cfg.Fingerprint.Length < sha256.BlockSize {
			err := fmt.Errorf("fingerprint size %d bytes cannot be smaller than %d bytes", config.Fingerprint.Length, sha256.BlockSize)
			return nil, fmt.Errorf("error while reading configuration of fingerprint: %w", err)
		}
		s.log.Debugf("fingerprint mode enabled: offset %d, length %d", s.cfg.Fingerprint.Offset, s.cfg.Fingerprint.Length)
		s.readBuffer = make([]byte, s.cfg.Fingerprint.Length)
	}

	err := s.resolveRecursiveGlobs(config)
	if err != nil {
		return nil, err
	}
	err = s.normalizeGlobPatterns()
	if err != nil {
		return nil, err
	}

	return &s, nil
}

// resolveRecursiveGlobs expands `**` from the globs in multiple patterns
func (s *fileScanner) resolveRecursiveGlobs(c fileScannerConfig) error {
	if !c.RecursiveGlob {
		s.log.Debug("recursive glob disabled")
		return nil
	}

	s.log.Debug("recursive glob enabled")
	var paths []string
	for _, path := range s.paths {
		patterns, err := file.GlobPatterns(path, RecursiveGlobDepth)
		if err != nil {
			return err
		}
		if len(patterns) > 1 {
			s.log.Debugf("%q expanded to %#v", path, patterns)
		}
		paths = append(paths, patterns...)
	}
	s.paths = paths
	return nil
}

// normalizeGlobPatterns calls `filepath.Abs` on all the globs from config
func (s *fileScanner) normalizeGlobPatterns() error {
	paths := make([]string, len(s.paths))
	for i, path := range s.paths {
		pathAbs, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("failed to get the absolute path for %s: %w", path, err)
		}
		paths[i] = pathAbs
	}
	s.paths = paths
	return nil
}

// GetFiles returns a map of file descriptors by filenames that
// match the configured paths.
func (s *fileScanner) GetFiles() map[string]loginp.FileDescriptor {
	fdByName := map[string]loginp.FileDescriptor{}
	// used to determine if a symlink resolves in a already known target
	uniqueIDs := map[string]string{}
	// used to filter out duplicate matches
	uniqueFiles := map[string]struct{}{}
	for _, path := range s.paths {
		matches, err := filepath.Glob(path)
		if err != nil {
			s.log.Errorf("glob(%s) failed: %v", path, err)
			continue
		}

		for _, filename := range matches {
			// in case multiple globs match on the same file we filter out duplicates
			if _, knownFile := uniqueFiles[filename]; knownFile {
				continue
			}
			uniqueFiles[filename] = struct{}{}

			it, err := s.getIngestTarget(filename)
			if err != nil {
				s.log.Debugf("cannot create an ingest target for file %q: %s", filename, err)
				continue
			}

			fd, err := s.toFileDescriptor(&it)
			if errors.Is(err, errFileTooSmall) {
				s.log.Debugf("cannot start ingesting from file %q: %s", filename, err)
				continue
			}
			if err != nil {
				s.log.Warnf("cannot create a file descriptor for an ingest target %q: %s", filename, err)
				continue
			}

			fileID := fd.FileID()
			if knownFilename, exists := uniqueIDs[fileID]; exists {
				s.log.Warnf("%q points to an already known ingest target %q [%s==%s]. Skipping", fd.Filename, knownFilename, fileID, fileID)
				continue
			}
			uniqueIDs[fileID] = fd.Filename
			fdByName[filename] = fd
		}
	}

	return fdByName
}

type ingestTarget struct {
	filename         string
	originalFilename string
	symlink          bool
	info             commonfile.ExtendedFileInfo
}

func (s *fileScanner) getIngestTarget(filename string) (it ingestTarget, err error) {
	if s.isFileExcluded(filename) {
		return it, fmt.Errorf("file %q is excluded from ingestion", filename)
	}

	if !s.isFileIncluded(filename) {
		return it, fmt.Errorf("file %q is not included in ingestion", filename)
	}

	it.filename = filename
	it.originalFilename = filename

	info, err := os.Lstat(it.filename) // to determine if it's a symlink
	if err != nil {
		return it, fmt.Errorf("failed to lstat %q: %w", it.filename, err)
	}
	it.info = commonfile.ExtendFileInfo(info)

	if it.info.IsDir() {
		return it, fmt.Errorf("file %q is a directory", it.filename)
	}

	it.symlink = it.info.Mode()&os.ModeSymlink > 0

	if it.symlink {
		if !s.cfg.Symlinks {
			return it, fmt.Errorf("file %q is a symlink and they're disabled", it.filename)
		}

		// now we know it's a symlink, we stat with link resolution
		info, err := os.Stat(it.filename)
		if err != nil {
			return it, fmt.Errorf("failed to stat the symlink %q: %w", it.filename, err)
		}
		it.info = commonfile.ExtendFileInfo(info)

		it.originalFilename, err = filepath.EvalSymlinks(it.filename)
		if err != nil {
			s.log.Debugf("finding path to original file has failed %s: %+v", it.filename, err)
			it.originalFilename = it.filename
		}

		if s.isFileExcluded(it.originalFilename) {
			return it, fmt.Errorf("file %q->%q is excluded from ingestion", it.filename, it.originalFilename)
		}

		if !s.isFileIncluded(it.originalFilename) {
			return it, fmt.Errorf("file %q->%q is not included in ingestion", it.filename, it.originalFilename)
		}
	}

	return it, nil
}

func (s *fileScanner) toFileDescriptor(it *ingestTarget) (fd loginp.FileDescriptor, err error) {
	fd.Filename = it.filename
	fd.Info = it.info

	if s.cfg.Fingerprint.Enabled {
		fileSize := it.info.Size()
		// we should not open the file if we know it's too small
		minSize := s.cfg.Fingerprint.Offset + s.cfg.Fingerprint.Length
		if fileSize < minSize {
			return fd, fmt.Errorf("filesize of %q is %d bytes, expected at least %d bytes for fingerprinting: %w", fd.Filename, fileSize, minSize, errFileTooSmall)
		}

		file, err := os.Open(it.originalFilename)
		if err != nil {
			return fd, fmt.Errorf("failed to open %q for fingerprinting: %w", it.originalFilename, err)
		}
		defer file.Close()

		if s.cfg.Fingerprint.Offset != 0 {
			_, err = file.Seek(s.cfg.Fingerprint.Offset, io.SeekStart)
			if err != nil {
				return fd, fmt.Errorf("failed to seek %q for fingerprinting: %w", fd.Filename, err)
			}
		}

		s.hasher.Reset()
		lr := io.LimitReader(file, s.cfg.Fingerprint.Length)
		written, err := io.CopyBuffer(s.hasher, lr, s.readBuffer)
		if err != nil {
			return fd, fmt.Errorf("failed to compute hash for first %d bytes of %q: %w", s.cfg.Fingerprint.Length, fd.Filename, err)
		}
		if written != s.cfg.Fingerprint.Length {
			return fd, fmt.Errorf("failed to read %d bytes from %q to compute fingerprint, read only %d", written, fd.Filename, s.cfg.Fingerprint.Length)
		}

		fd.Fingerprint = hex.EncodeToString(s.hasher.Sum(nil))
	}

	return fd, nil
}

func (s *fileScanner) isFileExcluded(file string) bool {
	return len(s.cfg.ExcludedFiles) > 0 && s.matchAny(s.cfg.ExcludedFiles, file)
}

func (s *fileScanner) isFileIncluded(file string) bool {
	return len(s.cfg.IncludedFiles) == 0 || s.matchAny(s.cfg.IncludedFiles, file)
}

// matchAny checks if the text matches any of the regular expressions
func (s *fileScanner) matchAny(matchers []match.Matcher, text string) bool {
	for _, m := range matchers {
		if m.MatchString(text) {
			return true
		}
	}
	return false
}
