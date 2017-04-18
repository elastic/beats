package cfgfile

import (
	"os"
	"path/filepath"
	"time"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/mitchellh/hashstructure"
)

type GlobWatcher struct {
	glob     string
	lastScan time.Time
	lastHash uint64
}

func NewGlobWatcher(glob string) *GlobWatcher {
	return &GlobWatcher{
		lastScan: time.Time{},
		lastHash: 0,
		glob:     glob,
	}
}

// Scan scans all files matched by the glob and checks if the number of files or the modtime of the files changed
// It returns the list of files, a boolean if anything in the glob changed and potential errors.
// To detect changes not only mod time is compared but also the hash of the files list. This is required to
// also detect files which were removed.
// The modtime is compared based on second as normally mod-time is in seconds. If it is unclear if something changed
// the method will return true for the changes. It is strongly recommend to call scan not more frequent then 1s.
func (gw *GlobWatcher) Scan() ([]string, bool, error) {

	globList, err := filepath.Glob(gw.glob)
	if err != nil {
		return nil, false, err
	}

	updatedFiles := false
	files := []string{}

	lastScan := time.Now()
	defer func() { gw.lastScan = lastScan }()

	for _, f := range globList {

		info, err := os.Stat(f)
		if err != nil {
			logp.Err("Error getting stats for file: %s", f)
			continue
		}

		// Directories are skipped
		if !info.Mode().IsRegular() {
			continue
		}

		// Check if one of the files was changed recently
		// File modification time can be in seconds. -1 + truncation is to cover for files which
		// were created during this second.
		// If the last scan was at 09:02:15.00001 it will pick up files which were modified also 09:02:14
		// As this scan no necessarily picked up files form 09:02:14
		// TODO: How could this be improved / simplified? Behaviour was sometimes flaky. Is ModTime updated with delay?
		if info.ModTime().After(gw.lastScan.Add(-1 * time.Second).Truncate(time.Second)) {
			updatedFiles = true
		}

		files = append(files, f)
	}

	hash, err := hashstructure.Hash(files, nil)
	if err != nil {
		return files, true, err
	}
	defer func() { gw.lastHash = hash }()

	// Check if something changed
	if !updatedFiles && hash == gw.lastHash {
		return files, false, nil
	}

	return files, true, nil
}
