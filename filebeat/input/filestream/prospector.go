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
	"strings"
	"time"

	"github.com/urso/sderr"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/go-concert/unison"
)

const (
	prospectorDebugKey = "file_prospector"
)

// fileProspector implements the Prospector interface.
// It contains a file scanner which returns file system events.
// The FS events then trigger either new Harvester runs or updates
// the statestore.
type fileProspector struct {
	filewatcher       loginp.FSWatcher
	identifier        fileIdentifier
	ignoreOlder       time.Duration
	cleanRemoved      bool
	stateChangeCloser stateChangeCloserConfig
}

func (p *fileProspector) Init(inputPrefix string, userIDConfigured bool, cleaner loginp.ProspectorCleaner) error {
	if p.cleanRemoved && userIDConfigured {
		cleaner.CleanIf(func(key string, v loginp.Value) bool {
			if !strings.HasPrefix(key, inputPrefix) {
				return false
			}
			var fm fileMeta
			err := v.UnpackCursorMeta(&fm)
			if err != nil {
				// remove faulty entries
				return true
			}

			_, err = os.Stat(fm.Source)
			if err != nil {
				return true
			}
			return false
		})
	}

	identifierName := p.identifier.Name()
	cleaner.UpdateIdentifiers(func(key string, v loginp.Value) (string, interface{}) {
		if !strings.HasPrefix(key, inputPrefix) {
			return "", nil
		}

		var fm fileMeta
		err := v.UnpackCursorMeta(&fm)
		if err != nil {
			return "", nil
		}
		if fm.IdentifierName != identifierName {
			fi, err := os.Stat(fm.Source)
			if err != nil {
				return "", fm
			}
			newKey := p.identifier.GetSource(loginp.FSEvent{NewPath: fm.Source, Info: fi}).Name()
			fm.IdentifierName = identifierName
			return newKey, fm
		}
		return "", nil
	})

	return nil
}

// Run starts the fileProspector which accepts FS events from a file watcher.
func (p *fileProspector) Run(ctx input.Context, s loginp.StateMetadataUpdater, hg loginp.HarvesterGroup) {
	log := ctx.Logger.With("prospector", prospectorDebugKey)
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

				hg.Start(ctx, src)

			case loginp.OpDelete:
				log.Debugf("File %s has been removed", fe.OldPath)

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
				log.Debugf("File %s has been renamed to %s", fe.OldPath, fe.NewPath)

				// if file_identity is based on path, the current reader has to be cancelled
				// and a new one has to start.
				if !p.identifier.Supports(trackRename) {
					prevSrc := p.identifier.GetSource(loginp.FSEvent{NewPath: fe.OldPath})
					hg.Stop(prevSrc)

					log.Debugf("Remove state for file as file renamed and path file_identity is configured: %s", fe.OldPath)
					err := s.Remove(prevSrc)
					if err != nil {
						log.Errorf("Error while removing old state of renamed file (%s): %v", fe.OldPath, err)
					}

					hg.Start(ctx, src)
				} else {
					// update file metadata as the path has changed
					var meta fileMeta
					err := s.FindCursorMeta(src, meta)
					if err != nil {
						log.Errorf("Error while getting cursor meta data of entry %s: %v", src.Name(), err)
					}
					s.UpdateMetadata(src, fileMeta{Source: src.newPath, IdentifierName: meta.IdentifierName})

					if p.stateChangeCloser.Renamed {
						log.Debugf("Stopping harvester as file %s has been renamed and close.on_state_change.renamed is enabled.", src.Name())
						hg.Stop(src)
					}
				}

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

func (p *fileProspector) stopHarvesterGroup(log *logp.Logger, hg loginp.HarvesterGroup) {
	err := hg.StopGroup()
	if err != nil {
		log.Errorf("Error while stopping harverster group: %v", err)
	}
}

func (p *fileProspector) Test() error {
	panic("TODO: implement me")
}
