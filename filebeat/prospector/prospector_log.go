package prospector

import (
	"os"
	"path/filepath"
	"time"

	"expvar"

	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/libbeat/logp"
)

var (
	filesRenamed  = expvar.NewInt("filebeat.prospector.log.files.renamed")
	filesTrucated = expvar.NewInt("filebeat.prospector.log.files.truncated")
)

type ProspectorLog struct {
	Prospector *Prospector
	config     prospectorConfig
	lastClean  time.Time
}

func NewProspectorLog(p *Prospector) (*ProspectorLog, error) {

	prospectorer := &ProspectorLog{
		Prospector: p,
		config:     p.config,
	}

	return prospectorer, nil
}

func (p *ProspectorLog) Init() {
	logp.Debug("prospector", "exclude_files: %s", p.config.ExcludeFiles)

	logp.Info("Load previous states from registry into memory")
	fileStates := p.Prospector.states.GetStates()

	// Make sure all states are set as finished
	for key, state := range fileStates {
		state.Finished = true
		fileStates[key] = state
	}

	// Overwrite prospector states
	p.Prospector.states.SetStates(fileStates)
	p.lastClean = time.Now()

	logp.Info("Previous states loaded: %v", p.Prospector.states.Count())
}

func (p *ProspectorLog) Run() {
	logp.Debug("prospector", "Start next scan")

	p.scan()

	// It is important that a first scan is run before cleanup to make sure all new states are read first
	if p.config.CleanInactive > 0 || p.config.CleanRemoved {
		beforeCount := p.Prospector.states.Count()
		p.Prospector.states.Cleanup()
		logp.Debug("prospector", "Prospector states cleaned up. Before: %d, After: %d", beforeCount, p.Prospector.states.Count())
	}

	// Marking removed files to be cleaned up. Cleanup happens after next scan to make sure all states are updated first
	if p.config.CleanRemoved {
		for _, state := range p.Prospector.states.GetStates() {
			// os.Stat will return an error in case the file does not exist
			_, err := os.Stat(state.Source)
			if err != nil {
				// Only clean up files where state is Finished
				if state.Finished {
					state.TTL = 0
					event := input.NewEvent(state)
					p.Prospector.harvesterChan <- event
					logp.Debug("prospector", "Remove state for file as file removed: %s", state.Source)
				} else {
					logp.Debug("prospector", "State for file not removed because not finished: %s", state.Source)
				}
			}
		}
	}
}

// getFiles returns all files which have to be harvested
// All globs are expanded and then directory and excluded files are removed
func (p *ProspectorLog) getFiles() map[string]os.FileInfo {

	paths := map[string]os.FileInfo{}

	for _, glob := range p.config.Paths {
		// Evaluate the path as a wildcards/shell glob
		matches, err := filepath.Glob(glob)
		if err != nil {
			logp.Err("glob(%s) failed: %v", glob, err)
			continue
		}

		// Check any matched files to see if we need to start a harvester
		for _, file := range matches {

			// check if the file is in the exclude_files list
			if p.isFileExcluded(file) {
				logp.Debug("prospector", "Exclude file: %s", file)
				continue
			}

			fileinfo, err := os.Lstat(file)
			if err != nil {
				logp.Debug("prospector", "stat(%s) failed: %s", file, err)
				continue
			}

			// Check if file is symlink
			if fileinfo.Mode()&os.ModeSymlink != 0 {
				logp.Debug("prospector", "File %s skipped as it is a symlink.", file)
				continue
			}

			if fileinfo.IsDir() {
				logp.Debug("prospector", "Skipping directory: %s", file)
				continue
			}

			paths[file] = fileinfo
		}
	}

	return paths
}

// Scan starts a scanGlob for each provided path/glob
func (p *ProspectorLog) scan() {

	for path, info := range p.getFiles() {

		logp.Debug("prospector", "Check file for harvesting: %s", path)

		// Create new state for comparison
		newState := file.NewState(info, path)

		// Load last state
		lastState := p.Prospector.states.FindPrevious(newState)

		// Ignores all files which fall under ignore_older
		if p.isIgnoreOlder(newState) {
			logp.Debug("prospector", "Ignore file because ignore_older reached: %s", newState.Source)
			if lastState.IsEmpty() && lastState.Finished == false {
				logp.Err("File is falling under ignore_older before harvesting is finished. Adjust your close_* settings: %s", newState.Source)
			}
			continue
		}

		// Decides if previous state exists
		if lastState.IsEmpty() {
			logp.Debug("prospector", "Start harvester for new file: %s", newState.Source)
			p.Prospector.startHarvester(newState, 0)
		} else {
			p.harvestExistingFile(newState, lastState)
		}
	}
}

// harvestExistingFile continues harvesting a file with a known state if needed
func (p *ProspectorLog) harvestExistingFile(newState file.State, oldState file.State) {

	logp.Debug("prospector", "Update existing file for harvesting: %s, offset: %v", newState.Source, oldState.Offset)

	// No harvester is running for the file, start a new harvester
	// It is important here that only the size is checked and not modification time, as modification time could be incorrect on windows
	// https://blogs.technet.microsoft.com/asiasupp/2010/12/14/file-date-modified-property-are-not-updating-while-modifying-a-file-without-closing-it/
	if oldState.Finished && newState.Fileinfo.Size() > oldState.Offset {
		// Resume harvesting of an old file we've stopped harvesting from
		// This could also be an issue with force_close_older that a new harvester is started after each scan but not needed?
		// One problem with comparing modTime is that it is in seconds, and scans can happen more then once a second
		logp.Debug("prospector", "Resuming harvesting of file: %s, offset: %v", newState.Source, oldState.Offset)
		p.Prospector.startHarvester(newState, oldState.Offset)
		return
	}

	// File size was reduced -> truncated file
	if oldState.Finished && newState.Fileinfo.Size() < oldState.Offset {
		logp.Debug("prospector", "Old file was truncated. Starting from the beginning: %s", newState.Source)
		p.Prospector.startHarvester(newState, 0)

		filesTrucated.Add(1)
		return
	}

	// Check if file was renamed
	if oldState.Source != "" && oldState.Source != newState.Source {
		// This does not start a new harvester as it is assume that the older harvester is still running
		// or no new lines were detected. It sends only an event status update to make sure the new name is persisted.
		logp.Debug("prospector", "File rename was detected, updating state: %s -> %s, Current offset: %v", oldState.Source, newState.Source, oldState.Offset)

		// Update state because of file rotation
		newState.Offset = oldState.Offset
		event := input.NewEvent(newState)
		p.Prospector.harvesterChan <- event

		filesRenamed.Add(1)
	}

	if !oldState.Finished {
		// Nothing to do. Harvester is still running and file was not renamed
		logp.Debug("prospector", "Harvester for file is still running: %s", newState.Source)
	} else {
		logp.Debug("prospector", "File didn't change: %s", newState.Source)
	}
}

// isFileExcluded checks if the given path should be excluded
func (p *ProspectorLog) isFileExcluded(file string) bool {
	patterns := p.config.ExcludeFiles
	return len(patterns) > 0 && harvester.MatchAnyRegexps(patterns, file)
}

// isIgnoreOlder checks if the given state reached ignore_older
func (p *ProspectorLog) isIgnoreOlder(state file.State) bool {

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
