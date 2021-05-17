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
	"regexp"
	"sort"
	"time"

	"github.com/urso/sderr"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/go-concert/unison"
)

const (
	copyTruncateProspectorDebugKey = "copy_truncate_file_prospector"
)

type copyTruncateConfig struct {
	commonRotationConfig `config:",inline"`
}

// rotatedFileInfo stores the file information of a rotated file.
type rotatedFileInfo struct {
	path string
	src  loginp.Source
}

// rotatedFileGroup includes the information of the original file
// and its identifier, and the list of its rotated files.
type rotatedFileGroup struct {
	originalSrc loginp.Source
	rotated     []rotatedFileInfo
	count       int
}

func newRotatedFiles(config copyTruncateConfig) *rotatedFiles {
	return &rotatedFiles{
		table:            make(map[string]*rotatedFileGroup, 0),
		maxRotationCount: config.Rotate,
	}
}

// rotatedFiles is a map of original files and their rotated instances.
type rotatedFiles struct {
	table            map[string]*rotatedFileGroup
	maxRotationCount int
}

// addOriginalFile adds a new original file and its identifying information
// to the bookkeeper.
func (r rotatedFiles) addOriginalFile(path string, src loginp.Source) {
	if _, ok := r.table[path]; ok {
		return
	}
	r.table[path] = &rotatedFileGroup{originalSrc: src, rotated: make([]rotatedFileInfo, r.maxRotationCount)}
}

// isOriginalAdded checks if an original file has been found.
func (r rotatedFiles) isOriginalAdded(path string) bool {
	_, ok := r.table[path]
	return ok
}

// originalSrc returns the original Source information of a given
// original file path.
func (r rotatedFiles) originalSrc(path string) loginp.Source {
	return r.table[path].originalSrc
}

// previousSrc returns the source identifier for the previous file, so its state
// information can be used when continuing on the rotated instance.
func (r rotatedFiles) previousSrc(originalPath string, idx int) loginp.Source {
	if idx == 0 {
		return r.originalSrc(originalPath)
	}
	return r.table[originalPath].rotated[idx-1].src
}

// addRotatedFile adds a new rotated file to the list and returns its index.
func (r rotatedFiles) addRotatedFile(original, rotated string, src loginp.Source) int {
	for i, info := range r.table[original].rotated {
		if info.path == rotated {
			return i
		}
	}

	r.table[original].rotated = append(r.table[original].rotated, rotatedFileInfo{rotated, src})
	sort.Slice(r.table[original].rotated, func(i, j int) bool {
		return r.table[original].rotated[i].path < r.table[original].rotated[j].path
	})

	for i, info := range r.table[original].rotated {
		if info.path == rotated {
			return i
		}
	}
	return -1
}

type copyTruncateFileProspector struct {
	fileProspector
	rotatedSuffix *regexp.Regexp
	rotatedFiles  rotatedFiles
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
				} else if fe.Op == loginp.OpWrite {
					log.Debugf("File %s has been updated", fe.NewPath)
				}

				if p.ignoreOlder > 0 {
					now := time.Now()
					if now.Sub(fe.Info.ModTime()) > p.ignoreOlder {
						log.Debugf("Ignore file because ignore_older reached. File %s", fe.NewPath)
						break
					}
				}

				// check if the event belongs to a rotated file
				if p.isRotated(fe) {
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
							continue
						}
						originalSrc := p.identifier.GetSource(loginp.FSEvent{NewPath: originalPath, Info: fi})
						p.rotatedFiles.addOriginalFile(originalPath, originalSrc)
					}
					currentIdx := p.rotatedFiles.addRotatedFile(originalPath, fe.NewPath, src)
					previousSrc := p.rotatedFiles.previousSrc(originalPath, currentIdx)
					hg.Continue(ctx, previousSrc, src)

				} else {
					// if file is original, add it to the bookeeper
					p.rotatedFiles.addOriginalFile(fe.NewPath, src)

					hg.Start(ctx, src)
				}

			case loginp.OpDelete:
				log.Debugf("File %s has been removed", fe.OldPath)

				if p.rotatedSuffix.MatchString(fe.OldPath) {
					originalPath := p.rotatedSuffix.ReplaceAllLiteralString(fe.OldPath, "")
					rotatedFilesCount := len(p.rotatedFiles.table[originalPath].rotated)
					if fe.OldPath == p.rotatedFiles.table[originalPath].rotated[rotatedFilesCount-1].path {
						p.rotatedFiles.table[originalPath].rotated = p.rotatedFiles.table[originalPath].rotated[:rotatedFilesCount-1]
					} else {
						log.Debug("Unexpected rotated file has been removed.")
					}
				} else {
					log.Debug("Original file has been removed unexpectedly.")
					delete(p.rotatedFiles.table, fe.OldPath)
				}

				if p.stateChangeCloser.Removed {
					log.Debugf("Stopping harvester as file %s has been removed and close.on_state_change.removed is enabled.", src.Name())
					hg.Stop(src)
				}

				if p.cleanRemoved {
					log.Debugf("Remove state for file as file removed: %s", fe.OldPath)

					err := s.Remove(src)
					if err != nil {
						log.Errorf("Error while removing state from statestore: %v", err)
					}
				}

			case loginp.OpRename:
				// Renames are not supported when using copytruncate method.

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
	if event.Op != loginp.OpCreate {
		return false
	}

	if p.rotatedSuffix.MatchString(event.NewPath) {
		return true
	}
	return false
}
