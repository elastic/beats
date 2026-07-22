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
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

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
		log:         logger.Named("scanner"),
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

	st := s.newScanState(opts)

	for _, lit := range s.literals {
		if _, err := os.Lstat(lit); err != nil {
			if isObservationError(err) {
				st.recordUnobservable(lit)
			}
			continue
		}
		st.process(lit, s.pathIndex[lit])
	}

	for _, g := range s.walkGroups {
		s.walk(g, st.process, st.recordUnobservable)
	}

	st.metrics.FilesUnique = int64(len(st.fdByName))

	// prefixes is returned to the watcher, so it is built unconditionally.
	var prefixes []string
	if len(st.unobservable) > 0 {
		prefixes = slices.Sorted(maps.Keys(st.unobservable))
		s.debugLogUnobservable(prefixes)
	}

	s.lastCount = len(st.fdByName)
	return st.fdByName, st.metrics, prefixes
}

// scanState is the mutable state of a single GetFiles scan. process and
// recordUnobservable mutate it as the literal paths and the directory walk yield
// entries.
type scanState struct {
	s    *fileScanner
	opts loginp.FileScanOptions

	// fdByName holds the descriptors GetFiles will return, keyed by filename.
	fdByName map[string]loginp.FileDescriptor
	// uniqueIDs maps a file identity to the path that claimed it, used to detect
	// when a symlink or a second glob resolves to an already-known target.
	uniqueIDs map[string]matchedTarget
	// uniqueFiles holds filenames already processed, used to filter duplicate
	// matches when multiple globs match the same file.
	uniqueFiles map[string]struct{}
	// unobservable collects path prefixes the scan could not read/stat/open due
	// to a resource or permission error (e.g. file-descriptor exhaustion) rather
	// than the path being gone. The watcher uses them to postpone delete detection
	// so a transient failure does not wipe registry state and re-ingest files.
	unobservable map[string]struct{}

	metrics loginp.FileScanMetrics
}

// newScanState allocates the per-scan maps, pre-sized from the previous scan's
// returned count.
func (s *fileScanner) newScanState(opts loginp.FileScanOptions) *scanState {
	return &scanState{
		s:            s,
		opts:         opts,
		fdByName:     make(map[string]loginp.FileDescriptor, s.lastCount),
		uniqueIDs:    make(map[string]matchedTarget, s.lastCount),
		uniqueFiles:  make(map[string]struct{}, s.lastCount),
		unobservable: map[string]struct{}{},
	}
}

// recordUnobservable marks path as a prefix the scan could not observe, counting
// it once. Passed to walk as the recordUnobservable callback.
func (st *scanState) recordUnobservable(path string) {
	if _, ok := st.unobservable[path]; ok {
		return
	}
	st.unobservable[path] = struct{}{}
	st.metrics.ScanErrors++
}

// process evaluates one matched filename: it filters duplicates, builds an ingest
// target and file descriptor, resolves file-identity collisions against paths
// already matched this scan, and records the descriptor (or the relevant metric)
// in the scan state. orderIndex is the position in s.paths of the pattern that
// matched filename, used to resolve identity collisions deterministically.
// Passed to walk as the process callback.
func (st *scanState) process(filename string, orderIndex int) {
	s, opts := st.s, st.opts
	st.metrics.FilesMatched++

	// in case multiple globs match on the same file we filter out duplicates
	if _, knownFile := st.uniqueFiles[filename]; knownFile {
		st.metrics.FilesNoIngestTarget++
		return
	}
	st.uniqueFiles[filename] = struct{}{}

	it, err := s.getIngestTarget(filename)
	if err != nil {
		if errors.Is(err, errFileEmpty) {
			st.metrics.FilesEmpty++
			return
		}

		s.log.Debugf("cannot create an ingest target for file %q: %s", filename, err)
		if errors.Is(err, errFileIgnored) {
			st.metrics.FilesIgnored++
			return
		}

		// A stat/lstat that failed for a reason other than the file being
		// gone (e.g. EMFILE) means we could not observe this path this scan.
		if isObservationError(err) {
			st.recordUnobservable(filename)
		}
		st.metrics.FilesNoIngestTarget++
		return
	}

	fd, err := s.toFileDescriptor(&it)
	if errors.Is(err, errFileTooSmall) {
		st.metrics.FilesNoIngestTarget++
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
		st.metrics.FilesNoIngestTarget++
		// Fingerprinting opens the file; under fd exhaustion the open fails
		// with EMFILE, which is an observation failure, not a missing file.
		if isObservationError(err) {
			st.recordUnobservable(filename)
		}
		s.log.Warnf("cannot create a file descriptor for an ingest target %q: %s", filename, err)
		return
	}

	fileID := fd.FileID()
	if known, exists := st.uniqueIDs[fileID]; exists {
		st.metrics.FilesNoIngestTarget++

		// The same file is reachable via more than one path. Keep the path
		// the previous implementation would have kept, so the returned
		// filename is stable across scans and releases; otherwise,
		//  - the "path" file identity would change and the file could be re-ingested,
		//  - the fingerprint file identity could choose another file to open.
		if !s.matchedEarlier(filename, orderIndex, known.name, known.order) {
			s.log.Warnf("%q points to an already known ingest target %q. Skipping", fd.Filename, known.name)
			return
		}
		s.log.Debugf("%q supersedes already matched ingest target %q for the same file", filename, known.name)
		// the superseded descriptor was already counted as ignored if it
		// matched the ignore options; take that back so FilesIgnored counts
		// only the descriptors actually returned
		if oldFd, ok := st.fdByName[known.name]; ok && isFileIgnored(oldFd, opts) {
			st.metrics.FilesIgnored--
		}
		delete(st.fdByName, known.name)
	}
	st.uniqueIDs[fileID] = matchedTarget{name: filename, order: orderIndex}
	s.attachBridgingRaw(&fd)
	st.fdByName[filename] = fd
	if isFileIgnored(fd, opts) {
		st.metrics.FilesIgnored++
	}
}

// debugLogUnobservable logs a sample of the path prefixes a scan could not
// observe (permissions or file-descriptor exhaustion). prefixes must be sorted.
func (s *fileScanner) debugLogUnobservable(prefixes []string) {
	if !s.log.IsDebug() {
		return
	}
	const maxSamples = 5
	sample := prefixes[:min(len(prefixes), maxSamples)]
	s.log.Debugf("scan could not observe %d path(s) (permissions or file-descriptor exhaustion); first %d: %v",
		len(prefixes), len(sample), sample)
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
	for i := range len(comps) {
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
