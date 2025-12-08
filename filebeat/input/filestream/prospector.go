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
	"strings"
	"time"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/unison"
)

type ignoreInactiveType uint8

const (
	InvalidIgnoreInactive = iota
	IgnoreInactiveSinceLastStart
	IgnoreInactiveSinceFirstStart

	ignoreInactiveSinceLastStartStr  = "since_last_start"
	ignoreInactiveSinceFirstStartStr = "since_first_start"
)

var ignoreInactiveSettings = map[string]ignoreInactiveType{
	ignoreInactiveSinceLastStartStr:  IgnoreInactiveSinceLastStart,
	ignoreInactiveSinceFirstStartStr: IgnoreInactiveSinceFirstStart,
}

var identifiersMap = map[string]fileIdentifier{}

func init() {
	for name, factory := range identifierFactories {
		if name == inodeMarkerName {
			// inode marker requires a specific config we cannot infer.
			continue
		}

		// only inode marker requires an active logger
		// passing nil logger for other identifier
		identifier, err := factory(nil, nil)
		if err != nil {
			// Skip identifiers we cannot create. E.g: inode_marker is not
			// supported on Windows
			continue
		}
		identifiersMap[name] = identifier
	}
}

// fileProspector implements the Prospector interface.
// It contains a file scanner which returns file system events.
// The FS events then trigger either new Harvester runs or updates
// the statestore.
type fileProspector struct {
	logger              *logp.Logger
	filewatcher         loginp.FSWatcher
	identifier          fileIdentifier
	ignoreOlder         time.Duration
	ignoreInactiveSince ignoreInactiveType
	cleanRemoved        bool
	stateChangeCloser   stateChangeCloserConfig
	takeOver            loginp.TakeOverConfig
}

func (p *fileProspector) Init(
	prospectorStore,
	globalStore loginp.StoreUpdater,
	newID func(loginp.Source) string,
) error {
	files := p.filewatcher.GetFiles()

	// If this fileProspector belongs to an input that did not have an ID
	// this will find its files in the registry and update them to use the
	// new ID.
	globalStore.UpdateIdentifiers(func(v loginp.Value) (id string, val interface{}) {
		var fm fileMeta
		err := v.UnpackCursorMeta(&fm)
		if err != nil {
			return "", nil
		}

		fd, ok := files[fm.Source]
		if !ok {
			return "", fm
		}

		newKey := newID(p.identifier.GetSource(loginp.FSEvent{NewPath: fm.Source, Descriptor: fd}))
		return newKey, fm
	})

	if p.cleanRemoved {
		prospectorStore.CleanIf(func(v loginp.Value) bool {
			var fm fileMeta
			err := v.UnpackCursorMeta(&fm)
			if err != nil {
				// remove faulty entries
				return true
			}

			_, ok := files[fm.Source]
			return !ok
		})
	}

	identifierName := p.identifier.Name()

	// If the file identity has changed to fingerprint, update the registry
	// keys so we can keep the state. This is only supported from file
	// identities that do not require configuration:
	//  - native (inode + device ID)
	//  - path
	if identifierName != fingerprintName {
		p.logger.Debugf("file identity is '%s', will not migrate registry", identifierName)
	} else {
		p.logger.Debug("trying to migrate file identity to fingerprint")
		prospectorStore.UpdateIdentifiers(func(v loginp.Value) (string, interface{}) {
			var fm fileMeta
			err := v.UnpackCursorMeta(&fm)
			if err != nil {
				return "", nil
			}

			fd, ok := files[fm.Source]
			if !ok {
				return "", fm
			}

			// Return early (do nothing) if:
			//  - The identifiers are the same
			//  - The old identifier is neither native nor path
			oldIdentifierName := fm.IdentifierName
			if oldIdentifierName == identifierName ||
				(oldIdentifierName != nativeName && oldIdentifierName != pathName) {
				return "", nil
			}

			// Our current file (source) is in the registry, now we need to ensure
			// this registry entry (resource) actually refers to our file. Sources
			// are identified by path. However, as log files rotate the same path
			// can point to a different file.
			//
			// So, to ensure we're dealing with the resource from our current file,
			// we use the old identifier to generate a registry key for the current
			// file we're trying to migrate, if this key matches with the key in the
			// registry, then we proceed to update the registry.
			registryKey := v.Key()
			oldIdentifier, ok := identifiersMap[oldIdentifierName]
			if !ok {
				// This should never happen, but we properly handle it just in case.
				// If we cannot find the identifier, move on to the next entry
				// some identifiers cannot be migrated
				p.logger.Errorf(
					"old file identity '%s' not found while migrating entry to "+
						"new file identity '%s'. If the file still exists, it will be re-ingested",
					oldIdentifierName,
					identifierName,
				)
				return "", nil
			}
			previousIdentifierKey := newID(oldIdentifier.GetSource(
				loginp.FSEvent{
					NewPath:    fm.Source,
					Descriptor: fd,
				}))

			// If the registry key and the key generated by the old identifier
			// do not match, log it at debug level and do nothing.
			if previousIdentifierKey != registryKey {
				return "", fm
			}

			// The resource matches the file we found in the file system, generate
			// a new registry key and return it alongside the updated meta.
			newKey := newID(p.identifier.GetSource(loginp.FSEvent{NewPath: fm.Source, Descriptor: fd}))
			fm.IdentifierName = identifierName
			p.logger.Infof("registry key: '%s' and previous file identity key: '%s', are the same, migrating. Source: '%s'",
				registryKey, previousIdentifierKey, fm.Source)

			return newKey, fm
		})
	}

	// Last, but not least, take over states if needed/enabled.
	if !p.takeOver.Enabled {
		return nil
	}

	// Take over states from other Filestream inputs or the log input
	prospectorStore.TakeOver(func(v loginp.Value) (string, interface{}) {
		var fm fileMeta
		err := v.UnpackCursorMeta(&fm)
		if err != nil {
			return "", nil
		}

		fd, ok := files[fm.Source]
		if !ok {
			return "", fm
		}

		// Return early (do nothing) if:
		//  - The old identifier is neither native, path or fingerprint
		oldIdentifierName := fm.IdentifierName
		if oldIdentifierName != nativeName &&
			oldIdentifierName != pathName &&
			oldIdentifierName != fingerprintName {
			return "", nil
		}

		// Our current file (source) is in the registry, now we need to ensure
		// this registry entry (resource) actually refers to our file. Sources
		// are identified by path, however as log files rotate the same path
		// can point to different files.
		//
		// So to ensure we're dealing with the resource from our current file,
		// we use the old identifier to generate a registry key for the current
		// file we're trying to migrate, if this key matches with the key in the
		// registry, then we proceed to update the registry.
		oldIdentifier, ok := identifiersMap[oldIdentifierName]
		if !ok {
			// This should never happen, but just in case we properly handle it.
			// If we cannot find the identifier, move on to the next entry
			// some identifiers cannot be migrated
			p.logger.Errorf(
				"old file identity '%s' not found while taking over old states, "+
					"new file identity '%s'. If the file still exists, it will be re-ingested",
				oldIdentifierName,
				identifierName,
			)
			return "", nil
		}

		fsEvent := loginp.FSEvent{
			NewPath:    fm.Source,
			Descriptor: fd,
		}
		split := strings.Split(v.Key(), "::")
		if len(split) != 4 {
			// This should never happen.
			p.logger.Errorf("registry key '%s' is in the wrong format, cannot migrate state", v.Key())
			return "", fm
		}

		idFromRegistry := strings.Join(split[2:], "::")
		idFromPreviousIdentity := oldIdentifier.GetSource(fsEvent).Name()
		if idFromPreviousIdentity != idFromRegistry {
			return "", fm
		}

		newKey := newID(p.identifier.GetSource(loginp.FSEvent{NewPath: fm.Source, Descriptor: fd}))
		fm.IdentifierName = identifierName
		p.logger.Infof("Taking over state: '%s' -> '%s'", v.Key(), newKey)
		return newKey, fm
	})

	return nil
}

// Run starts the fileProspector which accepts FS events from a file watcher.
//
//nolint:dupl // Different prospectors have a similar run method
func (p *fileProspector) Run(ctx input.Context, s loginp.StateMetadataUpdater, hg loginp.HarvesterGroup) {
	p.logger.Debug("Starting prospector")
	defer p.logger.Debug("Prospector has stopped")

	// ctx.Logger has its 'log.logger' set to 'input.filestream'.
	// Because the harvester is not really part of the prospector,
	// we use this logger instead of the prospector logger.
	defer p.stopHarvesterGroup(ctx.Logger, hg)

	var tg unison.MultiErrGroup

	// The harvester needs to notify the FileWatcher
	// when it closes
	hg.SetObserver(p.filewatcher.NotifyChan())

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
			p.onFSEvent(loggerWithEvent(p.logger, fe, src), ctx, fe, src, s, hg, ignoreInactiveSince)
		}
		return nil
	})

	errs := tg.Wait()
	if len(errs) > 0 {
		p.logger.Errorf("running prospector failed: %v", errors.Join(errs...))
	}
}

// onFSEvent uses 'log' instead of the [fileProspector] logger
// because 'log' has been enriched with event information
func (p *fileProspector) onFSEvent(
	log *logp.Logger,
	ctx input.Context,
	event loginp.FSEvent,
	src loginp.Source,
	updater loginp.StateMetadataUpdater,
	group loginp.HarvesterGroup,
	ignoreSince time.Time,
) {

	log = log.With("source_file", event.SrcID)

	// For growing_fingerprint, handle prefix matching and migration
	if p.identifier.Name() == growingFingerprintName {
		src = p.handleGrowingFingerprintLookup(log, event, src, updater)
	}

	switch event.Op {
	case loginp.OpCreate, loginp.OpWrite, loginp.OpNotChanged:
		switch event.Op {
		case loginp.OpCreate:
			log.Debugf("A new file %s has been found", event.NewPath)

			err := updater.UpdateMetadata(src, fileMeta{Source: event.NewPath, IdentifierName: p.identifier.Name()})
			if err != nil {
				log.Errorf("Failed to set cursor meta data of entry %s: %v", src.Name(), err)
			}

		case loginp.OpWrite:
			log.Debugf("File %s has been updated", event.NewPath)

		case loginp.OpNotChanged:
			log.Debugf("File %s has not changed, trying to start new harvester", event.NewPath)
		}

		if p.isFileIgnored(log, event, ignoreSince) {
			err := updater.ResetCursor(src, state{Offset: event.Descriptor.Info.Size()})
			if err != nil {
				log.Errorf("setting cursor for ignored file: %v", err)
			}
			return
		}

		// Note: For growing_fingerprint, migration updates the key in-place.
		// The harvester manager tracks by resource pointer, so if a harvester
		// is already running on this resource (even with old key), it will
		// be detected as "Harvester already running".
		group.Start(ctx, src)

	case loginp.OpTruncate:
		log.Debugf("File %s has been truncated setting offset to 0", event.NewPath)

		err := updater.ResetCursor(src, state{Offset: 0})
		if err != nil {
			log.Errorf("resetting cursor on truncated file: %v", err)
		}
		group.Restart(ctx, src)

	case loginp.OpDelete:
		log.Debugf("File %s has been removed", event.OldPath)

		p.onRemove(log, event, src, updater, group)

	case loginp.OpRename:
		log.Debugf("File %s has been renamed to %s", event.OldPath, event.NewPath)

		p.onRename(log, ctx, event, src, updater, group)

	default:
		log.Errorf("Unknown operation '%s'", event.Op.String())
	}
}

func (p *fileProspector) isFileIgnored(log *logp.Logger, fe loginp.FSEvent, ignoreInactiveSince time.Time) bool {
	if p.ignoreOlder > 0 {
		now := time.Now()
		if now.Sub(fe.Descriptor.Info.ModTime()) > p.ignoreOlder {
			log.Debugf("Ignore file because ignore_older reached. File %s", fe.NewPath)
			return true
		}
	}
	if !ignoreInactiveSince.IsZero() && fe.Descriptor.Info.ModTime().Sub(ignoreInactiveSince) <= 0 {
		log.Debugf("Ignore file because ignore_since.* reached time %v. File %s", p.ignoreInactiveSince, fe.NewPath)
		return true
	}
	return false
}

func (p *fileProspector) onRemove(log *logp.Logger, fe loginp.FSEvent, src loginp.Source, s loginp.StateMetadataUpdater, hg loginp.HarvesterGroup) {
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
}

func (p *fileProspector) onRename(log *logp.Logger, ctx input.Context, fe loginp.FSEvent, src loginp.Source, s loginp.StateMetadataUpdater, hg loginp.HarvesterGroup) {
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
		err := s.FindCursorMeta(src, &meta)
		if err != nil {
			meta.IdentifierName = p.identifier.Name()
			log.Warnf(
				"Error while getting cursor meta data of entry '%s': '%v', using prospector's identifier: '%s'",
				src.Name(), err, meta.IdentifierName)
		}
		err = s.UpdateMetadata(src, fileMeta{Source: fe.NewPath, IdentifierName: meta.IdentifierName})
		if err != nil {
			log.Errorf("Failed to update cursor meta data of entry %s: %v", src.Name(), err)
		}

		if p.stateChangeCloser.Renamed {
			log.Debugf("Stopping harvester as file %s has been renamed and close.on_state_change.renamed is enabled.", src.Name())

			fe.Op = loginp.OpDelete
			srcToClose := p.identifier.GetSource(fe)
			hg.Stop(srcToClose)
		}
	}
}

func (p *fileProspector) stopHarvesterGroup(log *logp.Logger, hg loginp.HarvesterGroup) {
	err := hg.StopHarvesters()
	if err != nil {
		log.Errorf("Error while stopping harvester group: %v", err)
	}
}

func (p *fileProspector) Test() error {
	panic("TODO: implement me")
}

func getIgnoreSince(t ignoreInactiveType, info beat.Info) time.Time {
	switch t {
	case IgnoreInactiveSinceLastStart:
		return info.StartTime
	case IgnoreInactiveSinceFirstStart:
		return info.FirstStart
	default:
		return time.Time{}
	}
}

func (t *ignoreInactiveType) Unpack(v string) error {
	val, ok := ignoreInactiveSettings[v]
	if !ok {
		return fmt.Errorf("invalid ignore_inactive setting: %s", v)
	}
	*t = val
	return nil
}

// handleGrowingFingerprintLookup handles the special lookup logic for
// growing_fingerprint identity.
func (p *fileProspector) handleGrowingFingerprintLookup(
	log *logp.Logger,
	event loginp.FSEvent,
	src loginp.Source,
	updater loginp.StateMetadataUpdater) loginp.Source {
	// Empty fingerprint - nothing to match
	if event.Descriptor.Fingerprint == "" {
		return src
	}

	// Try to find a prefix match (file may have grown)
	oldKey, found := p.findGrowingFingerprintMatch(updater, event.Descriptor.Fingerprint, event.NewPath)
	if !found {
		return src
	}

	// Found a prefix match - migrate to new key
	if err := p.migrateGrowingFingerprint(updater, oldKey, src, event); err != nil {
		log.Errorf("failed to migrate growing fingerprint: %v", err)
		// Continue anyway - might create duplicate, but better than losing data
		return src
	}

	// Migration succeeded - the old harvester is still running and will continue
	// reading. We should NOT start a new harvester.
	return src
}

// findGrowingFingerprintMatch looks for an existing registry entry whose
// fingerprint is a prefix of the current file's fingerprint. This handles
// the case where a file has grown since the last scan.
func (p *fileProspector) findGrowingFingerprintMatch(
	updater loginp.StateMetadataUpdater,
	currentFingerprint string,
	currentPath string,
) (oldKey string, found bool) {
	if currentFingerprint == "" {
		return "", false
	}

	var bestMatchKey string
	var bestMatchLen int

	// Use the IterateOnPrefix method to find potential matches
	updater.IterateOnPrefix(func(key string, meta interface{}) bool {
		// Only process growing_fingerprint keys
		steps := strings.Split(key, identitySep)
		if len(steps) > 4 {
			return true // continue iteration
		}
		if steps[2] != growingFingerprintName {
			return true // continue iteration
		}

		// Extract the fingerprint from the key
		storedFingerprint := steps[3]
		if storedFingerprint == "" {
			return true // continue iteration - empty fingerprint
		}

		// Skip if stored fingerprint is same size or larger than current
		// (can only match if stored is a prefix of current)
		if len(storedFingerprint) >= len(currentFingerprint) {
			return true // continue iteration
		}

		// Check if stored fingerprint is a prefix of current
		if !strings.HasPrefix(currentFingerprint, storedFingerprint) {
			return true // continue iteration - not a prefix
		}

		// Check path match via metadata
		var fm fileMeta
		if err := convertToFileMeta(meta, &fm); err != nil {
			return true // continue iteration
		}

		// IMPORTANT: Only match if the file path is the same.
		// This prevents migrating entries that belong to different files
		// (which can happen with collisions where multiple files have the
		// same fingerprint initially).
		if fm.Source != currentPath {
			return true // continue iteration - different file
		}

		// Found a match - keep track of the largest (most specific) match
		if len(storedFingerprint) > bestMatchLen {
			bestMatchKey = key
			bestMatchLen = len(storedFingerprint)
		}

		return true // continue iteration
	})

	if bestMatchKey != "" {
		p.logger.Debugf(
			"found growing fingerprint prefix match for %s: %s (size %d -> %d bytes)",
			currentPath, bestMatchKey, bestMatchLen/2, len(currentFingerprint)/2,
		)
		return bestMatchKey, true
	}

	return "", false
}

// migrateGrowingFingerprint migrates a registry entry from an old key to a new key.
// This is called when a file's fingerprint has grown.
func (p *fileProspector) migrateGrowingFingerprint(
	updater loginp.StateMetadataUpdater,
	oldKey string,
	newSrc loginp.Source,
	event loginp.FSEvent,
) error {
	newMeta := fileMeta{
		Source:         event.NewPath,
		IdentifierName: growingFingerprintName,
	}

	prefix := strings.Split(oldKey, growingFingerprintName)
	if len(prefix) != 2 {
		return fmt.Errorf("invalid old key format: %s", oldKey)
	}
	newKey := prefix[0] + newSrc.Name()

	err := updater.UpdateKey(oldKey, newKey, newMeta)
	if err != nil {
		return fmt.Errorf("failed to migrate growing fingerprint from %s to %s: %w", oldKey, newKey, err)
	}

	p.logger.Infof("migrated growing fingerprint entry: %s -> %s", oldKey, newKey)
	return nil
}

// convertToFileMeta converts an interface{} to fileMeta using type conversion.
// This is needed because the metadata is stored as interface{} in the store.
func convertToFileMeta(meta interface{}, fm *fileMeta) error {
	if meta == nil {
		return fmt.Errorf("meta is nil")
	}

	// Try direct type assertion first
	if m, ok := meta.(fileMeta); ok {
		*fm = m
		return nil
	}

	// Try map conversion (common when loaded from JSON)
	// TODO(AndersonQ): is it really needed?
	if m, ok := meta.(map[string]interface{}); ok {
		if source, ok := m["source"].(string); ok {
			fm.Source = source
		}
		if identifierName, ok := m["identifier_name"].(string); ok {
			fm.IdentifierName = identifierName
		}
		return nil
	}

	return fmt.Errorf("cannot convert %T to fileMeta", meta)
}
