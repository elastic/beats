package crawler

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	cfg "github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/logp"
)

type ProspectorLog struct {
	Prospector *Prospector

	iteration      uint32
	lastscan       time.Time
	missingFiles   map[string]os.FileInfo
	harvesterStats map[string]harvester.FileStat
	config         cfg.ProspectorConfig
}

func NewProspectorLog(p *Prospector) (*ProspectorLog, error) {

	prospectorer := &ProspectorLog{
		Prospector: p,
		config:     p.ProspectorConfig,
		lastscan:   time.Now(),
	}

	// Init File Stat list
	prospectorer.harvesterStats = make(map[string]harvester.FileStat)

	return prospectorer, nil
}

func (p *ProspectorLog) Init() {
	logp.Debug("prospector", "exclude_files: %s", p.config.ExcludeFiles)
	p.scan()
}

func (p *ProspectorLog) Run() {

	logp.Debug("prospector", "Start next scan")
	p.scan()

	// Clear out files that disappeared and we've stopped harvesting
	for file, lastinfo := range p.harvesterStats {
		if lastinfo.Finished() && lastinfo.LastIteration < p.iteration {
			delete(p.harvesterStats, file)
		}
	}

	p.iteration++ // Overflow is allowed

	// Defer next scan for the defined scanFrequency
	time.Sleep(p.config.ScanFrequencyDuration)

}

// Scan starts a scanGlob for each provided path/glob
func (p *ProspectorLog) scan() {

	newlastscan := time.Now()

	// Now let's do one quick scan to pick up new files
	for _, path := range p.config.Paths {
		p.scanGlob(path)
	}
	p.lastscan = newlastscan
}

// Scans the specific path which can be a glob (/**/**/*.log)
// For all found files it is checked if a harvester should be started
func (p *ProspectorLog) scanGlob(glob string) {

	logp.Debug("prospector", "scan path %s", glob)

	// Evaluate the path as a wildcards/shell glob
	matches, err := filepath.Glob(glob)
	if err != nil {
		logp.Debug("prospector", "glob(%s) failed: %v", glob, err)
		return
	}

	p.missingFiles = map[string]os.FileInfo{}

	// Check any matched files to see if we need to start a harvester
	for _, file := range matches {
		logp.Debug("prospector", "Check file for harvesting: %s", file)

		// check if the file is in the exclude_files list
		if p.isFileExcluded(file) {
			logp.Debug("prospector", "Exclude file: %s", file)
			continue
		}

		// Stat the file, following any symlinks.
		fileinfo, err := os.Stat(file)
		if err != nil {
			logp.Debug("prospector", "stat(%s) failed: %s", file, err)
			continue
		}

		newFile := input.NewFile(fileinfo)

		if newFile.FileInfo.IsDir() {
			logp.Debug("prospector", "Skipping directory: %s", file)
			continue
		}

		// Check the current info against p.prospectorinfo[file]
		lastinfo, isKnown := p.harvesterStats[file]

		oldFile := input.NewFile(lastinfo.Fileinfo)

		// Create a new prospector info with the stat info for comparison
		newInfo := harvester.NewFileStat(newFile.FileInfo, p.iteration)

		// Init harvester with info
		h, err := p.Prospector.AddHarvester(file, newInfo)

		if err != nil {
			logp.Err("Error initializing harvester: %v", err)
			continue
		}

		// Conditions for starting a new harvester:
		// - file path hasn't been seen before
		// - the file's inode or device changed
		if !isKnown {
			p.checkNewFile(h)
		} else {
			h.Stat.Continue(&lastinfo)
			p.checkExistingFile(h, &newFile, &oldFile)
		}

		// Track the stat data for this file for later comparison to check for
		// rotation/etc
		p.harvesterStats[h.Path] = *h.Stat
	}
}

// Check if harvester for new file has to be started
// For a new file the following options exist:
func (p *ProspectorLog) checkNewFile(h *harvester.Harvester) {

	logp.Debug("prospector", "Start harvesting unknown file: %s", h.Path)

	// Call crawler if there if there exists a state for the given file
	offset, resuming := p.Prospector.registrar.fetchState(h.Path, h.Stat.Fileinfo)

	if p.checkOldFile(h) {

		logp.Debug("prospector", "Fetching old state of file to resume: %s", h.Path)

		// Are we resuming a dead file? We have to resume even if dead so we catch any old updates to the file
		// This is safe as the harvester, once it hits the EOF and a timeout, will stop harvesting
		// Once we detect changes again we can resume another harvester again - this keeps number of go routines to a minimum
		if resuming {
			logp.Debug("prospector", "Resuming harvester on a previously harvested file: %s", h.Path)
			p.resumeHarvesting(h, offset)
		} else {
			// Old file, skip it, but push offset of file size so we start from the end if this file changes and needs picking up
			logp.Debug("prospector", "Skipping file (older than ignore older of %v, %v): %s",
				p.config.IgnoreOlderDuration,
				time.Since(h.Stat.Fileinfo.ModTime()),
				h.Path)
			h.Stat.Skip(h.Stat.Fileinfo.Size())
		}
	} else if previousFile, err := p.getPreviousFile(h.Path, h.Stat.Fileinfo); err == nil {
		p.continueExistingFile(h, previousFile)
	} else {
		p.resumeHarvesting(h, offset)
	}
}

// checkOldFile returns true if the given file is currently not harvested
// and the last time was modified before ignore_older
func (p *ProspectorLog) checkOldFile(h *harvester.Harvester) bool {

	// Resuming never needed if ignore_older disabled
	if p.config.IgnoreOlderDuration == 0 {
		return false
	}

	modTime := h.Stat.Fileinfo.ModTime()

	// Make sure modification time is before the last scan started to not pick it up twice
	if !modTime.Before(p.lastscan) {
		return false
	}

	// Only should be checked if older then ignore_older
	if time.Since(modTime) <= p.config.IgnoreOlderDuration {
		return false
	}

	return true
}

// checkExistingFile checks if a harvester has to be started for a already known file
// For existing files the following options exist:
// * Last reading position is 0, no harvester has to be started as old harvester probably still busy
// * The old known modification time is older then the current one. Start at last known position
// * The new file is not the same as the old file, means file was renamed
// ** New file is actually really a new file, start a new harvester
// ** Renamed file has a state, continue there
func (p *ProspectorLog) checkExistingFile(h *harvester.Harvester, newFile *input.File, oldFile *input.File) {

	logp.Debug("prospector", "Update existing file for harvesting: %s", h.Path)

	// We assume it is the same file, but it wasn't
	if !oldFile.IsSameFile(newFile) {

		logp.Debug("prospector", "File previously found: %s", h.Path)

		if previousFile, err := p.getPreviousFile(h.Path, h.Stat.Fileinfo); err == nil {
			p.continueExistingFile(h, previousFile)
		} else {
			// File is not the same file we saw previously, it must have rotated and is a new file
			logp.Debug("prospector", "Launching harvester on rotated file: %s", h.Path)

			// Forget about the previous harvester and let it continue on the old file - so start a new channel to use with the new harvester
			h.Stat.Ignore()

			// Start a new harvester on the path
			h.Start()
			p.Prospector.registrar.Persist <- h.GetState()

		}

		// Keep the old file in missingFiles so we don't rescan it if it was renamed and we've not yet reached the new filename
		// We only need to keep it for the remainder of this iteration then we can assume it was deleted and forget about it
		p.missingFiles[h.Path] = oldFile.FileInfo

	} else if h.Stat.Finished() && oldFile.FileInfo.ModTime() != h.Stat.Fileinfo.ModTime() {
		// Resume harvesting of an old file we've stopped harvesting from
		// Start a harvester on the path; a file was just modified and it doesn't have a harvester
		// The offset to continue from will be stored in the harvester channel - so take that to use and also clear the channel
		p.resumeHarvesting(h, <-h.Stat.Return)
		p.Prospector.registrar.Persist <- h.GetState()

	} else {
		logp.Debug("prospector", "Not harvesting, file didn't change: %s", h.Path)
	}
}

// Continue reading on an existing file.
// The given file was renamed from another file we know -> The same harvester channel is linked as the old file
// The file param is only used for logging
func (p *ProspectorLog) continueExistingFile(h *harvester.Harvester, previousFile string) {
	logp.Debug("prospector", "Launching harvester on renamed file. File rename was detected: %s -> %s", previousFile, h.Path)

	lastinfo := p.harvesterStats[previousFile]
	h.Stat.Continue(&lastinfo)

	// Update state because of file rotation
	p.Prospector.registrar.Persist <- h.GetState()
}

// Start / resume harvester with a predefined offset
func (p *ProspectorLog) resumeHarvesting(h *harvester.Harvester, offset int64) {

	logp.Debug("prospector", "Start / resuming harvester of file: %s", h.Path)
	h.SetOffset(offset)
	h.Start()

	// Update state because of file rotation
	p.Prospector.registrar.Persist <- h.GetState()
}

// Check if the given file was renamed. If file is known but with different path,
// the previous file path will be returned. If no file is found, an error
// will be returned.
func (p *ProspectorLog) getPreviousFile(file string, info os.FileInfo) (string, error) {

	for path, pFileStat := range p.harvesterStats {
		if path == file {
			continue
		}

		if os.SameFile(info, pFileStat.Fileinfo) {
			return path, nil
		}
	}

	// Now check the missingfiles
	for path, fileInfo := range p.missingFiles {

		if os.SameFile(info, fileInfo) {
			return path, nil
		}
	}

	return "", fmt.Errorf("No previous file found")
}

func (p *ProspectorLog) isFileExcluded(file string) bool {

	if len(p.config.ExcludeFilesRegexp) > 0 {

		if harvester.MatchAnyRegexps(p.config.ExcludeFilesRegexp, file) {
			return true
		}
	}

	return false
}
