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

type rotatedFileInfo struct {
	path string
	src  loginp.Source
}

type rotatedFileGroup struct {
	originalSrc loginp.Source
	rotated     []rotatedFileInfo
}

// addRotatedFile adds a new rotated file to the list and returns its index.
func (r *rotatedFileGroup) addRotatedFile(path string, src loginp.Source) int {
	for i, info := range r.rotated {
		if info.path == path {
			return i
		}
	}

	r.rotated = append(r.rotated, rotatedFileInfo{path, src})
	sort.Slice(r.rotated, func(i, j int) bool {
		return r.rotated[i].path < r.rotated[j].path
	})

	for i, info := range r.rotated {
		if info.path == path {
			return i
		}
	}
	return -1
}

type copyTruncateFileProspector struct {
	fileProspector
	rotatedSuffix *regexp.Regexp
	rotatedFiles  map[string]*rotatedFileGroup
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
					if _, ok := p.rotatedFiles[originalPath]; !ok {
						fi, err := os.Stat(originalPath)
						if err != nil {
							log.Errorf("Cannot continue file, error while getting the information of the original file: %+v", err)
							log.Debugf("Starting possibly rotated file from the beginning: %s", fe.NewPath)
							hg.Start(ctx, src)
							continue
						}
						originalSrc := p.identifier.GetSource(loginp.FSEvent{NewPath: originalPath, Info: fi})
						p.rotatedFiles[originalPath] = &rotatedFileGroup{originalSrc: originalSrc, rotated: make([]rotatedFileInfo, 0)}
					}
					currentIdx := p.rotatedFiles[originalPath].addRotatedFile(fe.NewPath, src)
					previousSrc := p.rotatedFiles[originalPath].originalSrc
					if currentIdx != 0 {
						previousSrc = p.rotatedFiles[originalPath].rotated[currentIdx-1].src
					}
					hg.Continue(ctx, previousSrc, src)
				} else {
					// if file is original, add it to the bookeeper
					if _, ok := p.rotatedFiles[fe.NewPath]; !ok {
						p.rotatedFiles[fe.NewPath] = &rotatedFileGroup{originalSrc: src, rotated: make([]rotatedFileInfo, 0)}
					}
					hg.Start(ctx, src)
				}

			case loginp.OpDelete:
				log.Debugf("File %s has been removed", fe.OldPath)

				if p.rotatedSuffix.MatchString(fe.OldPath) {
					originalPath := p.rotatedSuffix.ReplaceAllLiteralString(fe.OldPath, "")
					rotatedFilesCount := len(p.rotatedFiles[originalPath].rotated)
					if fe.OldPath == p.rotatedFiles[originalPath].rotated[rotatedFilesCount-1].path {
						p.rotatedFiles[originalPath].rotated = p.rotatedFiles[originalPath].rotated[:rotatedFilesCount-1]
					} else {
						log.Debug("Unexpected rotated file has been removed.")
					}
				} else {
					log.Debug("Original file has been removed unexpectedly.")
					delete(p.rotatedFiles, fe.OldPath)
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
