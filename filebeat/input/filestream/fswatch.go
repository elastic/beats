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
	"sync"
	"sync/atomic"
	"time"

	"github.com/elastic/go-concert/unison"

	"github.com/elastic/beats/v7/filebeat/input/file"
	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	commonfile "github.com/elastic/beats/v7/libbeat/common/file"
	"github.com/elastic/beats/v7/libbeat/common/match"
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
	errFileEmpty    = errors.New("file is empty")
)

// fileWatcherConfig is the prospector.scanner configuration
type fileWatcherConfig struct {
	// Interval is the time between two scans.
	Interval time.Duration `config:"check_interval"`
	// ResendOnModTime  if a file has been changed according to modtime but the size is the same
	// it is still considered truncation.
	ResendOnModTime bool `config:"resend_on_touch"`
	// Scanner is the configuration of the scanner.
	Scanner fileScannerConfig `config:",inline"`
	// SendNotChanged sends an event even when the file has not changed
	// This setting is for internal use only
	SendNotChanged bool `config:"-"`
}

// fileWatcher gets the list of files from a FSWatcher and creates events by
// comparing the files between its last two runs.
type fileWatcher struct {
	cfg              fileWatcherConfig
	prev             map[string]loginp.FileDescriptor
	scanner          loginp.FSScanner
	log              *logp.Logger
	events           chan loginp.FSEvent
	notifyChan       chan loginp.HarvesterStatus
	fileIdentifier   fileIdentifier
	sourceIdentifier *loginp.SourceIdentifier

	// growingFingerprint indicates that the growing fingerprint mode is active.
	// When true, prefix-based rename detection is used as a fallback
	// for files whose fingerprint grew between scans.
	growingFingerprint bool

	// closedHarvesters is a map of harvester ID to the current
	// offset of the file
	closedHarvesters map[string]int64
	// closedHarvestersMutex controls access to closedHarvesters
	closedHarvestersMutex sync.Mutex
}

// Ensure fileWatcher implements loginp.FSWatcher
var _ loginp.FSWatcher = &fileWatcher{}

func newFileWatcher(
	logger *logp.Logger,
	paths []string,
	config fileWatcherConfig,
	compression string,
	sendNotChanged bool,
	fi fileIdentifier,
	srci *loginp.SourceIdentifier,
) (*fileWatcher, error) {

	config.SendNotChanged = sendNotChanged
	scanner, err := newFileScanner(logger, paths, config.Scanner, compression)
	if err != nil {
		return nil, err
	}

	return &fileWatcher{
		log:              logger.Named(watcherDebugKey),
		cfg:              config,
		prev:             make(map[string]loginp.FileDescriptor, 0),
		scanner:          scanner,
		events:           make(chan loginp.FSEvent),
		closedHarvesters: map[string]int64{},
		// notifyChan is a buffered channel to prevent the harvester from
		// blocking while waiting for the fileWatcher to read from the channel
		notifyChan:         make(chan loginp.HarvesterStatus, 5), // magic number
		fileIdentifier:     fi,
		sourceIdentifier:   srci,
		growingFingerprint: config.Scanner.Fingerprint.Growing,
	}, nil
}

func defaultFileWatcherConfig() fileWatcherConfig {
	return fileWatcherConfig{
		Interval:        10 * time.Second,
		ResendOnModTime: false,
		Scanner:         defaultFileScannerConfig(),
		SendNotChanged:  false,
	}
}

func (w *fileWatcher) NotifyChan() chan loginp.HarvesterStatus {
	return w.notifyChan
}

func (w *fileWatcher) Run(ctx unison.Canceler) {
	defer close(w.events)

	// run initial scan before starting regular
	w.watch(ctx)

	// Read from notifyChan in a separate goroutine becase
	// there are cases when w.watch can take minutes or even
	// hours, so we do not want to block the harvesters
	go func() {
		for {
			select {
			case evt := <-w.notifyChan:
				w.processNotification(evt)
			case <-ctx.Done():
				return
			}
		}
	}()

	tick := time.Tick(w.cfg.Interval)
	for {
		select {
		case <-tick:
			w.watch(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (w *fileWatcher) processNotification(evt loginp.HarvesterStatus) {
	w.log.Debugf("Harvester Closed notification received. ID: %s, Size: %d", evt.ID, evt.Size)
	w.closedHarvestersMutex.Lock()
	w.closedHarvesters[evt.ID] = evt.Size
	w.closedHarvestersMutex.Unlock()
}

func (w *fileWatcher) watch(ctx unison.Canceler) {
	w.log.Debug("Start next scan")

	// file identity is updated in GetFiles
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
		// srcID is the file identity, it is the same value used to identify
		// the harvester and as registry key for the file's state
		srcID := w.getFileIdentity(fd)

		// if the scanner found a new path or an existing path
		// with a different file, it is a new file
		prevDesc, ok := w.prev[path]
		sfd := fd // to avoid memory aliasing
		if !ok || !loginp.SameFile(&prevDesc, &sfd) {
			newFilesByName[path] = &sfd
			newFilesByID[fd.FileID()] = &sfd
			continue
		}

		// If we got notifications about harvesters being closed, update
		// the state accordingly.
		//
		// This is used to prevent a sort of race condition:
		// When the reader/harvester reaches EOF, it blocks on a backoff,
		// if during this time [logFile.shouldBeClosed] is called, marks the
		// file as inactive and closes the reader context, once the backoff
		// time expires the reader and harvester are closed without ingesting
		// any more data.
		//
		// If the [fileWatcher] sends a write event while the harvester was blocked
		// no new harvester is started because one is already running, however the
		// [fileWatcher] updates its internal state and won't send write events until
		// more data is added to the file.
		//
		// This can cause some lines to be missed because the harvester closed
		// and the write event was lost.
		//
		// To prevent this from happening we get notified the offset of the file
		// (data ingested) when the harvester closes. If we have this data we
		// update our state to the same as the harvester, therefore starting
		// a new harvester if needed.
		w.closedHarvestersMutex.Lock()
		if size, harvesterClosed := w.closedHarvesters[srcID]; harvesterClosed {
			w.log.Debugf("Updating previous state because harvester was closed. '%s': %d", srcID, size)
			prevDesc.SetBytesIngested(size)
		}
		w.closedHarvestersMutex.Unlock()

		var e loginp.FSEvent
		switch {
		// the new size is smaller, the file was truncated
		case prevDesc.Info.Size() > fd.Info.Size():
			e = truncateEvent(path, fd, srcID)
			truncatedCount++

		// the size is the same, timestamps are different, the file was touched
		case prevDesc.Info.Size() == fd.Info.Size() && prevDesc.Info.ModTime() != fd.Info.ModTime():
			if w.cfg.ResendOnModTime {
				e = truncateEvent(path, fd, srcID)
				truncatedCount++
			}

		// the new size is larger, something was written.
		// If a harvester for this file was closed recently,
		// we use its state instead of the one we have cached.
		case prevDesc.SizeOrBytesIngested() < fd.Info.Size():
			e = writeEvent(path, fd, srcID)
			writtenCount++

		default:
			// For the delete feature we need to run the harvester for
			// files that have not changed until they're deleted.
			if w.cfg.SendNotChanged {
				e = notChangedEvent(path, fd, srcID)
			}
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
		// Delete used state from closedHarvesters
		w.closedHarvestersMutex.Lock()
		delete(w.closedHarvesters, srcID)
		w.closedHarvestersMutex.Unlock()
	}

	// Remaining files in the prev map are missing from this scan — either
	// deleted or renamed. Three rename-detection passes follow, in order:
	//
	//   1. Exact-FileID rename match — works for every identity including
	//      static fingerprint. Catches a plain rename where the file's
	//      content (and so its fingerprint) is unchanged.
	//   2. Prefix-match rename detection (Enhanced Fingerprint / growing
	//      mode only) — catches rename + content growth in the same scan.
	//   3. Unmatched-leftover emission — anything still in w.prev becomes
	//      OpDelete, anything still in newFilesByName becomes OpCreate.

	// Exact-FileID rename match: For growing mode, also accumulate the
	// short-fingerprint index from prev entries that did NOT get an exact
	// match — they are the candidates for the next (prefix-match) pass.
	var shortFingerprints *shortFingerprintSet
	if w.growingFingerprint {
		shortFingerprints = newShortFingerprintSet()
	}

	for remainingPath, remainingDesc := range w.prev {
		newDesc, renamed := newFilesByID[remainingDesc.FileID()]

		switch {
		// Exact-FileID rename match
		case renamed:
			srcID := w.getFileIdentity(remainingDesc)
			select {
			case <-ctx.Done():
				return
			case w.events <- renamedEvent(
				remainingPath, newDesc.Filename, *newDesc, srcID):
				renamedCount++
			}

			delete(newFilesByName, newDesc.Filename)
			delete(newFilesByID, remainingDesc.FileID())
			delete(w.prev, remainingPath)

		// If it isn't an exact match, make it a candidate for prefix-match.
		// growingRawFingerprint keeps only still-growing predecessors in the
		// index; completed entries match by their SHA-256 identity instead.
		case w.growingFingerprint:
			if raw := growingRawFingerprint(remainingDesc); raw != "" {
				shortFingerprints.Add(remainingPath, raw, remainingPath)
			}
		}
	}

	// Growing fingerprint: prefix-match rename detection.
	// For each new file that didn't match exactly, look for an unmatched prev entry whose raw
	// fingerprint is a STRICT PREFIX of the new file's raw material. The same file must be renamed
	// AND grown across the threshold in a single scan.
	//
	// The match is deliberately restricted to a new file whose fingerprint is Complete(): a short
	// raw prefix alone is too weak to prove identity, so a distinct file that appears in the same
	// scan a tracked file vanished and merely shares a leading header would otherwise be classified
	// as a rename.
	if shortFingerprints.Len() > 0 {
		type prefixMatch struct {
			oldPath string
			newPath string
			newDesc *loginp.FileDescriptor
		}
		var matches []prefixMatch

		for newPath, newDesc := range newFilesByName {
			// Only a completed fingerprint is strong enough to justify a cross-path rename match.
			if !newDesc.Fingerprint.Complete() {
				continue
			}
			oldPath, _, found := shortFingerprints.FindPrefixMatch(newDesc.Fingerprint.Raw, "")
			if found {
				matches = append(matches, prefixMatch{oldPath, newPath, newDesc})
				shortFingerprints.Remove(oldPath)
			}
		}

		for _, m := range matches {
			remainingDesc := w.prev[m.oldPath]
			srcID := w.getFileIdentity(remainingDesc)
			select {
			case <-ctx.Done():
				return
			case w.events <- renamedEvent(m.oldPath, m.newPath, *m.newDesc, srcID):
				renamedCount++
			}

			delete(newFilesByName, m.newPath)
			delete(newFilesByID, m.newDesc.FileID())
			delete(w.prev, m.oldPath)
		}
	}

	// Unmatched-leftover deletes: prev files that weren't matched by either
	// the exact-FileID or the prefix-match rename pass are genuinely gone.
	for remainingPath, remainingDesc := range w.prev {
		srcID := w.getFileIdentity(remainingDesc)
		select {
		case <-ctx.Done():
			return
		case w.events <- deleteEvent(remainingPath, remainingDesc, srcID):
			removedCount++
		}

		w.closedHarvestersMutex.Lock()
		delete(w.closedHarvesters, srcID)
		w.closedHarvestersMutex.Unlock()
	}

	// Unmatched-leftover creates: new files left over after both rename
	// passes are genuinely new.
	for path, fd := range newFilesByName {
		select {
		case <-ctx.Done():
			return
		case w.events <- createEvent(path, *fd, w.getFileIdentity(*fd)):
			createdCount++
		}
	}

	w.log.Debugw("File scan complete",
		"total", len(paths),
		"written", writtenCount,
		"truncated", truncatedCount,
		"renamed", renamedCount,
		"removed", removedCount,
		"created", createdCount,
	)

	// In growing mode, do a single pass over this scan's descriptors to:
	//   - Drop the bridging raw header from completed descriptors before they
	//     go into w.prev: a completed file is matched by its SHA-256 identity,
	//     so retaining the full header for every tracked file would bloat
	//     w.prev. The events above already carried the full descriptor, so
	//     trimming here only affects retained state.
	//   - Refresh the scanner's completedFingerprints set so its next scan can
	//     skip recomputing the (now redundant) raw header. fileWatcher.watch is
	//     the only writer of that set — see the completedFingerprints field for
	//     why the prospector's enumeration scans must not seed it.
	if w.growingFingerprint {
		completed := make(map[string]struct{}, len(paths))
		for p, fd := range paths {
			if fd.Fingerprint.Complete() {
				completed[p] = struct{}{}
				if fd.Fingerprint.Raw != "" {
					fd.Fingerprint.Raw = ""
					paths[p] = fd
				}
			}
		}
		if fs, ok := w.scanner.(*fileScanner); ok {
			fs.completedFingerprints = completed
		}
	}

	w.prev = paths
}

// getFileIdentity mimics the same algorithm used by the harvester to generate
// the file identity to any given file.
// See 'startHarvester' on internal/input-logfile/harvester.go.
func (w *fileWatcher) getFileIdentity(d loginp.FileDescriptor) string {
	src := w.fileIdentifier.GetSource(loginp.FSEvent{Descriptor: d})
	return w.sourceIdentifier.ID(src)
}

func createEvent(path string, fd loginp.FileDescriptor, srcID string) loginp.FSEvent {
	return loginp.FSEvent{Op: loginp.OpCreate, OldPath: "", NewPath: path, Descriptor: fd, SrcID: srcID}
}

func writeEvent(path string, fd loginp.FileDescriptor, srcID string) loginp.FSEvent {
	return loginp.FSEvent{Op: loginp.OpWrite, OldPath: path, NewPath: path, Descriptor: fd, SrcID: srcID}
}

func truncateEvent(path string, fd loginp.FileDescriptor, srcID string) loginp.FSEvent {
	return loginp.FSEvent{Op: loginp.OpTruncate, OldPath: path, NewPath: path, Descriptor: fd, SrcID: srcID}
}

func renamedEvent(oldPath, path string, fd loginp.FileDescriptor, srcID string) loginp.FSEvent {
	return loginp.FSEvent{Op: loginp.OpRename, OldPath: oldPath, NewPath: path, Descriptor: fd, SrcID: srcID}
}

func deleteEvent(path string, fd loginp.FileDescriptor, srcID string) loginp.FSEvent {
	return loginp.FSEvent{Op: loginp.OpDelete, OldPath: path, NewPath: "", Descriptor: fd, SrcID: srcID}
}

func notChangedEvent(path string, fd loginp.FileDescriptor, srcID string) loginp.FSEvent {
	return loginp.FSEvent{Op: loginp.OpNotChanged, OldPath: path, NewPath: path, Descriptor: fd, SrcID: srcID}
}

func (w *fileWatcher) Event() loginp.FSEvent {
	return <-w.events
}

// GetFiles runs a one-off enumeration scan for the prospector's Init and
// TakeOver phases. It is side-effect free: unlike the watch loop it does not
// advance the scanner's completedFingerprints set, so these pre-watch scans
// cannot suppress the bridging raw header a still-growing entry needs to
// migrate its registry key after a restart.
func (w *fileWatcher) GetFiles() map[string]loginp.FileDescriptor {
	return w.scanner.GetFiles()
}

type fingerprintConfig struct {
	Enabled bool  `config:"enabled"`
	Offset  int64 `config:"offset"`
	Length  int64 `config:"length"`
	// Growing enables Enhanced Fingerprint behaviour: files smaller than
	// Offset+Length are tracked using the raw bytes from Offset to the file's
	// end (hex-encoded). When a file reaches the threshold, its registry key
	// migrates to the same SHA-256 hex the static fingerprint produces, so
	// existing static-fingerprint state is preserved.
	//
	// Not user-configurable here: the YAML key under prospector.scanner.fingerprint
	// is silently ignored. The user-facing knob is file_identity.fingerprint.growing;
	// normalizeConfig in input.go propagates it here.
	Growing bool `config:"-"`
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
			Enabled: true,
			Offset:  0,
			Length:  DefaultFingerprintSize,
			// false by default: the file identity config will set it to true if
			// fingerprint is used
			Growing: false,
		},
	}
}

// fileScanner looks for files which match the patterns in paths.
// It is able to exclude files and symlinks.
type fileScanner struct {
	smallFilesWarned atomic.Bool
	paths            []string
	cfg              fileScannerConfig
	log              *logp.Logger
	hasher           hash.Hash
	readBuffer       []byte
	compression      string
	// completedFingerprints holds the paths whose fingerprint was already a
	// final SHA-256 on the previous watch-loop scan (growing mode only). The
	// bridging raw header is only useful on the scan a file crosses the
	// threshold, so for paths in this set toFileDescriptor skips recomputing it.
	// GetFiles itself is pure with respect to this set: only fileWatcher.watch
	// advances it (after each scan), so the enumeration-only scans the
	// prospector runs in Init/TakeOver cannot suppress the header a
	// still-growing entry needs to migrate across a restart.
	completedFingerprints map[string]struct{}
}

func newFileScanner(logger *logp.Logger, paths []string, config fileScannerConfig, compression string) (*fileScanner, error) {
	s := fileScanner{
		paths:       paths,
		cfg:         config,
		log:         logger.Named(scannerDebugKey),
		hasher:      sha256.New(),
		compression: compression,
	}

	if s.cfg.Fingerprint.Enabled {
		if s.cfg.Fingerprint.Length < sha256.BlockSize {
			err := fmt.Errorf("fingerprint size %d bytes cannot be smaller than %d bytes", config.Fingerprint.Length, sha256.BlockSize)
			return nil, fmt.Errorf("error while reading configuration of fingerprint: %w", err)
		}
		s.log.Debugf("fingerprint mode enabled: offset %d, length %d, growing %t",
			s.cfg.Fingerprint.Offset, s.cfg.Fingerprint.Length, s.cfg.Fingerprint.Growing)
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
				if !errors.Is(err, errFileEmpty) {
					s.log.Debugf("cannot create an ingest target for file %q: %s", filename, err)
				}
				continue
			}

			fd, err := s.toFileDescriptor(&it)
			if errors.Is(err, errFileTooSmall) {
				if s.smallFilesWarned.CompareAndSwap(false, true) {
					s.log.Warnf("ingestion from some files will be delayed, files need to be at "+
						"least %d in size for ingestion to start. To change this "+
						"behaviour set 'prospector.scanner.fingerprint.length' and "+
						"'prospector.scanner.fingerprint.offset'. "+
						"Enable debug logging to see all file names of delayed files.",
						s.cfg.Fingerprint.Offset+s.cfg.Fingerprint.Length)
				}
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
	if info.IsDir() {
		return it, fmt.Errorf("file %q is a directory", it.filename)
	}

	symlink := info.Mode()&os.ModeSymlink > 0

	// we don't need to process empty files
	if !symlink && info.Size() == 0 {
		return it, errFileEmpty
	}

	it.info = commonfile.ExtendFileInfo(info)
	it.symlink = symlink

	if it.symlink {
		if !s.cfg.Symlinks {
			return it, fmt.Errorf("file %q is a symlink and they're disabled", it.filename)
		}

		// now we know it's a symlink, we stat with link resolution
		info, err := os.Stat(it.filename)
		if err != nil {
			return it, fmt.Errorf("failed to stat the symlink %q: %w", it.filename, err)
		}
		// we don't need to process empty files
		if info.Size() == 0 {
			return it, errFileEmpty
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

// toFileDescriptor builds a FileDescriptor for the given ingest target.
// With fingerprinting enabled, it computes the file's identity according to
// the threshold rules:
//
//   - !Enabled: no fingerprint; FileID falls back to OS state.
//   - dataSize <= offset: file is too small to read anything from offset;
//     return errFileTooSmall.
//   - dataSize >= offset+length: read bytes[offset:offset+length] and hash
//     with SHA-256 (FingerprintID.Sum, so Complete() is true). In growing mode
//     the raw header bytes are also carried in FingerprintID.Raw so the one-time
//     crossing to the SHA-256 identity can be prefix-matched against a still
//     growing predecessor.
//   - dataSize in (offset, offset+length) under growing mode: read
//     bytes[offset:dataSize] and carry its hex as FingerprintID.Raw, leaving
//     Sum empty so Complete() is false.
//   - dataSize in (offset, offset+length) under non-growing mode: return
//     errFileTooSmall (today's static-fingerprint behaviour).
//
// GZIP is honoured: all reads are on the decompressed stream.
func (s *fileScanner) toFileDescriptor(it *ingestTarget) (fd loginp.FileDescriptor, err error) {
	fd.Filename = it.filename
	fd.Info = it.info

	if !s.cfg.Fingerprint.Enabled {
		return fd, nil
	}

	offset := s.cfg.Fingerprint.Offset
	length := s.cfg.Fingerprint.Length
	threshold := offset + length

	// opener is used to open the file only once
	opener := struct {
		Open func() (*os.File, error)
		f    *os.File
	}{}
	opener.Open = func() (*os.File, error) {
		if opener.f != nil {
			return opener.f, nil
		}

		opener.f, err = os.Open(it.originalFilename)
		if err != nil {
			return nil, fmt.Errorf("fileScanner: failed to open %q to create FileDescriptor: %w", it.originalFilename, err)
		}
		return opener.f, err
	}

	defer func() {
		if opener.f != nil {
			opener.f.Close()
		}
	}()

	switch s.compression {
	case CompressionNone:
		// fd.GZIP stays false
	case CompressionGZIP:
		fd.GZIP = true
	case CompressionAuto:
		osFile, err := opener.Open()
		if err != nil {
			return fd, fmt.Errorf("fileScanner: failed to open %q to create FileDescriptor: %w", it.originalFilename, err)
		}

		fd.GZIP, err = IsGZIP(osFile)
		if err != nil {
			return fd, fmt.Errorf("failed to check if %q is gzip: %w",
				it.originalFilename, err)
		}
	}

	// Fast path for non-GZIP files we know the size from lstat and can
	// reject too-small files in static mode without opening the file. This
	// preserves the no-open guarantee for static fingerprint on
	// unreadable/permission-denied small files.
	if !fd.GZIP {
		if !s.cfg.Fingerprint.Growing && it.info.Size() < threshold {
			return fd, fmt.Errorf(
				"filesize of %q is %d bytes, expected at least %d bytes for fingerprinting: %w",
				fd.Filename, it.info.Size(), threshold, errFileTooSmall)
		}
		// size <= offset we cannot read anything from the offset, regardless of
		// mode.
		if it.info.Size() <= offset {
			return fd, fmt.Errorf(
				"filesize of %q is %d bytes, less than fingerprint offset %d: %w",
				fd.Filename, it.info.Size(), offset, errFileTooSmall)
		}
	}

	// Wrap the open file (plain or GZIP) so subsequent reads/seeks operate
	// on the decompressed stream when applicable.
	var file File
	if fd.GZIP {
		osFile, err := opener.Open()
		if err != nil {
			return fd, fmt.Errorf("fileScanner: failed to open %q to create FileDescriptor: %w", it.originalFilename, err)
		}

		// Check if there is enough *decompressed* data for fingerprint
		file, err = newGzipSeekerReader(osFile, int(threshold))
		if err != nil {
			return fd, fmt.Errorf("failed to create gzip seeker: %w", err)
		}
		defer file.Close()
	} else {
		osFile, err := opener.Open()
		if err != nil {
			return fd, fmt.Errorf("fileScanner: failed to open %q to create FileDescriptor: %w", it.originalFilename, err)
		}
		file = newPlainFile(osFile)
	}

	// Seek to offset (for both growing and static paths).
	if offset != 0 {
		if _, err := file.Seek(offset, io.SeekStart); err != nil {
			// Seek past EOF (file smaller than offset) — untrackable.
			if errors.Is(err, io.EOF) {
				return fd, fmt.Errorf(
					"file %q is smaller than fingerprint offset %d: %w",
					fd.Filename, offset, errFileTooSmall)
			}
			return fd, fmt.Errorf("failed to seek %q to offset: %w", fd.Filename, err)
		}
	}

	// Read up to `length` bytes from offset into the read buffer.
	n, err := io.ReadFull(file, s.readBuffer[:length])
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return fd, fmt.Errorf("failed to read %q for fingerprinting: %w", fd.Filename, err)
	}

	// Growing fingerprint path
	if int64(n) < length {
		// File is below threshold: bytes available from offset is n < length.
		if !s.cfg.Fingerprint.Growing {
			return fd, fmt.Errorf(
				"filesize of %q is %d bytes (read %d from offset), expected at least %d bytes for fingerprinting: %w",
				fd.Filename, it.info.Size(), n, length, errFileTooSmall)
		}

		if n == 0 {
			// nothing readable from offset — also untrackable
			return fd, fmt.Errorf(
				"file %q has no bytes available from offset %d: %w",
				fd.Filename, offset, errFileTooSmall)
		}

		// Growing mode small file: hex of bytes[offset:offset+n].
		fd.Fingerprint = loginp.FingerprintID{Raw: hex.EncodeToString(s.readBuffer[:n])}

		return fd, nil
	}

	// File at or above threshold: compute SHA-256 of bytes[offset:offset+length].
	s.hasher.Reset()
	s.hasher.Write(s.readBuffer[:length])
	fd.Fingerprint = loginp.FingerprintID{
		Sum: hex.EncodeToString(s.hasher.Sum(nil)),
	}

	// In growing mode the raw header is carried alongside the SHA-256 so the
	// one-time transition to the final identity can be prefix-matched against a
	// still-growing predecessor (in-place growth or rename+grow). It is only
	// needed on the scan a file crosses the threshold: a path the watch loop
	// already saw complete on its previous scan has no growing predecessor left
	// to bridge, so recomputing the ~2*length-byte hex header every scan would
	// be wasted work. New and just-crossed paths are absent from
	// completedFingerprints and get the bridging header.
	if s.cfg.Fingerprint.Growing {
		if _, done := s.completedFingerprints[it.filename]; !done {
			fd.Fingerprint.Raw = hex.EncodeToString(s.readBuffer[:length])
		}
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
