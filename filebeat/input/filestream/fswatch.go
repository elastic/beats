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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/elastic/go-concert/timed"
	"github.com/elastic/go-concert/unison"

	"github.com/elastic/beats/v7/filebeat/input/file"
	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
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
		prevInfo, ok := w.prev[path]
		if !ok || !loginp.SameFile(&prevInfo, &fd) {
			newFilesByName[path] = &fd
			newFilesByID[fd.FileID()] = &fd
			continue
		}

		// if the two infos belong to the same file and it has been modified
		// if the size is smaller than before, it is truncated, if bigger, it is a write event.
		// It might happen that a file is truncated and then more data is added, both
		// within the same second, this will make the reader stop, but a new one will not
		// start because the modification data is the same, to avoid this situation,
		// we also check for size changes here.
		if prevInfo.Info.ModTime() != fd.Info.ModTime() || prevInfo.Size() != info.Size() {
			var e loginp.FSEvent
			if prevInfo.Info.Size() > fd.Info.Size() || w.cfg.ResendOnModTime && prevInfo.Info.Size() == fd.Info.Size() {
				e = truncateEvent(path, fd)
				truncatedCount++
			} else {
				e = writeEvent(path, fd)
				writtenCount++
			}
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
	for path, info := range newFilesByName {
		select {
		case <-ctx.Done():
			return
		case w.events <- createEvent(path, *info):
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
	paths []string
	cfg   fileScannerConfig
	log   *logp.Logger
}

func newFileScanner(paths []string, config fileScannerConfig) (loginp.FSScanner, error) {
	if config.Fingerprint.Enabled && config.Fingerprint.Length < sha256.BlockSize {
		err := fmt.Errorf("fingerprint size %d cannot be smaller than %d", config.Fingerprint.Length, sha256.BlockSize)
		return nil, fmt.Errorf("error while reading configuration of fingerprint: %w", err)
	}

	s := fileScanner{
		paths: paths,
		cfg:   config,
		log:   logp.NewLogger(scannerDebugKey),
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
	// used to determine if a symlink resolves in a already known file
	uniqueIDs := map[string]struct{}{}

	for _, path := range s.paths {
		matches, err := filepath.Glob(path)
		if err != nil {
			s.log.Errorf("glob(%s) failed: %v", path, err)
			continue
		}

		for _, filename := range matches {
			// creating a file descriptor can be expensive, so we do the light checks first
			if s.shouldSkipFile(filename) {
				continue
			}

			fd, err := s.createFileDescriptor(filename)
			if err != nil {
				s.log.Debug("createFileDescriptor(%s) failed: %s", filename, err)
				continue
			}

			if s.checkIfKnownFile(&fd, uniqueIDs) {
				continue
			}

			fdByName[filename] = fd
		}
	}

	return fdByName
}

func (s *fileScanner) shouldSkipFile(filename string) bool {
	if s.isFileExcluded(filename) || !s.isFileIncluded(filename) {
		s.log.Debugf("Exclude file: %s", filename)
		return true
	}

	var err error

	fileInfo, err := os.Lstat(filename)
	if err != nil {
		s.log.Debugf("lstat(%s) failed: %s", filename, err)
		return true
	}

	if fileInfo.IsDir() {
		s.log.Debugf("Skipping directory: %s", filename)
		return true
	}

	originalFilename := filename

	isSymlink := fileInfo.Mode()&os.ModeSymlink > 0
	if isSymlink {
		if !s.cfg.Symlinks {
			s.log.Debugf("File %s skipped as it is a symlink", filename)
			return true
		}
		originalFilename, err = filepath.EvalSymlinks(filename)
		if err != nil {
			s.log.Debugf("Finding path to original file has failed %s: %+v", filename, err)
			return true
		}
		// Check if original file is included to make sure we are not reading from
		// unwanted files.
		if s.isFileExcluded(originalFilename) || !s.isFileIncluded(originalFilename) {
			s.log.Debugf("Exclude original file: %s", filename)
			return true
		}
	}

	// if fingerprinting is enabled and the file is too small
	// for computing a fingerprint, we have to skip it
	// until it grows up and becomes a real file that's worth our attention.
	if s.cfg.Fingerprint.Enabled {
		var (
			fi  os.FileInfo
			err error
		)

		// the previous Lstat does not follow symlinks, so we have to stat again
		if !isSymlink {
			fi = fileInfo
		} else {
			fi, err = os.Stat(originalFilename)
			if err != nil {
				s.log.Debugf("stat(%s) failed: %s", filename, err)
				return true
			}
		}

		fileSize := fi.Size()
		minSize := s.cfg.Fingerprint.Offset + s.cfg.Fingerprint.Length
		if fileSize < minSize {
			s.log.Debugf("filesize - %d, expected at least - %d for fingerprinting", fileSize, minSize)
			return true
		}
	}

	return false
}

func (s *fileScanner) createFileDescriptor(filename string) (fd loginp.FileDescriptor, err error) {
	fd.Filename = filename
	fd.Info, err = os.Stat(filename)
	if err != nil {
		return fd, fmt.Errorf("failed to stat %q: %w", filename, err)
	}

	if s.cfg.Fingerprint.Enabled {
		h := sha256.New()
		file, err := os.Open(filename)
		if err != nil {
			return fd, fmt.Errorf("failed to open %q for fingerprinting: %w", filename, err)
		}
		defer file.Close()

		if s.cfg.Fingerprint.Offset != 0 {
			_, err = file.Seek(s.cfg.Fingerprint.Offset, io.SeekStart)
			if err != nil {
				return fd, fmt.Errorf("failed to seek %q for fingerprinting: %w", filename, err)
			}
		}

		r := io.LimitReader(file, s.cfg.Fingerprint.Length)
		buf := make([]byte, h.BlockSize())
		_, err = io.CopyBuffer(h, r, buf)
		if err != nil {
			return fd, fmt.Errorf("failed to compute hash for first %d bytes of %q: %w", s.cfg.Fingerprint.Length, filename, err)
		}

		fd.Fingerprint = hex.EncodeToString(h.Sum(nil))
	}

	return fd, nil
}

// If symlink is enabled, this function checks if the original file is not part of same input
// If original is harvested by other input, states will potentially overwrite each other
func (s *fileScanner) checkIfKnownFile(fd *loginp.FileDescriptor, uniqueIDs map[string]struct{}) bool {
	// if symlinks are not enabled there is no point for this check
	// since filenames are already unique in the initial set and symlinks are excluded from it
	if !s.cfg.Symlinks {
		return false
	}

	fileID := fd.FileID()
	if _, exists := uniqueIDs[fileID]; exists {
		s.log.Info("Symlink %q points to a known file %q. Skipping file: %q", fd.Filename, fd.Info.Name())
		return true
	}
	uniqueIDs[fileID] = struct{}{}

	return false
}

func (s *fileScanner) isFileExcluded(file string) bool {
	return len(s.cfg.ExcludedFiles) > 0 && s.matchAny(s.cfg.ExcludedFiles, file)
}

func (s *fileScanner) isFileIncluded(file string) bool {
	if len(s.cfg.IncludedFiles) == 0 {
		return true
	}
	return s.matchAny(s.cfg.IncludedFiles, file)
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
