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
	split := strings.Split(v.Key, "::")
	if len(split) != 4 {
		// This should never happen.
		p.logger.Errorf("registry key '%s' is in the wrong format, cannot migrate state", v.Key)
		return "", fm
	}

	idFromRegistry := strings.Join(split[2:], "::")
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

		registryKey := v.Key()
		split := strings.Split(registryKey, identitySep)
		// Wrong key format
		if len(split) != 4 {
			return "", fm
		}

		registryFileIdentity := split[2] + identitySep + split[3]
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
			// We preserve ONLY when there's a current file whose
			// GrowingFingerprint has this entry's stored fingerprint as a
			// STRICT prefix. GrowingFingerprint is emitted by the scanner
			// on the one-time scan a path crosses threshold, so a hit
			// here means a file has bridged from a (potentially renamed)
			// raw-hex identity to a final SHA-256 identity — strong
			// evidence of a rename + threshold crossing.
			//
			// Ordinary growing-phase rename across restart is intentionally
			// NOT covered by this skip: distinguishing it from a
			// shared-header collision (two distinct files starting with
			// the same content, one of them deleted) is ambiguous without
			// the threshold signal, and preserving in that case would
			// cause incorrect state reuse for the surviving file.
			if !fm.FingerprintGrowing {
				return true
			}
			key := v.Key()
			delim := identitySep + fingerprintName + identitySep
			idx := strings.LastIndex(key, delim)
			if idx < 0 {
				return true
			}
			storedFP := key[idx+len(delim):]
			for _, desc := range files {
				if isStrictPrefix(desc.GrowingFingerprint, storedFP) {
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
		p.logger.Debugf("trying to migrate file identity to %s", identifierName)
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

	// Bootstrap the scanner's hashedPaths set from the persisted registry.
	// The seed contains paths whose registry state has FingerprintGrowing=false
	// (or absent — legacy entries read as zero-value). Those files are already
	// keyed by a SHA-256 hex; their next scan should emit only SHA-256 and not
	// the (transient, large) GrowingFingerprint. Files in the growing phase
	// (FingerprintGrowing=true) are NOT in the seed and so will emit
	// GrowingFingerprint on the watch loop's first scan, enabling the
	// prospector's prefix-match-and-migrate path. Also subsumes the wipe of
	// any pollution caused by the enumeration-only GetFiles calls in
	// Init/takeover above (they touch the same hashedPaths set).
	//
	// Best-effort: a seed failure (filewatcher doesn't implement
	// hashedPathsSetter) is logged and the input continues. Not initializing
	// HashedPaths will cause the file watcher to emmit a growing fingerprint
	// for files that are already using the SHA-256 version of the fingerprint.
	// The error causes a performanhce hit at startup, later the HashedPaths is
	// populated as the input runs.
	if err := p.initFileWatcherHashedPaths(s); err != nil {
		p.logger.Errorf("failed to seed fileWatcher hashedPaths: %v", err)
	}

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
				Source:             event.NewPath,
				IdentifierName:     p.identifier.Name(),
				FingerprintGrowing: event.Descriptor.FingerprintGrowing,
			})
			if err != nil {
				log.Errorf("Failed to set cursor meta data of entry %s: %v", src.Name(), err)
			}

			// Only growing entries participate in prefix-matching. Final
			// SHA-256 entries (Growing=false) don't need to be in the index.
			if event.Descriptor.FingerprintGrowing {
				p.shortFingerprints.Add(event.SrcID, event.Descriptor.Fingerprint, event.NewPath)
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
		// migrate the registry key BEFORE onRename. We use event.OldPath for
		// the prefix lookup because shortFingerprintEntries still has the old
		// source path at this point.
		// Migration must happen first so onRename's UpdateMetadata finds the
		// entry under the new key (which uses the new fingerprint from src).
		if p.growingFingerprint {
			oldKey, found := p.findGrowingFingerprintMatch(
				updater,
				event.Descriptor.Fingerprint,
				event.Descriptor.GrowingFingerprint,
				event.OldPath)
			if found {
				newKey, err := p.migrateGrowingFingerprint(updater, oldKey, src, event)
				if err != nil {
					log.Errorf("failed to migrate growing fingerprint on rename: %v", err)
				} else {
					p.shortFingerprints.Remove(oldKey)
					if event.Descriptor.FingerprintGrowing {
						p.shortFingerprints.Add(newKey, event.Descriptor.Fingerprint, event.NewPath)
					}
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

// isStrictPrefix reports whether prefix is a non-empty string strictly
// shorter than target, and target begins with prefix.
func isStrictPrefix(target, prefix string) bool {
	return prefix != "" && len(prefix) < len(target) && strings.HasPrefix(target, prefix)
}

// initFileWatcherHashedPaths reads the registry and populates the fileWatcher's
// scanner-side "already-final" path set so that on the next scan we only emit
// GrowingFingerprint for files that actually still need to migrate.
//
// Selection: a fingerprint entry whose persisted state has
// FingerprintGrowing=false (the omitzero default, also the value legacy
// entries from registries written before the field existed read as) is a
// final SHA-256 entry; its source path goes into the seed. Entries with
// FingerprintGrowing=true are deliberately excluded — they need
// GrowingFingerprint emitted on the next scan to bridge to their SHA-256.
//
// Returns nil and is a no-op when growing mode is disabled. Returns an error
// when growing mode is enabled but p.filewatcher does not expose
// SetHashedPaths — the only capability needed here.
func (p *fileProspector) initFileWatcherHashedPaths(updater loginp.StateMetadataUpdater) error {
	if !p.growingFingerprint {
		return nil
	}

	fw, ok := p.filewatcher.(interface {
		SetHashedPaths(paths map[string]struct{})
	})
	if !ok {
		return fmt.Errorf(
			"filewatcher of type %T does not implement SetHashedPaths(map[string]struct{}); "+
				"hashedPaths cannot be seeded",
			p.filewatcher)
	}

	fullyGrownPaths := make(map[string]struct{})
	const fingerprintKeyPrefix = identitySep + fingerprintName + identitySep
	updater.IterateOnPrefix(func(key string, meta interface{}) bool {
		if !strings.Contains(key, fingerprintKeyPrefix) {
			return true
		}
		var fm fileMeta
		if err := typeconv.Convert(&fm, meta); err != nil {
			return true
		}
		if fm.FingerprintGrowing {
			return true // still growing; needs GrowingFingerprint on next scan
		}
		if fm.Source == "" {
			return true
		}
		fullyGrownPaths[fm.Source] = struct{}{}
		return true
	})

	fw.SetHashedPaths(fullyGrownPaths)
	p.logger.Debugf("seeded fileWatcher hashedPaths with %d already-final paths", len(fullyGrownPaths))
	return nil
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
// below threshold (raw-hex extending) and the one-time threshold crossing
// where the primary Fingerprint flips from raw-hex to SHA-256 (with the
// descriptor carrying GrowingFingerprint as the bridging value).
func (p *fileProspector) handleGrowingFingerprintLookup(
	log *logp.Logger,
	event loginp.FSEvent,
	src loginp.Source,
	updater loginp.StateMetadataUpdater) loginp.Source {
	// Empty fingerprint - nothing to match
	if event.Descriptor.Fingerprint == "" {
		return src
	}

	// Fast path: if the current fingerprint key already exists, no migration
	// needed.
	if updater.KeyExists(event.SrcID) {
		return src
	}

	// Try to find a prefix match across both Fingerprint and GrowingFingerprint.
	oldKey, found := p.findGrowingFingerprintMatch(
		updater, event.Descriptor.Fingerprint, event.Descriptor.GrowingFingerprint, event.NewPath)
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
	// Only re-add the new entry to the growing index if the migrated value
	// is itself still in the growing phase. After a threshold-crossing
	// migration the new key holds a final SHA-256 (Growing=false) and must
	// not participate in prefix matching.
	if event.Descriptor.FingerprintGrowing {
		p.shortFingerprints.Add(event.SrcID, event.Descriptor.Fingerprint, event.NewPath)
	}

	// Migration succeeded - the old harvester is still running and will continue
	// reading. We should NOT start a new harvester.
	return src
}

// buildShortFingerprintSet scans the store once and populates shortFingerprints
// with entries whose persisted state has Growing == true (their Fingerprint is
// a raw-hex value still below the configured threshold).
// Entries whose state has Growing == false (default, including legacy entries
// written before this field existed) are skipped: they are final SHA-256 keys
// and don't participate in prefix matching.
func (p *fileProspector) buildShortFingerprintSet(updater loginp.StateMetadataUpdater) {
	p.shortFingerprints = newShortFingerprintSet()

	updater.IterateOnPrefix(func(key string, meta interface{}) bool {
		// key format: filestream::INPUT_ID::fingerprint::FINGERPRINT
		// Find '::' separator positions manually to avoid wrong match
		// if the input ID contains "fingerprint::".
		var seps [4]int
		nSeps := 0
		for i := 0; i < len(key)-1; i++ {
			if key[i] == ':' && key[i+1] == ':' {
				seps[nSeps] = i
				nSeps++
				if nSeps == 4 {
					break
				}
				i++
			}
		}
		if nSeps != 3 {
			return true // malformed key
		}

		identityName := key[seps[1]+2 : seps[2]]
		if identityName != fingerprintName {
			return true // not a fingerprint entry
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
			return true
		}
		if !fm.FingerprintGrowing {
			return true // final SHA-256 entry; not eligible for prefix matching
		}

		fingerprint := key[seps[2]+2:]
		p.shortFingerprints.Add(key, fingerprint, fm.Source)
		return true
	})
}

// findGrowingFingerprintMatch looks for an existing registry entry whose
// raw-hex fingerprint is a prefix of the current scan's raw-hex fingerprint.
// Two prefix candidates are tried, in order:
//
//  1. currentFingerprint — handles same-format growth (raw-hex extending
//     while the file is still below threshold).
//  2. currentGrowingFingerprint — handles the one-time threshold-crossing
//     scan, where the primary Fingerprint has flipped to SHA-256 but the
//     descriptor also carries the raw-hex GrowingFingerprint for matching.
//
// For each candidate the lookup is tried first with same-source-path
// matching (high-confidence: stored entry's Source path == currentPath) and
// then with path-agnostic matching (handles restart + rename, where the
// stored entry's Source is the OLD path but the descriptor only has the
// NEW path). The path-agnostic fallback returns the longest prefix match
// across all growing entries, which makes accidental collisions extremely
// unlikely for SHA-256-length raw-hex fingerprints (~2048 chars).
//
// Only entries in shortFingerprintSet (those whose state has
// FingerprintGrowing == true) are searched, making this O(K) where K is
// the number of still-growing entries.
func (p *fileProspector) findGrowingFingerprintMatch(
	updater loginp.StateMetadataUpdater,
	currentFingerprint string,
	currentGrowingFingerprint string,
	currentPath string,
) (oldKey string, found bool) {
	if currentFingerprint == "" && currentGrowingFingerprint == "" {
		return "", false
	}

	// Build short fingerprint set on first use
	if p.shortFingerprints == nil {
		p.buildShortFingerprintSet(updater)
	}

	tryMatch := func(target string, allowPathAgnostic bool) (string, bool) {
		if target == "" {
			return "", false
		}
		// High-confidence: same source path.
		if key, _, ok := p.shortFingerprints.FindPrefixMatch(target, currentPath); ok {
			return key, true
		}
		if !allowPathAgnostic {
			return "", false
		}
		// Path-agnostic fallback: only enabled for the threshold-crossing
		// case (caller passes the descriptor's GrowingFingerprint here).
		// Restricted to this case because GrowingFingerprint is set only
		// on the one-time scan a file's primary Fingerprint flips from
		// raw-hex to SHA-256 — i.e. it carries strong evidence of an
		// identity transition. Ordinary same-format growth (raw-hex
		// extending below threshold) does NOT get the fallback, so two
		// distinct files with a shared header content prefix are not
		// confused for renames of one another.
		//
		// Additional guard: refuse if the stored entry's Source path still
		// exists on disk — a likely collision (the old file is still
		// there), not a rename.
		if key, entry, ok := p.shortFingerprints.FindPrefixMatch(target, ""); ok {
			if _, err := os.Stat(entry.Source); err != nil {
				return key, true
			}
		}
		return "", false
	}

	// Ordinary same-format growth: no path-agnostic fallback.
	if key, ok := tryMatch(currentFingerprint, false); ok {
		return key, true
	}
	// Threshold-crossing: path-agnostic fallback enabled.
	if key, ok := tryMatch(currentGrowingFingerprint, true); ok {
		return key, true
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
	// Carry the current descriptor's FingerprintGrowing flag into the migrated
	// meta.
	// On fingerprint growth bellow threshold the fingerprint is the raw-hex and
	// FingerprintGrowing=true.
	// On a threshold-crossing migration the new value is the final SHA-256
	// (FingerprintGrowing=false). FingerprintGrowing=false is not serialized,
	// making the migrated entry indistinguishable from a static fingerprint
	// entry on disk.
	newMeta := fileMeta{
		Source:             event.NewPath,
		IdentifierName:     fingerprintName,
		FingerprintGrowing: event.Descriptor.FingerprintGrowing,
	}

	// Find the boundary between input ID and identity name. We can't
	// use strings.Split on fingerprintName because the input ID may
	// also contain "fingerprint" as a substring (e.g. an input id of
	// "my-fingerprint-input"). The full delimiter "::fingerprint::"
	// is unambiguous: it can only mark the start of the identity
	// segment, since "::" cannot legally appear inside an input ID.
	delim := identitySep + fingerprintName + identitySep
	idx := strings.LastIndex(oldKey, delim)
	if idx < 0 {
		return "", fmt.Errorf("invalid old key format: %s", oldKey)
	}
	newKey := oldKey[:idx+len(identitySep)] + newSrc.Name()

	err := updater.UpdateKey(oldKey, newKey, newMeta)
	if err != nil {
		return "", fmt.Errorf("failed to migrate growing fingerprint from %s to %s: %w", oldKey, newKey, err)
	}

	return newKey, nil
}
