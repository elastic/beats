package input

import (
	"os"
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
)

type File struct {
	File      *os.File
	FileInfo  os.FileInfo
	Path      string
	FileState *FileState
}

// FileEvent is sent to the output and must contain all relevant information
type FileEvent struct {
	ReadTime time.Time
	Source   *string
	Offset   int64
	Line     uint64
	Text     *string
	Fields   *map[string]string
	Fileinfo *os.FileInfo
}

type FileState struct {
	Source      *string `json:"source,omitempty"`
	Offset      int64   `json:"offset,omitempty"`
	FileStateOS *FileStateOS
}

// Builds and returns the FileState object based on the Event info.
func (f *FileEvent) GetState() *FileState {

	state := &FileState{
		Source: f.Source,
		// take the offset + length of the line + newline char and
		// save it as the new starting offset.
		// This issues a problem, if the EOL is a CRLF! Then on start it read the LF again and generates a event with an empty line
		Offset:      f.Offset + int64(len(*f.Text)) + 1, // REVU: this is begging for BUGs
		FileStateOS: GetOSFileState(f.Fileinfo),
	}

	return state
}

func (f *FileEvent) ToMapStr() common.MapStr {
	event := common.MapStr{
		"timestamp": common.Time(f.ReadTime),
		"source":    f.Source,
		"offset":    f.Offset,
		"line":      f.Line,
		"message":   f.Text,
		"fileinfo":  f.Fileinfo,
		"type":      "log",
	}

	if f.Fields != nil {
		event["fields"] = f.Fields
	}

	return event
}

// Check that the file isn't a symlink, mode is regular or file is nil
func (f *File) IsRegularFile() bool {
	if f.File == nil {
		logp.Critical("Harvester: BUG: f arg is nil")
		return false
	}

	info, e := f.File.Stat()
	if e != nil {
		logp.Err("File check fault: stat error: %s", e.Error())
		return false
	}

	if !info.Mode().IsRegular() {
		logp.Warn("Harvester: not a regular file: %q %s", info.Mode(), info.Name())
		return false
	}
	return true
}

// IsSameFile checks if the given File path corresponds with the FileInfo given
func IsSameFile(path string, info os.FileInfo) bool {
	fileInfo, err := os.Stat(path)

	if err != nil {
		logp.Err("Error during file comparison: %s with %s - Error: %s", path, info.Name(), err)
		return false
	}

	return os.SameFile(fileInfo, info)
}

// Checks if the two files are the same.
func (f1 *File) IsSameFile(f2 *File) bool {
	return os.SameFile(f1.FileInfo, f2.FileInfo)
}
