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
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/elastic/beats/v7/filebeat/input/file"
	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/match"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/go-concert/timed"
	"github.com/elastic/go-concert/unison"
)

const (
	recursiveGlobDepth = 8
	scannerName        = "scanner"
	watcherDebugKey    = "file_watcher"
)

var watcherFactories = map[string]watcherFactory{
	scannerName: newScannerWatcher,
}

type watcherFactory func(paths []string, cfg *common.Config) (loginp.FSWatcher, error)

// fileScanner looks for files which match the patterns in paths.
// It is able to exclude files and symlinks.
type fileScanner struct {
	paths         []string
	excludedFiles []match.Matcher
	includedFiles []match.Matcher
	symlinks      bool

	log *logp.Logger
}

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
	interval        time.Duration
	resendOnModTime bool
	prev            map[string]os.FileInfo
	scanner         loginp.FSScanner
	log             *logp.Logger
	events          chan loginp.FSEvent
}

func newFileWatcher(paths []string, ns *common.ConfigNamespace) (loginp.FSWatcher, error) {
	if ns == nil {
		return newScannerWatcher(paths, common.NewConfig())
	}

	watcherType := ns.Name()
	f, ok := watcherFactories[watcherType]
	if !ok {
		return nil, fmt.Errorf("no such file watcher: %s", watcherType)
	}

	return f(paths, ns.Config())
}

func newScannerWatcher(paths []string, c *common.Config) (loginp.FSWatcher, error) {
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
		log:             logp.NewLogger(watcherDebugKey),
		interval:        config.Interval,
		resendOnModTime: config.ResendOnModTime,
		prev:            make(map[string]os.FileInfo, 0),
		scanner:         scanner,
		events:          make(chan loginp.FSEvent),
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

	timed.Periodic(ctx, w.interval, func() error {
		w.watch(ctx)

		return nil
	})
}

func (w *fileWatcher) watch(ctx unison.Canceler) {
	w.log.Info("Start next scan")

	paths := w.scanner.GetFiles()

	newFiles := make(map[string]os.FileInfo)

	for path, info := range paths {

		prevInfo, ok := w.prev[path]
		if !ok {
			newFiles[path] = paths[path]
			continue
		}

		if prevInfo.ModTime() != info.ModTime() {
			if prevInfo.Size() > info.Size() || w.resendOnModTime && prevInfo.Size() == info.Size() {
				select {
				case <-ctx.Done():
					return
				case w.events <- truncateEvent(path, info):
				}
			} else {
				select {
				case <-ctx.Done():
					return
				case w.events <- writeEvent(path, info):
				}
			}
		}

		// delete from previous state, as we have more up to date info
		delete(w.prev, path)
	}

	// remaining files are in the prev map are the ones that are missing
	// either because they have been deleted or renamed
	for removedPath, removedInfo := range w.prev {
		for newPath, newInfo := range newFiles {
			if os.SameFile(removedInfo, newInfo) {
				select {
				case <-ctx.Done():
					return
				case w.events <- renamedEvent(removedPath, newPath, newInfo):
					delete(newFiles, newPath)
					goto CHECK_NEXT_REMOVED
				}
			}
		}

		select {
		case <-ctx.Done():
			return
		case w.events <- deleteEvent(removedPath, removedInfo):
		}
	CHECK_NEXT_REMOVED:
	}

	// remaining files in newFiles are new
	for path, info := range newFiles {
		select {
		case <-ctx.Done():
			return
		case w.events <- createEvent(path, info):
		}
	}

	w.log.Debugf("Found %d paths", len(paths))
	w.prev = paths
}

func createEvent(path string, fi os.FileInfo) loginp.FSEvent {
	return loginp.FSEvent{Op: loginp.OpCreate, OldPath: "", NewPath: path, Info: fi}
}

func writeEvent(path string, fi os.FileInfo) loginp.FSEvent {
	return loginp.FSEvent{Op: loginp.OpWrite, OldPath: path, NewPath: path, Info: fi}
}

func truncateEvent(path string, fi os.FileInfo) loginp.FSEvent {
	return loginp.FSEvent{Op: loginp.OpTruncate, OldPath: path, NewPath: path, Info: fi}
}

func renamedEvent(oldPath, path string, fi os.FileInfo) loginp.FSEvent {
	return loginp.FSEvent{Op: loginp.OpRename, OldPath: oldPath, NewPath: path, Info: fi}
}

func deleteEvent(path string, fi os.FileInfo) loginp.FSEvent {
	return loginp.FSEvent{Op: loginp.OpDelete, OldPath: path, NewPath: "", Info: fi}
}

func (w *fileWatcher) Event() loginp.FSEvent {
	return <-w.events
}

func (w *fileWatcher) GetFiles() map[string]os.FileInfo {
	return w.scanner.GetFiles()
}

type fileScannerConfig struct {
	ExcludedFiles []match.Matcher `config:"exclude_files"`
	IncludedFiles []match.Matcher `config:"include_files"`
	Symlinks      bool            `config:"symlinks"`
	RecursiveGlob bool            `config:"recursive_glob"`
}

func defaultFileScannerConfig() fileScannerConfig {
	return fileScannerConfig{
		Symlinks:      false,
		RecursiveGlob: true,
	}
}

func newFileScanner(paths []string, cfg fileScannerConfig) (loginp.FSScanner, error) {
	fs := fileScanner{
		paths:         paths,
		excludedFiles: cfg.ExcludedFiles,
		includedFiles: cfg.IncludedFiles,
		symlinks:      cfg.Symlinks,
		log:           logp.NewLogger(scannerName),
	}
	err := fs.resolveRecursiveGlobs(cfg)
	if err != nil {
		return nil, err
	}
	err = fs.normalizeGlobPatterns()
	if err != nil {
		return nil, err
	}

	return &fs, nil
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
		patterns, err := file.GlobPatterns(path, recursiveGlobDepth)
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
	var paths []string
	for _, path := range s.paths {
		pathAbs, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("failed to get the absolute path for %s: %v", path, err)
		}
		paths = append(paths, pathAbs)
	}
	s.paths = paths
	return nil
}

// GetFiles returns a map of files and fileinfos which
// match the configured paths.
func (s *fileScanner) GetFiles() map[string]os.FileInfo {
	pathInfo := map[string]os.FileInfo{}

	for _, path := range s.paths {
		matches, err := filepath.Glob(path)
		if err != nil {
			s.log.Errorf("glob(%s) failed: %v", path, err)
			continue
		}

		for _, file := range matches {
			if s.shouldSkipFile(file) {
				continue
			}

			// If symlink is enabled, it is checked that original is not part of same input
			// If original is harvested by other input, states will potentially overwrite each other
			if s.isOriginalAndSymlinkConfigured(file, pathInfo) {
				continue
			}

			fileInfo, err := os.Stat(file)
			if err != nil {
				s.log.Debug("stat(%s) failed: %s", file, err)
				continue
			}
			pathInfo[file] = fileInfo
		}
	}

	return pathInfo
}

func (s *fileScanner) shouldSkipFile(file string) bool {
	if s.isFileExcluded(file) || !s.isFileIncluded(file) {
		s.log.Debugf("Exclude file: %s", file)
		return true
	}

	fileInfo, err := os.Lstat(file)
	if err != nil {
		s.log.Debugf("lstat(%s) failed: %s", file, err)
		return true
	}

	if fileInfo.IsDir() {
		s.log.Debugf("Skipping directory: %s", file)
		return true
	}

	isSymlink := fileInfo.Mode()&os.ModeSymlink > 0
	if isSymlink && !s.symlinks {
		s.log.Debugf("File %s skipped as it is a symlink", file)
		return true
	}

	originalFile, err := filepath.EvalSymlinks(file)
	if err != nil {
		s.log.Debugf("finding path to original file has failed %s: %+v", file, err)
		return true
	}
	// Check if original file is included to make sure we are not reading from
	// unwanted files.
	if s.isFileExcluded(originalFile) || !s.isFileIncluded(originalFile) {
		s.log.Debugf("Exclude original file: %s", file)
		return true
	}

	return false
}

func (s *fileScanner) isOriginalAndSymlinkConfigured(file string, paths map[string]os.FileInfo) bool {
	if s.symlinks {
		fileInfo, err := os.Stat(file)
		if err != nil {
			s.log.Debugf("stat(%s) failed: %s", file, err)
			return false
		}

		for _, finfo := range paths {
			if os.SameFile(finfo, fileInfo) {
				s.log.Info("Same file found as symlink and original. Skipping file: %s (as it same as %s)", file, finfo.Name())
				return true
			}
		}
	}
	return false
}

func (s *fileScanner) isFileExcluded(file string) bool {
	return len(s.excludedFiles) > 0 && s.matchAny(s.excludedFiles, file)
}

func (s *fileScanner) isFileIncluded(file string) bool {
	if len(s.includedFiles) == 0 {
		return true
	}
	return s.matchAny(s.includedFiles, file)
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
