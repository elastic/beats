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
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"time"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/common/file"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/unison"
)

const (
	copyTruncateProspectorDebugKey = "copy_truncate_file_prospector"
	copiedFileIdx                  = 0
)

var numericSuffixRegexp = regexp.MustCompile(`\d*$`)
var numbersFromSuffixRegexp = regexp.MustCompile(`\d+`)

// sorter is required for ordering rotated log files
// The slice is ordered so the newest rotated file comes first.
type sorter interface {
	sort([]rotatedFileInfo)
}

// rotatedFileInfo stores the file information of a rotated file.
type rotatedFileInfo struct {
	path string
	src  loginp.Source

	ts  time.Time
	idx int
}

func (f rotatedFileInfo) String() string {
	return f.path
}

// rotatedFilestream includes the information of the original file
// and its identifier, and the rotated file.
type rotatedFilestream struct {
	originalSrc loginp.Source
	rotated     []rotatedFileInfo
}

func newRotatedFilestreams(cfg *copyTruncateConfig) (*rotatedFilestreams, error) {
	var s sorter
	var err error

	if cfg.DateFormat != "" {
		s, err = newDateSorter(
			cfg.DateFormat,
			cfg.DateRegex,
			cfg.DateRegex)
		if err != nil {
			return nil, fmt.Errorf("failed to create date sorter: %w", err)
		}
	} else {
		s, err = newNumericSorter(cfg.SuffixRegex)
		if err != nil {
			return nil, fmt.Errorf("failed to create numeric sorter: %w", err)
		}
	}

	return &rotatedFilestreams{
		table:  make(map[string]*rotatedFilestream, 0),
		sorter: s,
	}, nil
}

// numericSorter sorts rotated log files that have a numeric suffix.
// Example: apache.log.1, apache.log.2
type numericSorter struct {
	suffix *regexp.Regexp
}

// TODO(AndersonQ): update user-facing docs
func newNumericSorter(suffixRegex string) (*numericSorter, error) {
	suffix := numericSuffixRegexp

	if suffixRegex != "" {
		var err error
		suffix, err = regexp.Compile(suffixRegex)
		if err != nil {
			return nil,
				fmt.Errorf("numeric sorter: failed to compile numeric suffix regex: %w",
					err)
		}
	}

	return &numericSorter{
		suffix: suffix,
	}, nil
}

func (s *numericSorter) sort(files []rotatedFileInfo) {
	sort.Slice(
		files,
		func(i, j int) bool {
			return s.GetIdx(&files[i]) < s.GetIdx(&files[j])
		},
	)
}

func (s *numericSorter) GetIdx(fi *rotatedFileInfo) int {
	if fi.idx > 0 {
		return fi.idx
	}

	suffix := s.suffix.FindString(fi.path)
	if suffix == "" {
		return -1
	}

	idxStr := numbersFromSuffixRegexp.FindString(suffix)
	if idxStr == "" {
		return -1
	}

	idx, err := strconv.Atoi(idxStr)
	if err != nil {
		return -1
	}
	fi.idx = idx

	return idx
}

// dateSorter sorts rotated log files that have a date suffix
// based on the configured format.
// Example: apache.log-21210526, apache.log-20210527
type dateSorter struct {
	format    string
	dateRegex *regexp.Regexp
	suffix    *regexp.Regexp
}

// TODO(AndersonQ): update user-facing docs
//   - it matches the 1st then tries to match the 2nd.
func newDateSorter(
	format,
	dateRegex,
	suffixRegex string) (*dateSorter, error) {
	if suffixRegex == "" {
		return nil, errors.New("date sorter: no suffix regex specified")
	}
	dateReg, err := regexp.Compile(dateRegex)
	if err != nil {
		return nil,
			fmt.Errorf(
				"date sorter: failed to compile 'date_regex': %w",
				err)
	}

	suffix, err := regexp.Compile(suffixRegex)
	if err != nil {
		return nil,
			fmt.Errorf("date sorter: failed to compile numeric suffix regex: %w",
				err)
	}

	return &dateSorter{
		format:    format,
		dateRegex: dateReg,
		suffix:    suffix,
	}, nil
}

func (s *dateSorter) sort(files []rotatedFileInfo) {
	sort.Slice(
		files,
		func(i, j int) bool {
			return s.GetTs(&files[j]).Before(s.GetTs(&files[i]))
		},
	)
}

func (s *dateSorter) GetTs(fi *rotatedFileInfo) time.Time {
	if !fi.ts.IsZero() {
		return fi.ts
	}

	suffix := s.suffix.FindString(fi.path)
	if suffix == "" {
		return time.Time{}
	}

	matches := s.dateRegex.FindStringSubmatch(suffix)
	var dateStr string
	switch {
	case len(matches) == 0:
		return time.Time{}
	case len(matches) == 1:
		dateStr = matches[0]
	case len(matches) >= 2:
		dateStr = matches[1]
	}

	ts, err := time.Parse(s.format, dateStr)
	if err != nil {
		return time.Time{}
	}
	fi.ts = ts
	return ts
}

// rotatedFilestreams is a map of original files and their rotated instances.
type rotatedFilestreams struct {
	table  map[string]*rotatedFilestream
	sorter sorter
}

// addOriginalFile adds a new original file and its identifying information
// to the bookkeeper.
func (r rotatedFilestreams) addOriginalFile(path string, src loginp.Source) {
	if _, ok := r.table[path]; ok {
		return
	}
	r.table[path] = &rotatedFilestream{originalSrc: src, rotated: make([]rotatedFileInfo, 0)}
}

// isOriginalAdded checks if an original file has been found.
func (r rotatedFilestreams) isOriginalAdded(path string) bool {
	_, ok := r.table[path]
	return ok
}

// addRotatedFile adds a new rotated file to the list and returns its index.
// if a file is already added, the source is updated and the index is returned.
func (r rotatedFilestreams) addRotatedFile(original, rotated string, src loginp.Source) int {
	for idx, fi := range r.table[original].rotated {
		if fi.path == rotated {
			r.table[original].rotated[idx].src = src
			return idx
		}
	}

	r.table[original].rotated = append(
		r.table[original].rotated,
		rotatedFileInfo{path: rotated, src: src, ts: time.Time{}, idx: 0})
	r.sorter.sort(r.table[original].rotated)

	for idx, fi := range r.table[original].rotated {
		if fi.path == rotated {
			return idx
		}
	}

	return -1
}

type copyTruncateFileProspector struct {
	fileProspector
	rotatedSuffix *regexp.Regexp
	rotatedFiles  *rotatedFilestreams
}

// Run starts the fileProspector which accepts FS events from a file watcher.
func (p *copyTruncateFileProspector) Run(ctx input.Context, s loginp.StateMetadataUpdater, hg loginp.HarvesterGroup) {
	log := ctx.Logger.With("prospector", copyTruncateProspectorDebugKey)
	log.Debug("Starting prospector")
	defer log.Debug("Prospector has stopped")

	defer p.stopHarvesterGroup(log, hg)

	var tg unison.MultiErrGroup

	tg.Go(func() error {
		p.filewatcher.Run(ctx.Cancelation)
		return nil
	})

	tg.Go(func() error {
		ignoreInactiveSince := getIgnoreSince(p.ignoreInactiveSince, ctx.Agent)

		for ctx.Cancelation.Err() == nil {
			fe := p.filewatcher.Event()

			if fe.Op == loginp.OpDone {
				return nil
			}

			src := p.identifier.GetSource(fe)
			p.onFSEvent(loggerWithEvent(log, fe, src), ctx, fe, src, s, hg, ignoreInactiveSince)

		}
		return nil
	})

	errs := tg.Wait()
	if len(errs) > 0 {
		log.Errorf("running prospector failed: %v", errors.Join(errs...))
	}
}

func (p *copyTruncateFileProspector) onFSEvent(
	log *logp.Logger,
	ctx input.Context,
	event loginp.FSEvent,
	src loginp.Source,
	updater loginp.StateMetadataUpdater,
	group loginp.HarvesterGroup,
	ignoreSince time.Time,
) {
	// TODO(AndersonQ): update to handle GZIP file
	switch event.Op {
	case loginp.OpCreate, loginp.OpWrite:
		switch event.Op {
		case loginp.OpCreate:
			log.Debugf("A new file %s has been found", event.NewPath)
		case loginp.OpWrite:
			log.Debugf("File %s has been updated", event.NewPath)
		}

		if p.fileProspector.isFileIgnored(log, event, ignoreSince) {
			return
		}

		if event.Op == loginp.OpCreate {
			err := updater.UpdateMetadata(src, fileMeta{Source: event.NewPath, IdentifierName: p.identifier.Name()})
			if err != nil {
				log.Errorf("Failed to set cursor meta data of entry %s: %v", src.Name(), err)
			}
		}

		// check if the event belongs to a rotated file
		if p.isRotated(event) {
			log.Debugf("File %s is rotated", event.NewPath)

			p.onRotatedFile(log, ctx, event, src, group)

		} else {
			log.Debugf("File %s is original", event.NewPath)
			// if file is original, add it to the bookeeper
			p.rotatedFiles.addOriginalFile(event.NewPath, src)

			group.Start(ctx, src)
		}

	case loginp.OpTruncate:
		if event.Descriptor.GZIP {
			// TODO(AndersonQ): will we keep this behaviour?
			log.Debugf("GZIP file %s has been truncated, stop ingesting it", event.NewPath)
			group.Stop(src)
			return
		}
		log.Debugf("File %s has been truncated", event.NewPath)

		err := updater.ResetCursor(src, state{Offset: 0})
		if err != nil {
			log.Errorf("failed to reset file cursor: %v", err)
		}
		group.Restart(ctx, src)

	case loginp.OpDelete:
		log.Debugf("File %s has been removed", event.OldPath)

		p.fileProspector.onRemove(log, event, src, updater, group)

	case loginp.OpRename:
		log.Debugf("File %s has been renamed to %s", event.OldPath, event.NewPath)

		if p.isRotated(event) {
			log.Debugf("File %s is rotated", event.NewPath)
			p.onRotatedFile(log, ctx, event, src, group)
		}

		p.fileProspector.onRename(log, ctx, event, src, updater, group)

	default:
		log.Error("Unknown return value %v", event.Op)
	}
}

func (p *copyTruncateFileProspector) isRotated(event loginp.FSEvent) bool {
	return p.rotatedSuffix.MatchString(event.NewPath)
}

func (p *copyTruncateFileProspector) onRotatedFile(
	log *logp.Logger,
	ctx input.Context,
	fe loginp.FSEvent,
	src loginp.Source,
	hg loginp.HarvesterGroup,
) {
	// Continue reading the rotated file from where we have left off with the original.
	// The original will be picked up again when updated and read from the beginning.
	originalPath := p.rotatedSuffix.ReplaceAllLiteralString(fe.NewPath, "")
	// if we haven't encountered the original file which was rotated, get its information
	if !p.rotatedFiles.isOriginalAdded(originalPath) {
		fi, err := os.Stat(originalPath)
		if err != nil {
			log.Errorf("Cannot continue file, error while getting the information of the original file: %+v", err)
			log.Debugf("Starting possibly rotated file from the beginning: %s", fe.NewPath)
			hg.Start(ctx, src)
			return
		}
		descCopy := fe.Descriptor
		descCopy.Info = file.ExtendFileInfo(fi)
		originalSrc := p.identifier.GetSource(loginp.FSEvent{NewPath: originalPath, Descriptor: descCopy})
		p.rotatedFiles.addOriginalFile(originalPath, originalSrc)
		p.rotatedFiles.addRotatedFile(originalPath, fe.NewPath, src)
		hg.Start(ctx, src)
		return
	}

	idx := p.rotatedFiles.addRotatedFile(originalPath, fe.NewPath, src)
	if idx == copiedFileIdx {
		// if a file is the freshest rotated file, continue reading from
		// where we have left off with the active file.
		previousSrc := p.rotatedFiles.table[originalPath].originalSrc
		hg.Continue(ctx, previousSrc, src)
	} else {
		// if a file is rotated but not the most fresh rotated file,
		// read it from where have left off.
		if fe.Op != loginp.OpRename {
			hg.Start(ctx, src)
		}
	}
}
