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
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
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

	// postponedWarnInterval throttles the "postponing delete detection" warning
	postponedWarnInterval = 5 * time.Minute
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

// isObservationError reports whether err means the scanner could not observe a
// path this scan, as opposed to the path being genuinely gone. It is true for
// filesystem syscall failures such as EMFILE/ENFILE (file-descriptor
// exhaustion), EACCES or EIO, and false for a missing file/dir or a path
// component that is no longer a directory (real deletion signals), and for
// logical rejections that carry no syscall error (e.g. "file is a directory",
// "symlink and they're disabled") — those wrap a plain error, not an
// *os.PathError.
func isObservationError(err error) bool {
	if err == nil || errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
		return false
	}
	var pathErr *os.PathError
	return errors.As(err, &pathErr)
}

// underAnyUnobservable reports whether path is equal to, or nested under, any of
// the given unobservable path prefixes. The comparison is separator-aware so
// "/a/b" is not considered a prefix of "/a/bc".
func underAnyUnobservable(path string, prefixes map[string]struct{}) bool {
	if _, ok := prefixes[path]; ok {
		return true
	}
	for i := len(path) - 1; i > 0; i-- {
		if path[i] == filepath.Separator {
			if _, ok := prefixes[path[:i]]; ok {
				return true
			}
		}
	}
	return false
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

	// lastPostponedWarn is when the "postponing delete detection" warning was last
	// emitted.
	lastPostponedWarn time.Time
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
	paths, scanMetrics, unobservable := w.scanner.GetFiles(scanOpts)
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
	// deleted, renamed, or under a path this scan could not observe. The passes
	// below run in a deliberate priority order, because a positive rename match
	// (the file's content was seen at a new path this scan) is stronger evidence
	// than "the old location was unobservable", which is in turn stronger than
	// "the file is gone":
	//
	//   1. Exact-FileID rename match — works for every identity including
	//      static fingerprint. Catches a plain rename where the file's
	//      content (and so its fingerprint) is unchanged.
	//   2. Prefix-match rename detection (Enhanced Fingerprint / growing
	//      mode only) — catches rename + content growth in the same scan.
	//   3. Postpone deletes for entries under an unobservable prefix. Runs
	//      AFTER both rename passes so a file renamed out of a directory that
	//      became unobservable this scan is still detected as a rename, instead
	//      of being carried forward here AND re-created from offset 0.
	//   4. Unmatched-leftover emission — anything still in w.prev becomes
	//      OpDelete, anything still in newFilesByName becomes OpCreate.

	// Exact-FileID rename match.

	// remaining files in the prev map are the ones that are missing
	// either because they have been deleted or renamed
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

	// Postpone deletes for entries still unmatched after both rename passes
	// whose path is under a prefix this scan could not observe (e.g. a directory
	// that hit EMFILE). We cannot tell whether these files are really gone;
	// treating them as deleted would wipe their registry state and re-ingest from
	// offset 0 once the resource frees up. Running this after the rename
	// passes is what prevents double-ingestion of a file renamed out of a now
	// unobservable directory: it was already emitted as a rename above, so only
	// entries that matched no rename reach this point.
	postponed := 0
	if len(unobservable) > 0 {
		unobservableSet := make(map[string]struct{}, len(unobservable))
		for _, p := range unobservable {
			unobservableSet[p] = struct{}{}
		}
		for remainingPath, remainingDesc := range w.prev {
			if !underAnyUnobservable(remainingPath, unobservableSet) {
				continue
			}
			paths[remainingPath] = remainingDesc
			delete(w.prev, remainingPath)
			postponed++
		}
	}

	// Unmatched-leftover deletes: prev files that weren't matched by either
	// the exact-FileID or the prefix-match rename pass, and are not under an
	// unobservable prefix, are genuinely gone.
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

	if postponed > 0 && now.Sub(w.lastPostponedWarn) >= postponedWarnInterval {
		w.lastPostponedWarn = now
		w.log.Warnf("some previously seen files could not be observed (e.g. file-descriptor exhaustion)"+
			"in the last %s, postponing their delete detection to avoid re-ingestion."+
			"See the filebeat.filestream.scan_errors metric for the current count.",
			postponedWarnInterval)
	}

	w.log.Debugw("File scan complete",
		"total", len(paths),
		"written", writtenCount,
		"truncated", truncatedCount,
		"renamed", renamedCount,
		"removed", removedCount,
		"created", createdCount,
		"postponed", postponed,
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
func (w *fileWatcher) GetFiles(opts loginp.FileScanOptions) (map[string]loginp.FileDescriptor, loginp.FileScanMetrics, []string) {
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

	// walkGroups and literals are derived from paths once (buildWalkGroups) and
	// drive GetFiles: walkGroups are glob patterns grouped by the base directory
	// to walk, literals are paths without any glob metacharacter.
	walkGroups map[string]*walkGroup
	literals   []string

	// pathIndex maps each pattern in paths to its position, and pathsCanOverlap
	// records whether any two patterns can match the same file. Together they let
	// GetFiles resolve a duplicate-identity collision from the scan-order index the
	// walk already knows, instead of rescanning paths on every collision (see
	// matchedEarlier). Both are set once by buildWalkGroups.
	pathIndex       map[string]int
	pathsCanOverlap bool

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

	s.buildWalkGroups()

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

// matchedTarget is a file already accepted for a FileID during a scan: its path
// and the scan-order index (position in s.paths) of the pattern that matched it.
// The index lets matchedEarlier resolve a later collision on the same FileID
// without rescanning s.paths, except when patterns can overlap.
type matchedTarget struct {
	name  string
	order int
}

// GetFiles returns a map of file descriptors by filenames that match the
// configured paths.
// It walks each pattern's base directory a single time and filters
// inline, so files are excluded as they're discovered.
func (s *fileScanner) GetFiles(opts loginp.FileScanOptions) (map[string]loginp.FileDescriptor, loginp.FileScanMetrics, []string) {
	if opts.CurrentTime.IsZero() {
		opts.CurrentTime = time.Now()
	}

	// Pre-size the per-scan maps from the previous scan's count.
	fdByName := make(map[string]loginp.FileDescriptor, s.lastCount)
	// used to determine if a symlink resolves in a already known target
	uniqueIDs := make(map[string]matchedTarget, s.lastCount)
	// used to filter out duplicate matches
	uniqueFiles := make(map[string]struct{}, s.lastCount)
	scanMetrics := loginp.FileScanMetrics{}

	// unobservable collects path prefixes the scan could not read/stat/open due
	// to a resource or permission error (e.g. file-descriptor exhaustion) rather
	// than the path being gone. The watcher uses them to postpone delete detection
	// so a transient failure does not wipe registry state and re-ingest files.
	unobservable := map[string]struct{}{}
	recordUnobservable := func(path string) {
		if _, ok := unobservable[path]; ok {
			return
		}
		unobservable[path] = struct{}{}
		scanMetrics.ScanErrors++
	}

	process := func(filename string, orderIndex int) {
		scanMetrics.FilesMatched++

		// in case multiple globs match on the same file we filter out duplicates
		if _, knownFile := uniqueFiles[filename]; knownFile {
			scanMetrics.FilesNoIngestTarget++
			return
		}
		uniqueFiles[filename] = struct{}{}

		it, err := s.getIngestTarget(filename)
		if err != nil {
			if errors.Is(err, errFileEmpty) {
				scanMetrics.FilesEmpty++
				return
			}

			s.log.Debugf("cannot create an ingest target for file %q: %s", filename, err)
			if errors.Is(err, errFileIgnored) {
				scanMetrics.FilesIgnored++
				return
			}

			// A stat/lstat that failed for a reason other than the file being
			// gone (e.g. EMFILE) means we could not observe this path this scan.
			if isObservationError(err) {
				recordUnobservable(filename)
			}
			scanMetrics.FilesNoIngestTarget++
			return
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
			return
		}
		if err != nil {
			scanMetrics.FilesNoIngestTarget++
			// Fingerprinting opens the file; under fd exhaustion the open fails
			// with EMFILE, which is an observation failure, not a missing file.
			if isObservationError(err) {
				recordUnobservable(filename)
			}
			s.log.Warnf("cannot create a file descriptor for an ingest target %q: %s", filename, err)
			return
		}

		fileID := fd.FileID()
		if known, exists := uniqueIDs[fileID]; exists {
			scanMetrics.FilesNoIngestTarget++

			// The same file is reachable via more than one path. Keep the path
			// the previous implementation would have kept, so the returned
			// filename is stable across scans and releases; otherwise,
			//  - the "path" file identity would change and the file could be re-ingested,
			//  - the fingerprint file identity could choose another file to open.
			if !s.matchedEarlier(filename, orderIndex, known.name, known.order) {
				s.log.Warnf("%q points to an already known ingest target %q [%s==%s]. Skipping", fd.Filename, known.name, fileID, fileID)
				return
			}
			s.log.Debugf("%q supersedes already matched ingest target %q for the same file", filename, known.name)
			// the superseded descriptor was already counted as ignored if it
			// matched the ignore options; take that back so FilesIgnored counts
			// only the descriptors actually returned
			if oldFd, ok := fdByName[known.name]; ok && isFileIgnored(oldFd, opts) {
				scanMetrics.FilesIgnored--
			}
			delete(fdByName, known.name)
		}
		uniqueIDs[fileID] = matchedTarget{name: filename, order: orderIndex}
		s.attachBridgingRaw(&fd)
		fdByName[filename] = fd
		if isFileIgnored(fd, opts) {
			scanMetrics.FilesIgnored++
		}
	}

	for _, lit := range s.literals {
		if _, err := os.Lstat(lit); err != nil {
			if isObservationError(err) {
				recordUnobservable(lit)
			}
			continue
		}
		process(lit, s.pathIndex[lit])
	}

	for _, g := range s.walkGroups {
		s.walk(g, process, recordUnobservable)
	}

	scanMetrics.FilesUnique = int64(len(fdByName))

	var prefixes []string
	if len(unobservable) > 0 {
		prefixes = make([]string, 0, len(unobservable))
		for p := range unobservable {
			prefixes = append(prefixes, p)
		}
		slices.Sort(prefixes)

		sample := prefixes
		maxSamples := 5
		if len(sample) > maxSamples {
			sample = sample[:maxSamples]
		}
		s.log.Debugf("scan could not observe %d path(s) (permissions or file-descriptor exhaustion); first %d: %v",
			len(prefixes), len(sample), sample)
	}

	s.lastCount = len(fdByName)
	return fdByName, scanMetrics, prefixes
}

// walkGroup is a set of (absolute, ** expanded) glob patterns that share the same
// base directory, indexed by their depth below that directory so the walker only
// tests a file against the patterns that can possibly match it.
type walkGroup struct {
	root     string
	maxDepth int
	byDepth  map[int][]string
}

// buildWalkGroups partitions s.paths into literal paths and walk groups keyed by
// their base directory. Patterns that share a base are walked together so the tree
// is read only once. Invalid patterns detectable upfront are dropped and reported
// once here; malformed patterns that escape this check (a bad token behind a
// literal prefix never reaches the parser when matching "") are reported once per
// scan by walk.
func (s *fileScanner) buildWalkGroups() {
	groups := map[string]*walkGroup{}
	var literals []string

	for _, path := range s.paths {
		if !hasGlobMeta(path) {
			literals = append(literals, path)
			continue
		}

		if _, err := filepath.Match(path, ""); err != nil {
			s.log.Errorf("invalid glob pattern %q: %v", path, err)
			continue
		}

		root := globRoot(path)
		g := groups[root]
		if g == nil {
			g = &walkGroup{root: root, byDepth: map[int][]string{}}
			groups[root] = g
		}
		d := depthBelow(root, path)
		g.byDepth[d] = append(g.byDepth[d], path)
		if d > g.maxDepth {
			g.maxDepth = d
		}
	}
	s.walkGroups = groups
	s.literals = literals

	// Index every pattern by its position in paths, and record whether any two
	// patterns can match the same file. When none can, the pattern the walk
	// matched a file against is that file's scan-order position, so a
	// duplicate-identity collision resolves from stored indices (matchedEarlier)
	// instead of rescanning paths.
	s.pathIndex = make(map[string]int, len(s.paths))
	for i, p := range s.paths {
		if _, ok := s.pathIndex[p]; !ok {
			s.pathIndex[p] = i
		}
	}
	s.pathsCanOverlap = pathsCanOverlap(s.paths)
}

// walkPattern is a group pattern together with its path components below the
// group root, used to decide component-wise whether a directory can lead to a
// match.
type walkPattern struct {
	pattern string
	comps   []string
	// orderIndex is the pattern's position in s.paths, carried through to
	// process so a matched file's scan order is known without rescanning s.paths.
	orderIndex int
}

// walk traverses g.root once and invokes process for every entry matching one of
// the group's patterns. A directory is only descended into when its name matches
// the next component of some pattern. Pattern depth bounds the recursion, which
// preserves the RecursiveGlobDepth cap and makes symlink cycles safe.
func (s *fileScanner) walk(g *walkGroup, process func(filename string, orderIndex int), recordUnobservable func(prefix string)) {
	// Flatten the group's patterns in ascending depth order rather than map order,
	// so per-scan malformed-pattern logging and matchLeaf's first-match break are
	// deterministic instead of dependent on Go's map iteration. orderIndex carries
	// each pattern's position in s.paths through to process, so a matched file's
	// scan order is known without rescanning s.paths.
	patterns := make([]walkPattern, 0, len(g.byDepth))
	for d := 0; d <= g.maxDepth; d++ {
		for _, p := range g.byDepth[d] {
			patterns = append(patterns, walkPattern{pattern: p, comps: patternComponents(g.root, p), orderIndex: s.pathIndex[p]})
		}
	}

	// badPatterns dedups ErrBadPattern logs: filepath.Match reports a malformed
	// pattern for every candidate name, but one line per scan is enough.
	badPatterns := map[string]struct{}{}
	logBadPattern := func(pattern string, err error) {
		if _, seen := badPatterns[pattern]; !seen {
			badPatterns[pattern] = struct{}{}
			s.log.Errorf("glob match(%q) failed: %v", pattern, err)
		}
	}

	// rec reads dir, whose entries are at childDepth below the root. alive holds
	// the patterns whose components matched every ancestor directory of dir.
	var rec func(dir string, depth int, alive []walkPattern)
	rec = func(dir string, depth int, alive []walkPattern) {
		childDepth := depth + 1

		// Patterns ending at this level match entries; deeper ones may match
		// below it.
		var exact, deeper []walkPattern
		for _, p := range alive {
			switch {
			case len(p.comps) == childDepth:
				exact = append(exact, p)
			case len(p.comps) > childDepth:
				deeper = append(deeper, p)
			}
		}

		onReadError := func(err error) {
			// Skip unreadable directories instead of aborting the whole scan, as
			// filepath.Glob did. But if the reason is an observation failure
			// (e.g. EMFILE) rather than the directory being gone, record the
			// subtree as unobservable so the watcher postpones deleting the files
			// under it — otherwise fd exhaustion would wipe their state and
			// re-ingest them once fds free up.
			if isObservationError(err) {
				recordUnobservable(dir)
			}
			s.log.Debugf("cannot read directory %q: %s", dir, err)
		}

		// matchLeaf matches one entry name against the exact patterns. Every
		// ancestor component was already matched on the way down, so only the last
		// component is checked here, and the full path is built (once) only when a
		// match is emitted.
		matchLeaf := func(name string) {
			for _, p := range exact {
				matched, matchErr := filepath.Match(p.comps[childDepth-1], name)
				if matchErr != nil {
					logBadPattern(p.pattern, matchErr)
					continue
				}
				if matched {
					process(filepath.Join(dir, name), p.orderIndex)
					break
				}
			}
		}

		// With nothing deeper to descend into, entry types are irrelevant, so read
		// only the names rather than os.ReadDir, which would allocate an
		// os.DirEntry per entry.
		if len(deeper) == 0 {
			names, err := readDirNames(dir)
			if err != nil {
				onReadError(err)
				return
			}
			for _, name := range names {
				matchLeaf(name)
			}
			return
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			onReadError(err)
			return
		}

		for _, e := range entries {
			matchLeaf(e.Name())

			isDir := e.IsDir()
			isSymlink := e.Type()&os.ModeSymlink != 0
			if !isDir && !isSymlink {
				continue
			}
			// Keep the patterns whose next component matches this directory name;
			// none matching means nothing below this directory can ever match.
			var childAlive []walkPattern
			for _, p := range deeper {
				ok, matchErr := filepath.Match(p.comps[childDepth-1], e.Name())
				if matchErr != nil {
					logBadPattern(p.pattern, matchErr)
					continue
				}
				if ok {
					childAlive = append(childAlive, p)
				}
			}
			if len(childAlive) == 0 {
				continue
			}
			full := filepath.Join(dir, e.Name())
			if !isDir {
				// Resolve the symlink to decide whether to descend. A broken
				// symlink cannot be descended into; if it  matched a pattern it
				// was already yielded above.
				info, statErr := os.Stat(full)
				if statErr != nil {
					// If we could not stat the target because of an observation
					// error (EACCES/EIO on the symlink target) we don't know
					// whether to descend; record the subtree so the watcher
					// postpones deleting files under it, as with a read error. A
					// broken symlink (ErrNotExist) is not observable-related and is
					// skipped.
					if isObservationError(statErr) {
						recordUnobservable(full)
					}
					continue
				}
				isDir = info.IsDir()
			}
			if isDir {
				rec(full, childDepth, childAlive)
			}
		}
	}
	rec(g.root, 0, patterns)
}

// readDirNames returns the sorted entry names of dir. It reads names only and
// so avoids the per-entry os.DirEntry allocation of os.ReadDir; the walker uses
// it for leaf directories, where entry types are not needed. Names are sorted
// to keep traversal order stable.
func readDirNames(dir string) ([]string, error) {
	f, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	names, err := f.Readdirnames(-1)
	_ = f.Close()
	if err != nil {
		return nil, err
	}
	slices.Sort(names)
	return names, nil
}

// hasGlobMeta reports whether path contains any glob metacharacter, mirroring the
// unexported path/filepath.hasMeta.
func hasGlobMeta(path string) bool {
	magic := `*?[`
	if filepath.Separator != '\\' {
		magic = `*?[\`
	}
	return strings.ContainsAny(path, magic)
}

// globRoot returns the longest leading directory of pattern that has no glob
// metacharacter — the directory from which the tree is walked.
func globRoot(pattern string) string {
	dir := pattern
	for hasGlobMeta(dir) {
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return dir
}

// depthBelow returns the number of path segments of pattern below root, including
// the trailing filename segment. root must be an ancestor of pattern.
func depthBelow(root, pattern string) int {
	return len(patternComponents(root, pattern))
}

// patternComponents returns pattern's path segments below root. root must be an
// ancestor of pattern.
func patternComponents(root, pattern string) []string {
	rel := strings.TrimPrefix(pattern, root)
	rel = strings.TrimPrefix(rel, string(filepath.Separator))
	if rel == "" {
		return nil
	}
	return strings.Split(rel, string(filepath.Separator))
}

// scanOrderIndex returns the index of the first configured (** expanded) pattern
// in s.paths that matches filename. This reproduces the order in which the
// previous filepath.Glob implementation processed matches: it globbed s.paths in
// order, and s.paths is ordered by configured path, then by ascending recursive
// depth. Scanning the whole of s.paths (rather than a single group) keeps the
// result correct even when configured paths overlap.
func (s *fileScanner) scanOrderIndex(filename string) int {
	for i, p := range s.paths {
		if ok, _ := filepath.Match(p, filename); ok {
			return i
		}
	}
	// Sentinel: rank a non-matching path after every real match (valid indices
	// are 0..len(s.paths)-1, so len(s.paths) sorts strictly last). Unreachable in
	// practice — callers only pass paths the walker already matched against a
	// pattern in s.paths.
	return len(s.paths)
}

// matchedEarlier reports whether path a would have been processed before path b by
// the previous implementation using filepath.Glob. The path matched by the earlier
// pattern wins; ties are broken comparing path components, mirroring Glob's
// per-directory sort. Used only to resolve the rare case where two paths resolve
// to the same file, so the current implementation does not affect which paths
// are kept, preserving the behavior.
//
// aIndex and bIndex are the scan-order indices the walk already computed for each
// path (the position in s.paths of the pattern it matched). They are authoritative
// only when patterns cannot overlap: then each file matches exactly one pattern,
// so the walk's index is the file's scan order. When patterns can overlap a file
// may match an earlier pattern than the one the walk used, so the indices are
// recomputed with scanOrderIndex.
func (s *fileScanner) matchedEarlier(a string, aIndex int, b string, bIndex int) bool {
	if s.pathsCanOverlap {
		aIndex, bIndex = s.scanOrderIndex(a), s.scanOrderIndex(b)
	}
	if aIndex != bIndex {
		return aIndex < bIndex
	}
	// filepath.Glob sorts names within each directory and concatenates, so its
	// order is lexicographic on path components, not on full-path bytes: the two
	// diverge when a sibling name is a byte-prefix of another and the next byte
	// sorts before '/' (e.g. Glob visits "d" before "d-x", yet "d-x/a" < "d/z").
	as := strings.Split(a, string(filepath.Separator))
	bs := strings.Split(b, string(filepath.Separator))
	for i := 0; i < len(as) && i < len(bs); i++ {
		if as[i] != bs[i] {
			return as[i] < bs[i]
		}
	}
	return len(as) < len(bs)
}

// pathsCanOverlap reports whether any two of the given (** expanded) patterns can
// match the same file. It is conservative: it only rules a pair out when an
// aligned path component is a differing literal in both patterns (which soundly
// proves no path matches both), so it never returns false when an overlap is
// possible. When it returns false, the pattern the walk matched a file against is
// that file's scan-order position and matchedEarlier can skip scanOrderIndex.
func pathsCanOverlap(paths []string) bool {
	sep := string(filepath.Separator)
	comps := make([][]string, len(paths))
	for i, p := range paths {
		comps[i] = strings.Split(p, sep)
	}
	for i := 0; i < len(comps); i++ {
		for j := i + 1; j < len(comps); j++ {
			// Different segment counts can never match the same path: a wildcard
			// does not cross the separator and there is no "**" left after
			// expansion, so filepath.Match requires equal segment counts.
			if len(comps[i]) != len(comps[j]) {
				continue
			}
			if patternsCanCoMatch(comps[i], comps[j]) {
				return true
			}
		}
	}
	return false
}

// patternsCanCoMatch reports whether two equal-length component lists could match
// a common path. It returns false only on a provable disjointness: an aligned
// component that is a literal (no glob metacharacter) in both and differs. Any
// other pair is treated as possibly overlapping.
func patternsCanCoMatch(a, b []string) bool {
	for k := range a {
		if !hasGlobMeta(a[k]) && !hasGlobMeta(b[k]) && a[k] != b[k] {
			return false
		}
	}
	return true
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
