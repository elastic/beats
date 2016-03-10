package input

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type File struct {
	File      *os.File
	FileInfo  os.FileInfo
	Path      string
	FileState *FileState
}

// FileEvent is sent to the output and must contain all relevant information
type FileEvent struct {
	common.EventMetadata
	ReadTime        time.Time
	Source          *string
	InputType       string
	DocumentType    string
	Offset          int64
	Bytes           int
	Text            *string
	Fileinfo        *os.FileInfo
	fieldsUnderRoot bool
	jsonDecoder     *config.JsonDecoderConfig
}

type FileState struct {
	Source      *string `json:"source,omitempty"`
	Offset      int64   `json:"offset,omitempty"`
	FileStateOS *FileStateOS
}

// NewFile create new File object
func NewFile(fileInfo os.FileInfo) File {
	return File{
		FileInfo: fileInfo,
	}
}

// GetState builds and returns the FileState object based on the Event info.
func (f *FileEvent) GetState() *FileState {
	// Add read bytes to current offset to point to the end
	offset := f.Offset + int64(f.Bytes)

	state := &FileState{
		Source:      f.Source,
		Offset:      offset,
		FileStateOS: GetOSFileState(f.Fileinfo),
	}

	return state
}

// SetJsonDecoder configures how to do JSON parsing. If nil is passed,
// JSON decoding is disabled.
func (f *FileEvent) SetJsonDecoder(jsonDecoder *config.JsonDecoderConfig) {
	f.jsonDecoder = jsonDecoder
}

func (f *FileEvent) ToMapStr() common.MapStr {
	event := common.MapStr{
		common.EventMetadataKey: f.EventMetadata,
		"@timestamp":            common.Time(f.ReadTime),
		"source":                f.Source,
		"offset":                f.Offset, // Offset here is the offset before the starting char.
		"message":               f.Text,
		"type":                  f.DocumentType,
		"input_type":            f.InputType,
		"count":                 1,
	}

	if f.jsonDecoder != nil {
		var jsonObj common.MapStr
		err := json.Unmarshal([]byte(*f.Text), &jsonObj)
		if err != nil {
			logp.Err("Error decoding JSON: %v", err)
			if f.jsonDecoder.OnUnmarshalError == "add_error_key" {
				event["json_error"] = fmt.Sprintf("Error decoding JSON: %v", err)
			}
		} else {
			// no error, move keys into the object
			for key, value := range jsonObj {
				_, found := event[key]
				if !found || f.jsonDecoder.OverwriteKeys {
					event[key] = value
				}
			}

			if !f.jsonDecoder.KeepOriginal {
				delete(event, "message")
			}
		}
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

func IsRegularFile(file *os.File) bool {
	f := &File{File: file}
	return f.IsRegularFile()
}

// Checks if the two files are the same.
func (f1 *File) IsSameFile(f2 *File) bool {
	return os.SameFile(f1.FileInfo, f2.FileInfo)
}
