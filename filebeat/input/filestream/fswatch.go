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
	"path/filepath"
	"sync"
	"time"

	"github.com/elastic/go-concert/unison"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	watcherDebugKey = "file_watcher"

	// postponedWarnInterval throttles the "postponing delete detection" warning
	postponedWarnInterval = 5 * time.Minute
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
		w.log.Warnf("some previously seen files could not be observed (e.g. file-descriptor exhaustion) in the last %s, postponing their delete detection to avoid re-ingestion. See the filebeat.filestream.scan_errors metric for the current count.",
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
