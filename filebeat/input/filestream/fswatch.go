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
	// MinFingerprintSize is the smallest allowed fingerprint length (one SHA-256 block).
	MinFingerprintSize int64 = sha256.BlockSize
	// MaxFingerprintSize caps fingerprint length; larger values risk exhausting scanner memory.
	MaxFingerprintSize int64 = 10 * 1024 * 1024 // 10MB
	scannerDebugKey          = "scanner"
	watcherDebugKey          = "file_watcher"
)

var (
	errFileTooSmall = errors.New("file size is too small for ingestion")
	errFileEmpty    = errors.New("file is empty")
	errFileIgnored  = errors.New("ignored by scanner configuration")
)

type ignoredFileError string

func (e ignoredFileError) Error() string {
	return string(e)
}

func (e ignoredFileError) Unwrap() error {
	return errFileIgnored
}

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

func (w *fileWatcher) Run(
	ctx unison.Canceler,
	metrics *loginp.Metrics,
	ignoreOlder time.Duration,
	ignoreInactiveSince time.Time,
) {
	defer close(w.events)
	defer metrics.Cleanup()

	// run initial scan before starting regular
	w.watch(ctx, metrics, ignoreOlder, ignoreInactiveSince)

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
			w.watch(ctx, metrics, ignoreOlder, ignoreInactiveSince)
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

func (w *fileWatcher) watch(
	ctx unison.Canceler,
	metrics *loginp.Metrics,
	ignoreOlder time.Duration,
	ignoreInactiveSince time.Time,
) {
	w.log.Debug("Start next scan")

	// file identity is updated in GetFiles
	now := time.Now()
	scanOpts := loginp.FileScanOptions{
		CurrentTime:         now,
		IgnoreOlder:         ignoreOlder,
		IgnoreInactiveSince: ignoreInactiveSince,
	}
	paths, scanMetrics := w.scanner.GetFiles(scanOpts)
	metrics.UpdateFileScanMetrics(scanMetrics)

	// for debugging purposes
	writtenCount := 0
	truncatedCount := 0
	renamedCount := 0
	removedCount := 0
	createdCount := 0

	newFilesByName := make(map[string]*loginp.FileDescriptor)
	newFilesByID := make(map[string]*loginp.FileDescriptor)
	harvesterFiles := make([]loginp.HarvesterFile, 0, len(paths))

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

		// srcID is the file identity (harvester ID/registry key), resolved lazily via ensureSrcID:
		// an unchanged, untracked file (gzip, empty, ignore_older) never needs one, saving allocs.
		var srcID string
		ensureSrcID := func() string {
			if srcID == "" { // getFileIdentity never returns ""
				srcID = w.getFileIdentity(fd)
			}
			return srcID
		}

		// closedHarvesters is empty in the steady state; this reconciliation is usually skipped.
		if w.hasClosedHarvesters() {
			w.reconcileClosedHarvester(&prevDesc, ensureSrcID())
		}

		var e loginp.FSEvent
		switch {
		// the new size is smaller, the file was truncated
		case prevDesc.Info.Size() > fd.Info.Size():
			e = truncateEvent(path, fd, ensureSrcID())
			truncatedCount++

		// the size is the same, timestamps are different, the file was touched
		case prevDesc.Info.Size() == fd.Info.Size() && prevDesc.Info.ModTime() != fd.Info.ModTime():
			if w.cfg.ResendOnModTime {
				e = truncateEvent(path, fd, ensureSrcID())
				truncatedCount++
			}

		// the new size is larger, something was written.
		// If a harvester for this file was closed recently,
		// we use its state instead of the one we have cached.
		case prevDesc.SizeOrBytesIngested() < fd.Info.Size():
			e = writeEvent(path, fd, ensureSrcID())
			writtenCount++

		default:
			// For the delete feature we need to run the harvester for
			// files that have not changed until they're deleted.
			if w.cfg.SendNotChanged {
				e = notChangedEvent(path, fd, ensureSrcID())
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

		// Record progress metrics for trackable, non-truncated files (tracksHarvesterProgress).
		if e.Op != loginp.OpTruncate && tracksHarvesterProgress(&fd, scanOpts) {
			harvesterFiles = append(harvesterFiles, loginp.HarvesterFile{ID: ensureSrcID(), Size: fd.Info.Size()})
		}

		// delete from previous state to mark that we've seen the existing file again
		delete(w.prev, path)
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

	// Exact-FileID rename match.
	for remainingPath, remainingDesc := range w.prev {
		newDesc, renamed := newFilesByID[remainingDesc.FileID()]
		if !renamed {
			continue
		}

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
	}

	// Prefix-match candidates are the still-growing prev entries left after
	// the exact-match pass (GrowingRaw is empty for completed entries, which
	// match by their SHA-256 identity instead). The index is only built when
	// this scan has a new file that could justify a match — a delete-only
	// scan pays no hashing.
	var shortFingerprints *shortFingerprintSet
	if w.growingFingerprint {
		for _, newDesc := range newFilesByName {
			if newDesc.Fingerprint.Complete() {
				shortFingerprints = newShortFingerprintSet()
				break
			}
		}
	}
	if shortFingerprints != nil {
		for remainingPath, remainingDesc := range w.prev {
			if raw := remainingDesc.Fingerprint.GrowingRaw(); raw != "" {
				shortFingerprints.AddRaw(remainingPath, raw, remainingPath)
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
		srcID := w.getFileIdentity(*fd)

		select {
		case <-ctx.Done():
			return
		case w.events <- createEvent(path, *fd, srcID):
			createdCount++
		}

		// New files skip the main loop via early continue, so collect their metrics here.
		if tracksHarvesterProgress(fd, scanOpts) {
			harvesterFiles = append(harvesterFiles, loginp.HarvesterFile{ID: srcID, Size: fd.Info.Size()})
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

	metrics.UpdateHarvesterBuckets(harvesterFiles)

	w.prev = paths
}

// hasClosedHarvesters reports whether any harvester-close notification awaits reconciliation.
func (w *fileWatcher) hasClosedHarvesters() bool {
	w.closedHarvestersMutex.Lock()
	defer w.closedHarvestersMutex.Unlock()
	return len(w.closedHarvesters) > 0
}

// reconcileClosedHarvester folds a recently-closed harvester's ingested offset (from
// closedHarvesters) into prevDesc so a restarted harvester resumes from the right position.
// It guards a close-during-backoff race that would otherwise withhold writes and lose lines.
func (w *fileWatcher) reconcileClosedHarvester(prevDesc *loginp.FileDescriptor, id string) {
	w.closedHarvestersMutex.Lock()
	defer w.closedHarvestersMutex.Unlock()
	size, ok := w.closedHarvesters[id]
	if !ok {
		return
	}
	w.log.Debugf("Updating previous state because harvester was closed. '%s': %d", id, size)
	prevDesc.SetBytesIngested(size)
	delete(w.closedHarvesters, id)
}

// tracksHarvesterProgress reports whether a file contributes to the harvester progress metrics.
func tracksHarvesterProgress(fd *loginp.FileDescriptor, opts loginp.FileScanOptions) bool {
	return !fd.GZIP && fd.Info.Size() > 0 && !isFileIgnored(*fd, opts)
}

// isFileIgnored returns true when a file is ignored, no matter the reason.
func isFileIgnored(
	fd loginp.FileDescriptor,
	opts loginp.FileScanOptions,
) bool {
	modTime := fd.Info.ModTime()

	if opts.IgnoreOlder > 0 && opts.CurrentTime.Sub(modTime) > opts.IgnoreOlder {
		return true
	}

	if !opts.IgnoreInactiveSince.IsZero() && modTime.Sub(opts.IgnoreInactiveSince) <= 0 {
		return true
	}

	return false
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
// TakeOver phases. Unlike the watch loop it does not advance the scanner's
// completedFingerprints set, so these pre-watch scans cannot suppress the
// bridging raw header a still-growing entry needs to migrate its registry key
// after a restart.
func (w *fileWatcher) GetFiles(opts loginp.FileScanOptions) (map[string]loginp.FileDescriptor, loginp.FileScanMetrics) {
	return w.scanner.GetFiles(opts)
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
	// completedFingerprints holds paths already complete on the previous watch scan
	// (growing mode), so attachBridgingRaw can skip re-encoding their bridging header.
	// Only fileWatcher.watch advances it, so prospector enumeration can't wrongly suppress it.
	completedFingerprints map[string]struct{}

	// lastCount is the number of unique files the previous scan produced.
	lastCount int
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
		if s.cfg.Fingerprint.Length < MinFingerprintSize {
			err := fmt.Errorf("fingerprint size %d bytes cannot be smaller than %d bytes", config.Fingerprint.Length, MinFingerprintSize)
			return nil, fmt.Errorf("error while reading configuration of fingerprint: %w", err)
		}
		if s.cfg.Fingerprint.Length > MaxFingerprintSize {
			s.log.Warnf("fingerprint length %d bytes exceeds the maximum of %d bytes, capping to the maximum",
				s.cfg.Fingerprint.Length, MaxFingerprintSize)
			s.cfg.Fingerprint.Length = MaxFingerprintSize
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
func (s *fileScanner) GetFiles(opts loginp.FileScanOptions) (map[string]loginp.FileDescriptor, loginp.FileScanMetrics) {
	if opts.CurrentTime.IsZero() {
		opts.CurrentTime = time.Now()
	}

	// Pre-size the per-scan maps from the previous scan's count.
	fdByName := make(map[string]loginp.FileDescriptor, s.lastCount)
	// used to determine if a symlink resolves in a already known target
	uniqueIDs := make(map[string]string, s.lastCount)
	// used to filter out duplicate matches
	uniqueFiles := make(map[string]struct{}, s.lastCount)
	scanMetrics := loginp.FileScanMetrics{}

	for _, path := range s.paths {
		matches, err := filepath.Glob(path)
		if err != nil {
			s.log.Errorf("glob(%s) failed: %v", path, err)
			continue
		}
		scanMetrics.FilesMatched += int64(len(matches))

		for _, filename := range matches {
			// in case multiple globs match on the same file we filter out duplicates
			if _, knownFile := uniqueFiles[filename]; knownFile {
				scanMetrics.FilesNoIngestTarget++
				continue
			}
			uniqueFiles[filename] = struct{}{}

			it, err := s.getIngestTarget(filename)
			if err != nil {
				if errors.Is(err, errFileEmpty) {
					scanMetrics.FilesEmpty++
					continue
				}

				s.log.Debugf("cannot create an ingest target for file %q: %s", filename, err)
				if errors.Is(err, errFileIgnored) {
					scanMetrics.FilesIgnored++
					continue
				}

				scanMetrics.FilesNoIngestTarget++
				continue
			}

			fd, err := s.toFileDescriptor(&it)
			if errors.Is(err, errFileTooSmall) {
				scanMetrics.FilesNoIngestTarget++
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
				scanMetrics.FilesNoIngestTarget++
				s.log.Warnf("cannot create a file descriptor for an ingest target %q: %s", filename, err)
				continue
			}

			fileID := fd.FileID()
			if knownFilename, exists := uniqueIDs[fileID]; exists {
				scanMetrics.FilesNoIngestTarget++
				s.log.Warnf("%q points to an already known ingest target %q. Skipping", fd.Filename, knownFilename)
				continue
			}
			uniqueIDs[fileID] = fd.Filename
			s.attachBridgingRaw(&fd)
			fdByName[filename] = fd
			if isFileIgnored(fd, opts) {
				scanMetrics.FilesIgnored++
			}
		}
	}

	scanMetrics.FilesUnique = int64(len(fdByName))
	s.lastCount = len(fdByName)
	return fdByName, scanMetrics
}

type ingestTarget struct {
	filename         string
	originalFilename string
	symlink          bool
	info             commonfile.ExtendedFileInfo
}

func (s *fileScanner) getIngestTarget(filename string) (it ingestTarget, err error) {
	if s.isFileExcluded(filename) {
		return it, ignoredFileError(fmt.Sprintf("file %q is excluded from ingestion", filename))
	}

	if !s.isFileIncluded(filename) {
		return it, ignoredFileError(fmt.Sprintf("file %q is not included in ingestion", filename))
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
			return it, ignoredFileError(fmt.Sprintf("file %q->%q is excluded from ingestion", it.filename, it.originalFilename))
		}

		if !s.isFileIncluded(it.originalFilename) {
			return it, ignoredFileError(fmt.Sprintf("file %q->%q is not included in ingestion", it.filename, it.originalFilename))
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
//     with SHA-256 (FingerprintID.Sum, so Complete() is true). The growing-mode
//     bridging raw header is added later by attachBridgingRaw, not here.
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
		// size <= offset we cannot read anything from the offset, regardless of mode.
		if it.info.Size() <= offset {
			return fd, fmt.Errorf(
				"filesize of %q is %d bytes, less than fingerprint offset %d: %w",
				fd.Filename, it.info.Size(), offset, errFileTooSmall)
		}
		if !s.cfg.Fingerprint.Growing && it.info.Size() < threshold {
			return fd, fmt.Errorf(
				"filesize of %q is %d bytes, expected at least %d bytes for fingerprinting: %w",
				fd.Filename, it.info.Size(), threshold, errFileTooSmall)
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
				"only %d bytes readable from offset %d in %q, expected at least %d bytes for fingerprinting: %w",
				n, offset, fd.Filename, length, errFileTooSmall)
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

	return fd, nil
}

// attachBridgingRaw sets a complete descriptor's raw header.
func (s *fileScanner) attachBridgingRaw(fd *loginp.FileDescriptor) {
	if !s.cfg.Fingerprint.Growing || !fd.Fingerprint.Complete() {
		return
	}
	if _, done := s.completedFingerprints[fd.Filename]; done {
		return
	}
	fd.Fingerprint.Raw = hex.EncodeToString(s.readBuffer[:s.cfg.Fingerprint.Length])
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
