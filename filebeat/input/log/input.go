package log

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/filebeat/util"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/atomic"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
)

const (
	recursiveGlobDepth = 8
)

var (
	filesRenamed     = monitoring.NewInt(nil, "filebeat.input.log.files.renamed")
	filesTruncated   = monitoring.NewInt(nil, "filebeat.input.log.files.truncated")
	harvesterSkipped = monitoring.NewInt(nil, "filebeat.harvester.skipped")
)

func init() {
	err := input.Register("log", NewInput)
	if err != nil {
		panic(err)
	}
}

// Input contains the input and its config
type Input struct {
	cfg           *common.Config
	config        config
	states        *file.States
	harvesters    *harvester.Registry
	outlet        channel.Outleter
	stateOutlet   channel.Outleter
	done          chan struct{}
	numHarvesters atomic.Uint32
}

// NewInput instantiates a new Log
func NewInput(
	cfg *common.Config,
	outlet channel.Factory,
	context input.Context,
) (input.Input, error) {

	// Note: underlying output.
	//  The input and harvester do have different requirements
	//  on the timings the outlets must be closed/unblocked.
	//  The outlet generated here is the underlying outlet, only closed
	//  once all workers have been shut down.
	//  For state updates and events, separate sub-outlets will be used.
	out, err := outlet(cfg, context.DynamicFields)
	if err != nil {
		return nil, err
	}

	// stateOut will only be unblocked if the beat is shut down.
	// otherwise it can block on a full publisher pipeline, so state updates
	// can be forwarded correctly to the registrar.
	stateOut := channel.CloseOnSignal(channel.SubOutlet(out), context.BeatDone)

	p := &Input{
		config:      defaultConfig,
		cfg:         cfg,
		harvesters:  harvester.NewRegistry(),
		outlet:      out,
		stateOutlet: stateOut,
		states:      file.NewStates(),
		done:        context.Done,
	}

	if err := cfg.Unpack(&p.config); err != nil {
		return nil, err
	}
	if err := p.config.resolveRecursiveGlobs(); err != nil {
		return nil, fmt.Errorf("Failed to resolve recursive globs in config: %v", err)
	}
	if err := p.config.normalizeGlobPatterns(); err != nil {
		return nil, fmt.Errorf("Failed to normalize globs patterns: %v", err)
	}

	// Create empty harvester to check if configs are fine
	// TODO: Do config validation instead
	_, err = p.createHarvester(file.State{}, nil)
	if err != nil {
		return nil, err
	}

	if len(p.config.Paths) == 0 {
		return nil, fmt.Errorf("each input must have at least one path defined")
	}

	err = p.loadStates(context.States)
	if err != nil {
		return nil, err
	}

	logp.Info("Configured paths: %v", p.config.Paths)

	return p, nil
}

// LoadStates loads states into input
// It goes through all states coming from the registry. Only the states which match the glob patterns of
// the input will be loaded and updated. All other states will not be touched.
func (p *Input) loadStates(states []file.State) error {
	logp.Debug("input", "exclude_files: %s. Number of stats: %d", p.config.ExcludeFiles, len(states))

	for _, state := range states {
		// Check if state source belongs to this input. If yes, update the state.
		if p.matchesFile(state.Source) {
			state.TTL = -1

			// In case a input is tried to be started with an unfinished state matching the glob pattern
			if !state.Finished {
				return fmt.Errorf("Can only start an input when all related states are finished: %+v", state)
			}

			// Update input states and send new states to registry
			err := p.updateState(state)
			if err != nil {
				logp.Err("Problem putting initial state: %+v", err)
				return err
			}
		}
	}

	logp.Debug("input", "input with previous states loaded: %v", p.states.Count())
	return nil
}

// Run runs the input
func (p *Input) Run() {
	logp.Debug("input", "Start next scan")

	// TailFiles is like ignore_older = 1ns and only on startup
	if p.config.TailFiles {
		ignoreOlder := p.config.IgnoreOlder

		// Overwrite ignore_older for the first scan
		p.config.IgnoreOlder = 1
		defer func() {
			// Reset ignore_older after first run
			p.config.IgnoreOlder = ignoreOlder
			// Disable tail_files after the first run
			p.config.TailFiles = false
		}()
	}
	p.scan()

	// It is important that a first scan is run before cleanup to make sure all new states are read first
	if p.config.CleanInactive > 0 || p.config.CleanRemoved {
		beforeCount := p.states.Count()
		cleanedStates, pendingClean := p.states.Cleanup()
		logp.Debug("input", "input states cleaned up. Before: %d, After: %d, Pending: %d",
			beforeCount, beforeCount-cleanedStates, pendingClean)
	}

	// Marking removed files to be cleaned up. Cleanup happens after next scan to make sure all states are updated first
	if p.config.CleanRemoved {
		for _, state := range p.states.GetStates() {
			// os.Stat will return an error in case the file does not exist
			stat, err := os.Stat(state.Source)
			if err != nil {
				if os.IsNotExist(err) {
					p.removeState(state)
					logp.Debug("input", "Remove state for file as file removed: %s", state.Source)
				} else {
					logp.Err("input state for %s was not removed: %s", state.Source, err)
				}
			} else {
				// Check if existing source on disk and state are the same. Remove if not the case.
				newState := file.NewState(stat, state.Source, p.config.Type)
				if !newState.FileStateOS.IsSame(state.FileStateOS) {
					p.removeState(state)
					logp.Debug("input", "Remove state for file as file removed or renamed: %s", state.Source)
				}
			}
		}
	}
}

func (p *Input) removeState(state file.State) {
	// Only clean up files where state is Finished
	if !state.Finished {
		logp.Debug("input", "State for file not removed because harvester not finished: %s", state.Source)
		return
	}

	state.TTL = 0
	err := p.updateState(state)
	if err != nil {
		logp.Err("File cleanup state update error: %s", err)
	}
}

// getFiles returns all files which have to be harvested
// All globs are expanded and then directory and excluded files are removed
func (p *Input) getFiles() map[string]os.FileInfo {
	paths := map[string]os.FileInfo{}

	for _, path := range p.config.Paths {
		matches, err := filepath.Glob(path)
		if err != nil {
			logp.Err("glob(%s) failed: %v", path, err)
			continue
		}

	OUTER:
		// Check any matched files to see if we need to start a harvester
		for _, file := range matches {

			// check if the file is in the exclude_files list
			if p.isFileExcluded(file) {
				logp.Debug("input", "Exclude file: %s", file)
				continue
			}

			// Fetch Lstat File info to detected also symlinks
			fileInfo, err := os.Lstat(file)
			if err != nil {
				logp.Debug("input", "lstat(%s) failed: %s", file, err)
				continue
			}

			if fileInfo.IsDir() {
				logp.Debug("input", "Skipping directory: %s", file)
				continue
			}

			isSymlink := fileInfo.Mode()&os.ModeSymlink > 0
			if isSymlink && !p.config.Symlinks {
				logp.Debug("input", "File %s skipped as it is a symlink.", file)
				continue
			}

			// Fetch Stat file info which fetches the inode. In case of a symlink, the original inode is fetched
			fileInfo, err = os.Stat(file)
			if err != nil {
				logp.Debug("input", "stat(%s) failed: %s", file, err)
				continue
			}

			// If symlink is enabled, it is checked that original is not part of same input
			// It original is harvested by other input, states will potentially overwrite each other
			if p.config.Symlinks {
				for _, finfo := range paths {
					if os.SameFile(finfo, fileInfo) {
						logp.Info("Same file found as symlink and originap. Skipping file: %s", file)
						continue OUTER
					}
				}
			}

			paths[file] = fileInfo
		}
	}

	return paths
}

// matchesFile returns true in case the given filePath is part of this input, means matches its glob patterns
func (p *Input) matchesFile(filePath string) bool {
	// Path is cleaned to ensure we always compare clean paths
	filePath = filepath.Clean(filePath)

	for _, glob := range p.config.Paths {

		// Glob is cleaned to ensure we always compare clean paths
		glob = filepath.Clean(glob)

		// Evaluate if glob matches filePath
		match, err := filepath.Match(glob, filePath)
		if err != nil {
			logp.Debug("input", "Error matching glob: %s", err)
			continue
		}

		// Check if file is not excluded
		if match && !p.isFileExcluded(filePath) {
			return true
		}
	}
	return false
}

type FileSortInfo struct {
	info os.FileInfo
	path string
}

func getSortInfos(paths map[string]os.FileInfo) []FileSortInfo {
	sortInfos := make([]FileSortInfo, 0, len(paths))
	for path, info := range paths {
		sortInfo := FileSortInfo{info: info, path: path}
		sortInfos = append(sortInfos, sortInfo)
	}

	return sortInfos
}

func getSortedFiles(scanOrder string, scanSort string, sortInfos []FileSortInfo) ([]FileSortInfo, error) {
	var sortFunc func(i, j int) bool
	switch scanSort {
	case "modtime":
		switch scanOrder {
		case "asc":
			sortFunc = func(i, j int) bool {
				return sortInfos[i].info.ModTime().Before(sortInfos[j].info.ModTime())
			}
		case "desc":
			sortFunc = func(i, j int) bool {
				return sortInfos[i].info.ModTime().After(sortInfos[j].info.ModTime())
			}
		default:
			return nil, fmt.Errorf("Unexpected value for scan.order: %v", scanOrder)
		}
	case "filename":
		switch scanOrder {
		case "asc":
			sortFunc = func(i, j int) bool {
				return strings.Compare(sortInfos[i].info.Name(), sortInfos[j].info.Name()) < 0
			}
		case "desc":
			sortFunc = func(i, j int) bool {
				return strings.Compare(sortInfos[i].info.Name(), sortInfos[j].info.Name()) > 0
			}
		default:
			return nil, fmt.Errorf("Unexpected value for scan.order: %v", scanOrder)
		}
	default:
		return nil, fmt.Errorf("Unexpected value for scan.sort: %v", scanSort)
	}

	if sortFunc != nil {
		sort.Slice(sortInfos, sortFunc)
	}

	return sortInfos, nil
}

func getFileState(path string, info os.FileInfo, p *Input) (file.State, error) {
	var err error
	var absolutePath string
	absolutePath, err = filepath.Abs(path)
	if err != nil {
		return file.State{}, fmt.Errorf("could not fetch abs path for file %s: %s", absolutePath, err)
	}
	logp.Debug("input", "Check file for harvesting: %s", absolutePath)
	// Create new state for comparison
	newState := file.NewState(info, absolutePath, p.config.Type)
	return newState, nil
}

func getKeys(paths map[string]os.FileInfo) []string {
	files := make([]string, 0)
	for file := range paths {
		files = append(files, file)
	}
	return files
}

// Scan starts a scanGlob for each provided path/glob
func (p *Input) scan() {
	var sortInfos []FileSortInfo
	var files []string

	paths := p.getFiles()

	var err error

	if p.config.ScanSort != "" {
		sortInfos, err = getSortedFiles(p.config.ScanOrder, p.config.ScanSort, getSortInfos(paths))
		if err != nil {
			logp.Err("Failed to sort files during scan due to error %s", err)
		}
	}

	if sortInfos == nil {
		files = getKeys(paths)
	}

	for i := 0; i < len(paths); i++ {

		var path string
		var info os.FileInfo

		if sortInfos == nil {
			path = files[i]
			info = paths[path]
		} else {
			path = sortInfos[i].path
			info = sortInfos[i].info
		}

		select {
		case <-p.done:
			logp.Info("Scan aborted because input stopped.")
			return
		default:
		}

		newState, err := getFileState(path, info, p)
		if err != nil {
			logp.Err("Skipping file %s due to error %s", path, err)
		}

		// Load last state
		lastState := p.states.FindPrevious(newState)

		// Ignores all files which fall under ignore_older
		if p.isIgnoreOlder(newState) {
			err := p.handleIgnoreOlder(lastState, newState)
			if err != nil {
				logp.Err("Updating ignore_older state error: %s", err)
			}
			continue
		}

		// Decides if previous state exists
		if lastState.IsEmpty() {
			logp.Debug("input", "Start harvester for new file: %s", newState.Source)
			err := p.startHarvester(newState, 0)
			if err != nil {
				logp.Err("Harvester could not be started on new file: %s, Err: %s", newState.Source, err)
			}
		} else {
			p.harvestExistingFile(newState, lastState)
		}
	}
}

// harvestExistingFile continues harvesting a file with a known state if needed
func (p *Input) harvestExistingFile(newState file.State, oldState file.State) {
	logp.Debug("input", "Update existing file for harvesting: %s, offset: %v", newState.Source, oldState.Offset)

	// No harvester is running for the file, start a new harvester
	// It is important here that only the size is checked and not modification time, as modification time could be incorrect on windows
	// https://blogs.technet.microsoft.com/asiasupp/2010/12/14/file-date-modified-property-are-not-updating-while-modifying-a-file-without-closing-it/
	if oldState.Finished && newState.Fileinfo.Size() > oldState.Offset {
		// Resume harvesting of an old file we've stopped harvesting from
		// This could also be an issue with force_close_older that a new harvester is started after each scan but not needed?
		// One problem with comparing modTime is that it is in seconds, and scans can happen more then once a second
		logp.Debug("input", "Resuming harvesting of file: %s, offset: %d, new size: %d", newState.Source, oldState.Offset, newState.Fileinfo.Size())
		err := p.startHarvester(newState, oldState.Offset)
		if err != nil {
			logp.Err("Harvester could not be started on existing file: %s, Err: %s", newState.Source, err)
		}
		return
	}

	// File size was reduced -> truncated file
	if oldState.Finished && newState.Fileinfo.Size() < oldState.Offset {
		logp.Debug("input", "Old file was truncated. Starting from the beginning: %s, offset: %d, new size: %d ", newState.Source, newState.Fileinfo.Size())
		err := p.startHarvester(newState, 0)
		if err != nil {
			logp.Err("Harvester could not be started on truncated file: %s, Err: %s", newState.Source, err)
		}

		filesTruncated.Add(1)
		return
	}

	// Check if file was renamed
	if oldState.Source != "" && oldState.Source != newState.Source {
		// This does not start a new harvester as it is assume that the older harvester is still running
		// or no new lines were detected. It sends only an event status update to make sure the new name is persisted.
		logp.Debug("input", "File rename was detected: %s -> %s, Current offset: %v", oldState.Source, newState.Source, oldState.Offset)

		if oldState.Finished {
			logp.Debug("input", "Updating state for renamed file: %s -> %s, Current offset: %v", oldState.Source, newState.Source, oldState.Offset)
			// Update state because of file rotation
			oldState.Source = newState.Source
			err := p.updateState(oldState)
			if err != nil {
				logp.Err("File rotation state update error: %s", err)
			}

			filesRenamed.Add(1)
		} else {
			logp.Debug("input", "File rename detected but harvester not finished yet.")
		}
	}

	if !oldState.Finished {
		// Nothing to do. Harvester is still running and file was not renamed
		logp.Debug("input", "Harvester for file is still running: %s", newState.Source)
	} else {
		logp.Debug("input", "File didn't change: %s", newState.Source)
	}
}

// handleIgnoreOlder handles states which fall under ignore older
// Based on the state information it is decided if the state information has to be updated or not
func (p *Input) handleIgnoreOlder(lastState, newState file.State) error {
	logp.Debug("input", "Ignore file because ignore_older reached: %s", newState.Source)

	if !lastState.IsEmpty() {
		if !lastState.Finished {
			logp.Info("File is falling under ignore_older before harvesting is finished. Adjust your close_* settings: %s", newState.Source)
		}
		// Old state exist, no need to update it
		return nil
	}

	// Make sure file is not falling under clean_inactive yet
	if p.isCleanInactive(newState) {
		logp.Debug("input", "Do not write state for ignore_older because clean_inactive reached")
		return nil
	}

	// Set offset to end of file to be consistent with files which were harvested before
	// See https://github.com/elastic/beats/pull/2907
	newState.Offset = newState.Fileinfo.Size()

	// Write state for ignore_older file as none exists yet
	newState.Finished = true
	err := p.updateState(newState)
	if err != nil {
		return err
	}

	return nil
}

// isFileExcluded checks if the given path should be excluded
func (p *Input) isFileExcluded(file string) bool {
	patterns := p.config.ExcludeFiles
	return len(patterns) > 0 && harvester.MatchAny(patterns, file)
}

// isIgnoreOlder checks if the given state reached ignore_older
func (p *Input) isIgnoreOlder(state file.State) bool {
	// ignore_older is disable
	if p.config.IgnoreOlder == 0 {
		return false
	}

	modTime := state.Fileinfo.ModTime()
	if time.Since(modTime) > p.config.IgnoreOlder {
		return true
	}

	return false
}

// isCleanInactive checks if the given state false under clean_inactive
func (p *Input) isCleanInactive(state file.State) bool {
	// clean_inactive is disable
	if p.config.CleanInactive <= 0 {
		return false
	}

	modTime := state.Fileinfo.ModTime()
	if time.Since(modTime) > p.config.CleanInactive {
		return true
	}

	return false
}

// subOutletWrap returns a factory method that will wrap the passed outlet
// in a SubOutlet and memoize the result so the wrapping is done only once.
func subOutletWrap(outlet channel.Outleter) func() channel.Outleter {
	var subOutlet channel.Outleter
	return func() channel.Outleter {
		if subOutlet == nil {
			subOutlet = channel.SubOutlet(outlet)
		}
		return subOutlet
	}
}

// createHarvester creates a new harvester instance from the given state
func (p *Input) createHarvester(state file.State, onTerminate func()) (*Harvester, error) {
	// Each wraps the outlet, for closing the outlet individually
	h, err := NewHarvester(
		p.cfg,
		state,
		p.states,
		func(d *util.Data) bool {
			return p.stateOutlet.OnEvent(d)
		},
		subOutletWrap(p.outlet),
	)
	if err == nil {
		h.onTerminate = onTerminate
	}
	return h, err
}

// startHarvester starts a new harvester with the given offset
// In case the HarvesterLimit is reached, an error is returned
func (p *Input) startHarvester(state file.State, offset int64) error {
	if p.numHarvesters.Inc() > p.config.HarvesterLimit && p.config.HarvesterLimit > 0 {
		p.numHarvesters.Dec()
		harvesterSkipped.Add(1)
		return fmt.Errorf("Harvester limit reached")
	}
	// Set state to "not" finished to indicate that a harvester is running
	state.Finished = false
	state.Offset = offset

	// Create harvester with state
	h, err := p.createHarvester(state, func() { p.numHarvesters.Dec() })
	if err != nil {
		p.numHarvesters.Dec()
		return err
	}

	err = h.Setup()
	if err != nil {
		p.numHarvesters.Dec()
		return fmt.Errorf("Error setting up harvester: %s", err)
	}

	// Update state before staring harvester
	// This makes sure the states is set to Finished: false
	// This is synchronous state update as part of the scan
	h.SendStateUpdate()

	if err = p.harvesters.Start(h); err != nil {
		p.numHarvesters.Dec()
	}
	return err
}

// updateState updates the input state and forwards the event to the spooler
// All state updates done by the input itself are synchronous to make sure not states are overwritten
func (p *Input) updateState(state file.State) error {
	// Add ttl if cleanOlder is enabled and TTL is not already 0
	if p.config.CleanInactive > 0 && state.TTL != 0 {
		state.TTL = p.config.CleanInactive
	}

	// Update first internal state
	p.states.Update(state)

	data := util.NewData()
	data.SetState(state)
	ok := p.outlet.OnEvent(data)
	if !ok {
		logp.Info("input outlet closed")
		return errors.New("input outlet closed")
	}

	return nil
}

// Wait waits for the all harvesters to complete and only then call stop
func (p *Input) Wait() {
	p.harvesters.WaitForCompletion()
	p.Stop()
}

// Stop stops all harvesters and then stops the input
func (p *Input) Stop() {
	// Stop all harvesters
	// In case the beatDone channel is closed, this will not wait for completion
	// Otherwise Stop will wait until output is complete
	p.harvesters.Stop()

	// close state updater
	p.stateOutlet.Close()

	// stop all communication between harvesters and publisher pipeline
	p.outlet.Close()
}
