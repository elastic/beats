package prospector

import (
	"os"
	"path/filepath"
	"time"

	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/libbeat/logp"
)

type ProspectorLog struct {
	Prospector *Prospector
	config     prospectorConfig
	lastScan   time.Time
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
	if p.config.CleanOlder > 0 {
		p.Prospector.states.Cleanup()
		logp.Debug("prospector", "Prospector states cleaned up.")
	}
	p.lastScan = time.Now()
}

// getFiles returns all files which have to be harvested
// All globs are expanded and then directory and excluded files are removed
func (p *ProspectorLog) getFiles() map[string]os.FileInfo {
	// Now let's do one quick scan to pick up new files

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

	newLastScan := time.Now()

	// TODO: Track harvesters to prevent any file from being harvested twice. Finished state could be delayed?
	// Now let's do one quick scan to pick up new files
	for f, fileinfo := range p.getFiles() {

		logp.Debug("prospector", "Check file for harvesting: %s", f)

		// Create new state for comparison
		newState := file.NewState(fileinfo, f)

		// Load last state
		index, lastState := p.Prospector.states.FindPrevious(newState)

		// Decides if previous state exists
		if index == -1 {
			p.harvestNewFile(newState)
		} else {
			p.harvestExistingFile(newState, lastState)
		}
	}

	p.lastScan = newLastScan
}

// harvestNewFile harvest a new file
func (p *ProspectorLog) harvestNewFile(state file.State) {

	if !p.isIgnoreOlder(state) {
		logp.Debug("prospector", "Start harvester for new file: %s", state.Source)
		p.Prospector.startHarvester(state, 0)
	} else {
		logp.Debug("prospector", "Ignore file because ignore_older reached: %s", state.Source)
	}
}

// harvestExistingFile continues harvesting a file with a known state if needed
func (p *ProspectorLog) harvestExistingFile(newState file.State, oldState file.State) {

	logp.Debug("prospector", "Update existing file for harvesting: %s, offset: %v", newState.Source, oldState.Offset)

	// TODO: check for ignore_older reached? or should that happen in scan already?

	// No harvester is running for the file, start a new harvester
	// It is important here that only the size is checked and not modification time, as modification time could be incorrect on windows
	// https://blogs.technet.microsoft.com/asiasupp/2010/12/14/file-date-modified-property-are-not-updating-while-modifying-a-file-without-closing-it/
	if oldState.Finished && newState.Fileinfo.Size() > oldState.Offset {
		// Resume harvesting of an old file we've stopped harvesting from
		// This could also be an issue with force_close_older that a new harvester is started after each scan but not needed?
		// One problem with comparing modTime is that it is in seconds, and scans can happen more then once a second
		logp.Debug("prospector", "Resuming harvesting of file: %s, offset: %v", newState.Source, oldState.Offset)
		p.Prospector.startHarvester(newState, oldState.Offset)

	} else if oldState.Source != "" && oldState.Source != newState.Source {
		// This does not start a new harvester as it is assume that the older harvester is still running
		// or no new lines were detected. It sends only an event status update to make sure the new name is persisted.
		logp.Debug("prospector", "File rename was detected, updating state: %s -> %s, Current offset: %v", oldState.Source, newState.Source, oldState.Offset)

		h, _ := p.Prospector.createHarvester(newState)
		h.SetOffset(oldState.Offset)

		// Update state because of file rotation
		h.SendStateUpdate()
	} else {
		// TODO: improve logging depedent on what the exact reason is that harvesting does not continue
		// Nothing to do. Harvester is still running and file was not renamed
		logp.Debug("prospector", "No updates needed, file %s is already harvested.", newState.Source)
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
