package input

import "os"

// FileState is used to communicate the reading state of a file
type FileState struct {
	Source      string      `json:"source"`
	Offset      int64       `json:"offset"`
	Finished    bool        `json:"-"` // harvester state
	Fileinfo    os.FileInfo `json:"-"` // the file info
	FileStateOS FileStateOS
}

// NewFileState creates a new file state
func NewFileState(fileInfo os.FileInfo, path string) FileState {
	return FileState{
		Fileinfo:    fileInfo,
		Source:      path,
		Finished:    false,
		FileStateOS: GetOSFileState(fileInfo),
	}
}
