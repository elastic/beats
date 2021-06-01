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
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	"github.com/urso/sderr"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/go-concert/unison"
)

const (
	copyTruncateProspectorDebugKey = "copy_truncate_file_prospector"
	copiedFileIdx                  = 0
)

// sorter is required for ordering rotated log files
type sorter interface {
	sort([]rotatedFileInfo)
}

// rotatedFileInfo stores the file information of a rotated file.
type rotatedFileInfo struct {
	path string
	src  loginp.Source
	ts   time.Time
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

func newRotatedFilestreams(cfg *copyTruncateConfig) *rotatedFilestreams {
	var sorter sorter
	sorter = &defaultSorter{}
	if cfg.DateFormat != "" {
		sorter = &dateSorter{cfg.DateFormat}
	}
	return &rotatedFilestreams{
		table:  make(map[string]*rotatedFilestream, 0),
		count:  cfg.Count,
		sorter: sorter,
	}
}

// defaultSorter sorts rotated log files that have a numeric suffix.
// Example: apache.log.1, apache.log.2
type defaultSorter struct{}

func (s *defaultSorter) sort(files []rotatedFileInfo) {
	sort.Slice(
		files,
		func(i, j int) bool {
			return files[i].path < files[j].path
		},
	)
}

// dateSorter sorts rotated log files that have a date suffix
// based on the configured format.
// Example: apache.log-21210526, apache.log-20210527
type dateSorter struct {
	format string
}

func (s *dateSorter) sort(files []rotatedFileInfo) {
	sort.Slice(
		files,
		func(i, j int) bool {
			return s.GetTs(&files[i]).After(s.GetTs(&files[j]))
		},
	)
}

func (s *dateSorter) GetTs(fi *rotatedFileInfo) time.Time {
	if !fi.ts.IsZero() {
		return fi.ts
	}
	filename := filepath.Base(fi.path)
	fileTs := (filename[len(filename)-len(s.format):])

	ts, err := time.Parse(s.format, fileTs)
	if err != nil {
		return time.Time{}
	}
	fi.ts = ts
	return ts
}

// rotatedFilestreams is a map of original files and their rotated instances.
type rotatedFilestreams struct {
	table  map[string]*rotatedFilestream
	count  int
	sorter sorter
}

// addOriginalFile adds a new original file and its identifying information
// to the bookkeeper.
func (r rotatedFilestreams) addOriginalFile(path string, src loginp.Source) {
	if _, ok := r.table[path]; ok {
		return
	}
	r.table[path] = &rotatedFilestream{originalSrc: src, rotated: make([]rotatedFileInfo, 0, r.count)}
}

// isOriginalAdded checks if an original file has been found.
func (r rotatedFilestreams) isOriginalAdded(path string) bool {
	_, ok := r.table[path]
	return ok
}

// originalSrc returns the original Source information of a given
// original file path.
func (r rotatedFilestreams) originalSrc(path string) loginp.Source {
	return r.table[path].originalSrc
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

	r.table[original].rotated = append(r.table[original].rotated, rotatedFileInfo{rotated, src, time.Time{}})
	r.sorter.sort(r.table[original].rotated)

	for idx, fi := range r.table[original].rotated {
		if fi.path == rotated {
			return idx
		}
	}

	return -1
}

// addRotatedFile adds a new rotated file to the list and returns its index.
// if a file is already added, the source is updated and the index is returned.
func (r rotatedFilestreams) removeRotatedFile(original, rotated string) {
	for idx, fi := range r.table[original].rotated {
		if fi.path == rotated {
			r.table[original].rotated = append(r.table[original].rotated[:idx], r.table[original].rotated[idx+1:]...)
			return
		}
	}
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
			switch fe.Op {
			case loginp.OpCreate, loginp.OpWrite:
				if fe.Op == loginp.OpCreate {
					log.Debugf("A new file %s has been found", fe.NewPath)

					err := s.UpdateMetadata(src, fileMeta{Source: fe.NewPath, IdentifierName: p.identifier.Name()})
					if err != nil {
						log.Errorf("Failed to set cursor meta data of entry %s: %v", src.Name(), err)
					}

				} else if fe.Op == loginp.OpWrite {
					log.Debugf("File %s has been updated", fe.NewPath)
				}

				if p.fileProspector.isFileIgnored(log, fe, ignoreInactiveSince) {
					break
				}

				// check if the event belongs to a rotated file
				if p.isRotated(fe) {
					log.Debugf("File %s is rotated", fe.NewPath)

					p.onRotatedFile(log, ctx, fe, src, hg)

				} else {
					log.Debugf("File %s is original", fe.NewPath)
					// if file is original, add it to the bookeeper
					p.rotatedFiles.addOriginalFile(fe.NewPath, src)

					hg.Start(ctx, src)
				}

			case loginp.OpTruncate:
				log.Debugf("File %s has been truncated", fe.NewPath)

				s.ResetCursor(src, state{Offset: 0})
				hg.Restart(ctx, src)

			case loginp.OpDelete:
				log.Debugf("File %s has been removed", fe.OldPath)

				// if file is rotated, stop harvester and clean up state
				if p.isRotated(fe) {
					log.Debugf("Stopping harvester as rotated file %s has been removed.", src.Name())

					hg.Stop(src)

					log.Debugf("Remove state for file as rotated file has been removed: %s", fe.OldPath)

					err := s.Remove(src)
					if err != nil {
						log.Errorf("Error while removing state from statestore: %v", err)
					}

					originalPath := p.rotatedSuffix.ReplaceAllLiteralString(fe.OldPath, "")
					p.rotatedFiles.removeRotatedFile(originalPath, fe.OldPath)
				} else {
					p.fileProspector.onRemove(log, fe, src, s, hg)
				}

			case loginp.OpRename:
				log.Debugf("File %s has been renamed to %s", fe.OldPath, fe.NewPath)

				// check if the event belongs to a rotated file
				if p.isRotated(fe) {
					log.Debugf("File %s is rotated", fe.NewPath)

					p.onRotatedFile(log, ctx, fe, src, hg)
				}

				p.fileProspector.onRename(log, ctx, fe, src, s, hg)

			default:
				log.Error("Unkown return value %v", fe.Op)
			}
		}
		return nil
	})

	errs := tg.Wait()
	if len(errs) > 0 {
		log.Error("%s", sderr.WrapAll(errs, "running prospector failed"))
	}
}

func (p *copyTruncateFileProspector) isRotated(event loginp.FSEvent) bool {
	if p.rotatedSuffix.MatchString(event.NewPath) {
		return true
	}
	return false
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
		originalSrc := p.identifier.GetSource(loginp.FSEvent{NewPath: originalPath, Info: fi})
		p.rotatedFiles.addOriginalFile(originalPath, originalSrc)
		p.rotatedFiles.addRotatedFile(originalPath, fe.NewPath, src)
		hg.Start(ctx, src)
		return
	}

	idx := p.rotatedFiles.addRotatedFile(originalPath, fe.NewPath, src)
	if idx == copiedFileIdx {
		// if a file is the most fresh rotated file, continue reading from
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
