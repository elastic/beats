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
	"strings"
	"time"

	"github.com/elastic/beats/v7/filebeat/input/file"
	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/transform/typeconv"
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
	logger                *logp.Logger
	filewatcher           loginp.FSWatcher
	identifier            fileIdentifier
	ignoreOlder           time.Duration
	ignoreInactiveSince   ignoreInactiveType
	cleanRemoved          bool
	stateChangeCloser     stateChangeCloserConfig
	takeOver              loginp.TakeOverConfig
	filestreamIdentifiers map[string]fileIdentifier
	logIdentifiers        map[string]file.StateIdentifier
	shortFingerprints     *shortFingerprintSet
	growingFingerprint    bool
}

func (p *fileProspector) previousID(name string, fd loginp.FileDescriptor, v loginp.TakeOverState) string {
	if p.takeOver.FromFilestream() {
		fsEvent := loginp.FSEvent{
			NewPath:    v.Source,
			Descriptor: fd,
		}

		return p.filestreamIdentifiers[name].GetSource(fsEvent).Name()
	}

	state := file.State{
		FileStateOS: v.FileStateOS,
		Source:      v.Source,
	}

	// The stream field is used when generating the ID, so if takeOver has
	// a stream set, we use it, so the ID matches the input we're taking over.
	if p.takeOver.Stream == "stdout" || p.takeOver.Stream == "stderr" {
		state.Meta = map[string]string{
			"stream": p.takeOver.Stream,
		}
	}

	id, _ := p.logIdentifiers[name].GenerateID(state)
	return id
}

func (p *fileProspector) takeOverFn(
	v loginp.TakeOverState,
	files map[string]loginp.FileDescriptor,
	newID func(loginp.Source) string,
) (string, any) {
	fm := fileMeta{
		Source:         v.Source,
		IdentifierName: v.IdentifierName,
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
	rk, ok := parseRegistryKey(v.Key)
	if !ok {
		// This should never happen.
		p.logger.Errorf("registry key '%s' is in the wrong format, cannot migrate state", v.Key)
		return "", fm
	}

	idFromRegistry := rk.identity()
	idFromPreviousIdentity := p.previousID(oldIdentifierName, fd, v)

	if idFromPreviousIdentity != idFromRegistry {
		return "", fm
	}

	newKey := newID(p.identifier.GetSource(loginp.FSEvent{NewPath: fm.Source, Descriptor: fd}))
	fm.IdentifierName = p.identifier.Name()
	p.logger.Infof("Taking over state: '%s' -> '%s'", v.Key, newKey)
	return newKey, fm
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
	globalStore.UpdateIdentifiers(func(v loginp.Value) (id string, val any) {
		var fm fileMeta
		err := v.UnpackCursorMeta(&fm)
		if err != nil {
			return "", nil
		}

		fd, ok := files[fm.Source]
		if !ok {
			return "", fm
		}

		rk, ok := parseRegistryKey(v.Key())
		// Wrong key format
		if !ok {
			return "", fm
		}

		registryFileIdentity := rk.identity()
		fileIdentity := p.identifier.GetSource(loginp.FSEvent{
			NewPath:    fm.Source,
			Descriptor: fd,
		}).Name()

		// Same paths, different file, do not migrate ID
		if registryFileIdentity != fileIdentity {
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

			if _, ok := files[fm.Source]; ok {
				return false // source still present, keep
			}

			// Growing entry whose Source path is absent from the current
			// scan: could be a true deletion (cleanup removes it) or a
			// rename while filebeat was stopped (the entry needs to stay
			// alive so the next scan's migrate path can pick it up).
			//
			// We preserve ONLY when there's a current file that has already
			// crossed the threshold (Fingerprint.Complete()) and whose raw
			// header has this entry's stored fingerprint as a STRICT prefix.
			// A completed file carrying raw material that extends a stored
			// growing entry is strong evidence of a rename + threshold
			// crossing while filebeat was stopped.
			//
			// Ordinary growing-phase rename across restart is intentionally
			// NOT covered by this skip: distinguishing it from a
			// shared-header collision (two distinct files starting with
			// the same content, one of them deleted) is ambiguous without
			// the threshold signal, and preserving in that case would
			// cause incorrect state reuse for the surviving file.
			// A non-empty Fingerprint (held in the entry VALUE, since the key
			// is only a bounded hash of it) marks an entry as still growing.
			// Final/static entries leave it empty and are not eligible for
			// this rename+threshold-crossing preservation.
			storedFP := fm.Fingerprint
			if storedFP == "" {
				return true
			}
			for _, desc := range files {
				if desc.Fingerprint.Complete() && isStrictPrefix(desc.Fingerprint.Raw, storedFP) {
					return false // possible rename + threshold crossing, preserve
				}
			}
			return true
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
		prospectorStore.UpdateIdentifiers(func(v loginp.Value) (string, any) {
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
			key := v.Key()
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
			if previousIdentifierKey != key {
				return "", fm
			}

			// The resource matches the file we found in the file system, generate
			// a new registry key and return it alongside the updated meta.
			newKey := newID(p.identifier.GetSource(loginp.FSEvent{NewPath: fm.Source, Descriptor: fd}))
			fm.IdentifierName = identifierName
			p.logger.Infof("registry key: '%s' and previous file identity key: '%s', are the same, migrating. Source: '%s'",
				key, previousIdentifierKey, fm.Source)

			return newKey, fm
		})
	}

	return nil
}

// TakeOver migrates states from other inputs (Log input or other Filestream
// inputs with different IDs) to this input. It must be called after Init and
// before Run so that it is not triggered during CheckConfig validation.
func (p *fileProspector) TakeOver(prospectorStore loginp.StoreUpdater, newID func(loginp.Source) string) error {
	if !p.takeOver.Enabled {
		return nil
	}

	files := p.filewatcher.GetFiles()

	// Take over states from other Filestream inputs or the log input
	prospectorStore.TakeOver(func(v loginp.TakeOverState) (string, any) {
		return p.takeOverFn(v, files, newID)
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
			p.onFSEvent(loggerWithEvent(p.logger, fe), ctx, fe, src, s, hg, ignoreInactiveSince)
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
	// For growing fingerprint mode, handle prefix matching and migration.
	// Skip for OpRename: handleGrowingFingerprintLookup assumes event.SrcID is
	// the current identity — its KeyExists fast path returns true for the old key
	// and skips migration. OpRename migration is handled in the OpRename case below.
	if p.growingFingerprint && event.Op != loginp.OpRename {
		src = p.handleGrowingFingerprintLookup(log, event, src, updater)
	}

	switch event.Op {
	case loginp.OpCreate, loginp.OpWrite, loginp.OpNotChanged:
		switch event.Op {
		case loginp.OpCreate:
			log.Debugf("A new file %s has been found", event.NewPath)

			err := updater.UpdateMetadata(src, fileMeta{
				Source:         event.NewPath,
				IdentifierName: p.identifier.Name(),
				Fingerprint:    growingRawFingerprint(event.Descriptor),
			})
			if err != nil {
				log.Errorf("Failed to set cursor meta data of entry %s: %v", src.Name(), err)
			}

			p.indexGrowingFingerprint(event.SrcID, event.Descriptor, event.NewPath)

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

		// Note: In growing fingerprint mode, migration updates the key in-place.
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

		// Remove stale short fingerprint entry by source path.
		// We can't use event.SrcID because truncation changes the fingerprint,
		// so the SrcID is based on the truncated content, not the old fingerprint.
		p.shortFingerprints.RemoveBySource(event.NewPath)

	case loginp.OpDelete:
		log.Debugf("File %s has been removed", event.OldPath)

		p.onRemove(log, event, src, updater, group)
		p.shortFingerprints.Remove(event.SrcID)

	case loginp.OpRename:
		log.Debugf("File %s has been renamed to %s", event.OldPath, event.NewPath)

		// For growing fingerprint mode: if the fingerprint grew during the rename,
		// migrate the registry key BEFORE onRename. (findGrowingFingerprintMatch
		// matches on event.OldPath for renames, since shortFingerprints still
		// holds the old source path at this point.)
		// Migration must happen first so onRename's UpdateMetadata finds the
		// entry under the new key (which uses the new fingerprint from src).
		if p.growingFingerprint {
			oldKey, found := p.findGrowingFingerprintMatch(updater, event)
			if found {
				newKey, err := p.migrateGrowingFingerprint(updater, oldKey, src, event)
				if err != nil {
					log.Errorf("failed to migrate growing fingerprint on rename: %v", err)
				} else {
					p.shortFingerprints.Remove(oldKey)
					p.indexGrowingFingerprint(newKey, event.Descriptor, event.NewPath)
				}
			}
		}

		p.onRename(log, ctx, event, src, updater, group)

		// Update source path for non-migrated short fingerprint entries
		// (e.g., rename without fingerprint growth, or non-growing identities).
		p.shortFingerprints.UpdateSource(event.SrcID, event.NewPath)

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
		// The path changed; update the persisted metadata but preserve the raw growing Fingerprint.
		var meta fileMeta
		err := s.FindCursorMeta(src, &meta)
		if err != nil {
			// No usable prior metadata (commonly no entry yet, or a partial unpack). We still fall
			// through to UpdateMetadata so the renamed path is recorded.
			meta = fileMeta{IdentifierName: p.identifier.Name()}
			log.Warnf(
				"Error while getting cursor meta data of entry '%s': '%v', using prospector's identifier: '%s'",
				src.Name(), err, meta.IdentifierName)
		}
		meta.Source = fe.NewPath
		err = s.UpdateMetadata(src, meta)
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

// isStrictPrefix reports whether prefix is a non-empty string strictly
// shorter than target, and target begins with prefix.
func isStrictPrefix(target, prefix string) bool {
	return prefix != "" && len(prefix) < len(target) && strings.HasPrefix(target, prefix)
}

// growingRawFingerprint returns the raw (hex) fingerprint to persist in the
// entry value while a file is still growing. With the bounded-key optimization
// the registry key is only a hash of this value, so prefix matching relies on
// the raw fingerprint being stored in fileMeta.Fingerprint. It returns an empty
// string for completed (final SHA-256) entries, which don't participate in
// prefix matching and must be byte-identical on disk to static fingerprint
// entries.
func growingRawFingerprint(d loginp.FileDescriptor) string {
	if d.Fingerprint.Complete() {
		return ""
	}
	return d.Fingerprint.Raw
}

// indexGrowingFingerprint adds an entry to the prefix-matching index, but only
// while the file is still growing. Completed (final SHA-256) entries match by
// their exact identity and must not participate in prefix matching, so they are
// skipped.
func (p *fileProspector) indexGrowingFingerprint(key string, d loginp.FileDescriptor, source string) {
	if raw := growingRawFingerprint(d); raw != "" {
		p.shortFingerprints.Add(key, raw, source)
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

// handleGrowingFingerprintLookup handles the lookup logic for growing-mode
// fingerprint events. Two transitions are reconciled here: continued growth
// below threshold (raw header extending) and the one-time threshold crossing
// where the identity becomes a SHA-256 (the descriptor still carries the raw
// header in Fingerprint.Raw to bridge the transition).
func (p *fileProspector) handleGrowingFingerprintLookup(
	log *logp.Logger,
	event loginp.FSEvent,
	src loginp.Source,
	updater loginp.StateMetadataUpdater) loginp.Source {
	if !event.Descriptor.Fingerprint.Complete() && event.Descriptor.Fingerprint.Raw == "" {
		return src // No fingerprint material
	}

	if updater.KeyExists(event.SrcID) {
		return src // The current fingerprint key already exists, no migration needed.
	}

	// Try to find a prefix match against the event's raw fingerprint material.
	oldKey, found := p.findGrowingFingerprintMatch(updater, event)
	if !found {
		return src
	}

	// Found a prefix match - migrate to new key
	if _, err := p.migrateGrowingFingerprint(updater, oldKey, src, event); err != nil {
		log.Errorf("failed to migrate growing fingerprint: %v", err)
		// Continue anyway - might create duplicate, but better than losing data
		return src
	}

	// Update short fingerprint set after successful migration.
	p.shortFingerprints.Remove(oldKey)
	p.indexGrowingFingerprint(event.SrcID, event.Descriptor, event.NewPath)

	// Migration succeeded - the old harvester is still running and will continue
	// reading. We should NOT start a new harvester.
	return src
}

// buildShortFingerprintSet scans the store once and populates shortFingerprints
// with the still-growing entries: those whose persisted fileMeta.Fingerprint is
// a non-empty raw-hex value (still below the configured threshold). Completed
// (final SHA-256) and legacy/static entries leave Fingerprint empty and are
// skipped, since they match by exact identity rather than prefix.
func (p *fileProspector) buildShortFingerprintSet(updater loginp.StateMetadataUpdater) {
	p.shortFingerprints = newShortFingerprintSet()

	updater.IterateOnPrefix(func(key string, meta any) {
		// Only fingerprint-identity entries hold growing state and participate
		// in prefix matching; malformed and other-identity keys are skipped.
		rk, ok := parseRegistryKey(key)
		if !ok || !rk.isFingerprint() {
			return
		}

		// Convert with typeconv: when entries are freshly written in this
		// process the cursorMeta is a fileMeta value, but when loaded from
		// the persistent registry on startup it is a map[string]interface{}
		// produced by JSON decoding into interface{}. typeconv.Convert
		// handles both cases.
		var fm fileMeta
		if err := typeconv.Convert(&fm, meta); err != nil {
			p.logger.Debugf("buildShortFingerprintSet: skipping %s: cannot convert meta to fileMeta: %v",
				key, err)
			return
		}
		// The registry key holds only a fixed-size hash of the fingerprint
		// (bounded-key optimization), so the raw fingerprint needed for prefix
		// matching is read from the persisted value, not parsed from the key.
		// A non-empty Fingerprint is exactly the set of still-growing entries;
		// final SHA-256 and legacy/static entries leave it empty and are not
		// eligible for prefix matching.
		if fm.Fingerprint == "" {
			return
		}
		p.shortFingerprints.Add(key, fm.Fingerprint, fm.Source)
	})
}

// isLikelyRename reports whether a path-agnostic prefix match looks like a
// rename rather than a collision between two distinct files: the stored
// entry's source path no longer exists on disk, so the growing file most
// likely moved to the path in the current event. If the old path still
// exists it is probably a different file that merely shares a content prefix.
func isLikelyRename(entry shortFingerprintEntry) bool {
	// Only a confirmed non-existence counts as a rename. A transient stat error
	// (permissions, I/O, unmounted FS, too many open files, ...) must NOT be
	// read as "the old path is gone", otherwise a still-present distinct file
	// sharing a header prefix could be mistaken for a rename and hijack another
	// entry's cursor (the #51417 conflation class). Be conservative on error.
	_, err := os.Stat(entry.Source)
	return errors.Is(err, os.ErrNotExist)
}

// findGrowingFingerprintMatch looks for an existing growing-phase registry
// entry whose raw-hex fingerprint is a prefix of the event's raw fingerprint
// material — i.e. the same file seen earlier with fewer bytes — and returns
// its key so the caller can migrate it to the current identity. Only still
// growing entries are searched, making this O(K) in the number of growing
// entries.
//
// A single candidate, the descriptor's raw fingerprint material
// (Fingerprint.Raw), covers both transitions: below threshold it is the
// extending raw header, and on the scan a file crosses the threshold it still
// carries the full raw header (with the SHA-256 in Fingerprint.Sum), so the
// stored raw-hex remains a prefix of it.
//
// The match is first attempted against the entry sharing the event's path
// (high-confidence). Only for a completed (threshold-crossing) event — strong
// evidence of an identity transition — a path-agnostic fallback then recovers
// a restart+rename where the stored entry still holds the OLD path. The
// fallback is gated by isLikelyRename so two distinct files that merely share
// a content prefix are not confused.
func (p *fileProspector) findGrowingFingerprintMatch(
	updater loginp.StateMetadataUpdater,
	event loginp.FSEvent,
) (oldKey string, found bool) {
	raw := event.Descriptor.Fingerprint.Raw
	if raw == "" {
		return "", false
	}

	if p.shortFingerprints == nil {
		p.buildShortFingerprintSet(updater)
	}

	// shortFingerprints still holds the pre-rename source path, so on rename
	// we match against OldPath; for in-place growth the entry already carries
	// the current path.
	matchPath := event.NewPath
	if event.Op == loginp.OpRename {
		matchPath = event.OldPath
	}

	// Require the matched entry to share the event's path, so two distinct
	// files with a shared header prefix are not mistaken for renames of one
	// another.
	if key, _, ok := p.shortFingerprints.FindPrefixMatch(raw, matchPath); ok {
		return key, true
	}

	// A completed fingerprint is strong evidence of an identity transition, so
	// beyond the same-path match we also allow a path-agnostic fallback to
	// recover a restart+rename where the stored entry still holds the OLD path.
	// isLikelyRename filters the candidates so a file still present on disk (a
	// real collision, not a rename) is not picked over a genuinely renamed one.
	if event.Descriptor.Fingerprint.Complete() {
		if key, _, ok := p.shortFingerprints.FindPrefixMatchFunc(raw, isLikelyRename); ok {
			return key, true
		}
	}

	return "", false
}

// migrateGrowingFingerprint migrates a registry entry from an old key to a new key.
// This is called when a file's fingerprint has grown.
// Returns the new key on success.
func (p *fileProspector) migrateGrowingFingerprint(
	updater loginp.StateMetadataUpdater,
	oldKey string,
	newSrc loginp.Source,
	event loginp.FSEvent,
) (string, error) {
	// Carry the raw growing fingerprint into the migrated meta.
	// On growth below threshold it is the (longer) raw-hex, keeping the entry
	// marked as growing. On a threshold-crossing migration the descriptor is
	// final SHA-256, so growingRawFingerprint returns "" and the field is
	// omitted on disk — making the migrated entry byte-identical to a static
	// fingerprint entry.
	newMeta := fileMeta{
		Source:         event.NewPath,
		IdentifierName: fingerprintName,
		Fingerprint:    growingRawFingerprint(event.Descriptor),
	}

	// Keep the old key's plugin/input prefix and swap in the new identity.
	// keyForIdentity is robust to input IDs that merely contain "fingerprint"
	// as a substring (e.g. "my-fingerprint-input").
	rk, ok := parseRegistryKey(oldKey)
	if !ok {
		return "", fmt.Errorf("invalid old key format: %s", oldKey)
	}
	newKey := rk.keyForIdentity(newSrc.Name())

	err := updater.UpdateKey(oldKey, newKey, newMeta)
	if err != nil {
		return "", fmt.Errorf("failed to migrate growing fingerprint from %s to %s: %w", oldKey, newKey, err)
	}

	return newKey, nil
}
