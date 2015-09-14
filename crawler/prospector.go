package crawler

import (
	"os"
	"path/filepath"
	"time"

	cfg "github.com/elastic/filebeat/config"
	. "github.com/elastic/filebeat/input"
	"github.com/elastic/libbeat/logp"
)

// Last reading state of the prospector
type ProspectorResume struct {
	Files   map[string]*FileState
	Persist chan *FileState
}

type Prospector struct {
	FileConfig     cfg.FileConfig
	prospectorinfo map[string]ProspectorInfo
	iteration      uint32
	lastscan       time.Time
}

type ProspectorInfo struct {
	Fileinfo  os.FileInfo /* the file info */
	Harvester chan int64  /* the harvester will send an event with its offset when it closes */
	Last_seen uint32      /* int number of the last iterations in which we saw this file */
}

func (restart *ProspectorResume) Scan(files []cfg.FileConfig, persist map[string]*FileState, eventChan chan *FileEvent) {
	pendingProspectorCnt := 0

	// Prospect the globs/paths given on the command line and launch harvesters
	for _, fileconfig := range files {

		logp.Debug("prospector", "File Config:", fileconfig)

		prospector := &Prospector{FileConfig: fileconfig}
		go prospector.Prospect(restart, eventChan)
		pendingProspectorCnt++
	}

	// Now determine which states we need to persist by pulling the events from the prospectors
	// When we hit a nil source a prospector had finished so we decrease the expected events
	logp.Debug("prospector", "Waiting for %d prospectors to initialise", pendingProspectorCnt)

	for event := range restart.Persist {
		if event.Source == nil {
			pendingProspectorCnt--
			if pendingProspectorCnt == 0 {
				break
			}
			continue
		}
		persist[*event.Source] = event
		logp.Debug("prospector", "Registrar will re-save state for %s", *event.Source)
	}

	logp.Info("All prospectors initialised with %d states to persist", len(persist))

}

func (p *Prospector) Prospect(resume *ProspectorResume, output chan *FileEvent) {
	p.prospectorinfo = make(map[string]ProspectorInfo)

	// Handle any "-" (stdin) paths
	for i, path := range p.FileConfig.Paths {

		logp.Debug("prospector", "Harvest path: %s", path)

		if path == "-" {
			// Offset and Initial never get used when path is "-"
			harvester := Harvester{Path: path, FileConfig: p.FileConfig}
			go harvester.Harvest(output)

			// Remove it from the file list
			p.FileConfig.Paths = append(p.FileConfig.Paths[:i], p.FileConfig.Paths[i+1:]...)
		}
	}

	// Seed last scan time
	p.lastscan = time.Now()

	// In case dead time is not set, set it to 24h
	if p.FileConfig.DeadTime == "" {
		// Default dead time
		p.FileConfig.DeadTime = "24h"
	}

	var err error

	p.FileConfig.DeadtimeSpan, err = time.ParseDuration(p.FileConfig.DeadTime)

	if err != nil {
		logp.Warn("Failed to parse dead time duration '%s'. Error was: %s\n", p.FileConfig.DeadTime, err)
	}


	// Now let's do one quick scan to pick up new files
	for _, path := range p.FileConfig.Paths {
		p.scan(path, output, resume)
	}

	// This signals we finished considering the previous state
	event := &FileState{
		Source: nil,
	}
	resume.Persist <- event

	for {
		newlastscan := time.Now()

		for _, path := range p.FileConfig.Paths {
			// Scan - flag false so new files always start at beginning
			p.scan(path, output, nil)
		}

		p.lastscan = newlastscan

		// Defer next scan for a bit.
		time.Sleep(10 * time.Second) // Make this tunable

		// Clear out files that disappeared and we've stopped harvesting
		for file, lastinfo := range p.prospectorinfo {
			if len(lastinfo.Harvester) != 0 && lastinfo.Last_seen < p.iteration {
				delete(p.prospectorinfo, file)
			}
		}

		p.iteration++ // Overflow is allowed
	}
}

func (p *Prospector) scan(path string, output chan *FileEvent, resume *ProspectorResume) {

	logp.Debug("prospector", "scan path %s", path)
	// Evaluate the path as a wildcards/shell glob
	matches, err := filepath.Glob(path)
	if err != nil {
		logp.Debug("prospector", "glob(%s) failed: %v", path, err)
		return
	}

	// To keep the old inode/dev reference if we see a file has renamed, in case it was also renamed prior
	missinginfo := make(map[string]os.FileInfo)

	// Check any matched files to see if we need to start a harvester
	for _, file := range matches {
		logp.Debug("prospector", "Check file for harvesting: %s", file)

		// Stat the file, following any symlinks.
		fileinfo, err := os.Stat(file)

		// TODO(sissel): check err
		if err != nil {
			logp.Debug("prospector", "stat(%s) failed: %s", file, err)
			continue
		}

		newFile := File{
			FileInfo: fileinfo,
		}

		if newFile.FileInfo.IsDir() {
			logp.Debug("prospector", "Skipping directory: %s", file)
			continue
		}

		// Check the current info against p.prospectorinfo[file]
		lastinfo, is_known := p.prospectorinfo[file]

		oldFile := File{
			FileInfo: lastinfo.Fileinfo,
		}

		newinfo := lastinfo

		// Conditions for starting a new harvester:
		// - file path hasn't been seen before
		// - mathe file's inode or device changed
		if !is_known {
			logp.Debug("prospector", "Start harvesting unkown file:", file)
			// Create a new prospector info with the stat info for comparison
			newinfo = ProspectorInfo{Fileinfo: newFile.FileInfo, Harvester: make(chan int64, 1), Last_seen: p.iteration}

			// Check for dead time, but only if the file modification time is before the last scan started
			// This ensures we don't skip genuine creations with dead times less than 10s
			if newFile.FileInfo.ModTime().Before(p.lastscan) && time.Since(newFile.FileInfo.ModTime()) > p.FileConfig.DeadtimeSpan {
				var offset int64 = 0
				var is_resuming bool = false

				if resume != nil {
					// Call the calculator - it will process resume state if there is one
					offset, is_resuming = p.calculateResume(file, newFile.FileInfo, resume)
				}

				// Are we resuming a dead file? We have to resume even if dead so we catch any old updates to the file
				// This is safe as the harvester, once it hits the EOF and a timeout, will stop harvesting
				// Once we detect changes again we can resume another harvester again - this keeps number of go routines to a minimum
				if is_resuming {
					logp.Debug("prospector", "Resuming harvester on a previously harvested file: %s", file)
					harvester := &Harvester{Path: file, FileConfig: p.FileConfig, Offset: offset, FinishChan: newinfo.Harvester}
					go harvester.Harvest(output)
				} else {
					// Old file, skip it, but push offset of file size so we start from the end if this file changes and needs picking up
					logp.Debug("prospector", "Skipping file (older than dead time of %v): %s", p.FileConfig.DeadtimeSpan, file)
					newinfo.Harvester <- newFile.FileInfo.Size()
				}
			} else if previous := p.isFileRenamed(file, newFile.FileInfo, missinginfo); previous != "" {
				// This file was simply renamed (known inode+dev) - link the same harvester channel as the old file
				logp.Debug("prospector", "File rename was detected: %s -> %s", previous, file)

				newinfo.Harvester = p.prospectorinfo[previous].Harvester
			} else {
				var offset int64 = 0
				var is_resuming bool = false

				if resume != nil {
					// Call the calculator - it will process resume state if there is one
					offset, is_resuming = p.calculateResume(file, newFile.FileInfo, resume)
				}

				// Are we resuming a file or is this a completely new file?
				if is_resuming {
					logp.Debug("prospector", "Resuming harvester on a previously harvested file: %s", file)
				} else {
					logp.Debug("prospector", "Launching harvester on new file: %s", file)
				}

				// Launch the harvester
				harvester := &Harvester{Path: file, FileConfig: p.FileConfig, Offset: offset, FinishChan: newinfo.Harvester}
				go harvester.Harvest(output)
			}
		} else {

			logp.Debug("prospector", "Update existing file for harvesting:", file)
			// Update the fileinfo information used for future comparisons, and the last_seen counter
			newinfo.Fileinfo = newFile.FileInfo
			newinfo.Last_seen = p.iteration

			if !oldFile.IsSameFile(&newFile) {
				if previous := p.isFileRenamed(file, newFile.FileInfo, missinginfo); previous != "" {
					// This file was renamed from another file we know - link the same harvester channel as the old file
					logp.Debug("prospector", "File rename was detected: %s -> %s", previous, file)
					logp.Debug("prospector", "Launching harvester on renamed file: %s", file)

					newinfo.Harvester = p.prospectorinfo[previous].Harvester
				} else {
					// File is not the same file we saw previously, it must have rotated and is a new file
					logp.Debug("prospector", "Launching harvester on rotated file: %s", file)

					// Forget about the previous harvester and let it continue on the old file - so start a new channel to use with the new harvester
					newinfo.Harvester = make(chan int64, 1)

					// Start a harvester on the path
					harvester := &Harvester{Path: file, FileConfig: p.FileConfig, FinishChan: newinfo.Harvester}
					go harvester.Harvest(output)
				}

				// Keep the old file in missinginfo so we don't rescan it if it was renamed and we've not yet reached the new filename
				// We only need to keep it for the remainder of this iteration then we can assume it was deleted and forget about it
				missinginfo[file] = oldFile.FileInfo
			} else if len(newinfo.Harvester) != 0 && oldFile.FileInfo.ModTime() != newFile.FileInfo.ModTime() {
				// Resume harvesting of an old file we've stopped harvesting from
				logp.Debug("prospector", "Resuming harvester on an old file that was just modified: %s", file)

				// Start a harvester on the path; an old file was just modified and it doesn't have a harvester
				// The offset to continue from will be stored in the harvester channel - so take that to use and also clear the channel
				harvester := &Harvester{Path: file, FileConfig: p.FileConfig, Offset: <-newinfo.Harvester, FinishChan: newinfo.Harvester}
				go harvester.Harvest(output)
			} else {
				logp.Debug("prospector", "Not harvesting, harvester probably still busy: ", file)
			}
		}

		// Track the stat data for this file for later comparison to check for
		// rotation/etc
		p.prospectorinfo[file] = newinfo
	} // for each file matched by the glob
}

func (p *Prospector) calculateResume(file string, fileinfo os.FileInfo, resume *ProspectorResume) (int64, bool) {
	last_state, is_found := resume.Files[file]

	if is_found && IsSameFile(file, fileinfo, last_state) {
		// We're resuming - throw the last state back downstream so we resave it
		// And return the offset - also force harvest in case the file is old and we're about to skip it
		resume.Persist <- last_state
		return last_state.Offset, true
	}

	if previous := p.isFileRenamedResumelist(file, fileinfo, resume.Files); previous != "" {
		// File has rotated between shutdown and startup
		// We return last state downstream, with a modified event source with the new file name
		// And return the offset - also force harvest in case the file is old and we're about to skip it
		logp.Debug("prospector", "Detected rename of a previously harvested file: %s -> %s", previous, file)
		last_state := resume.Files[previous]
		last_state.Source = &file
		resume.Persist <- last_state
		return last_state.Offset, true
	}

	if is_found {
		logp.Debug("prospector", "Not resuming rotated file: %s", file)
	}

	// New file so just start from an automatic position
	return 0, false
}
