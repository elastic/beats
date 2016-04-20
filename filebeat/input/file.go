package input

import (
	"fmt"
	"os"
	"time"

	"github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

const (
	jsonErrorKey = "json_error"
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
	ReadTime     time.Time
	Source       string
	InputType    string
	DocumentType string
	Offset       int64
	Bytes        int
	Text         *string
	Fileinfo     *os.FileInfo
	JSONFields   common.MapStr
	JSONConfig   *config.JSONConfig
}

type FileState struct {
	Source      string `json:"source,omitempty"`
	Offset      int64  `json:"offset,omitempty"`
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

// mergeJSONFields writes the JSON fields in the event map,
// respecting the KeysUnderRoot and OverwriteKeys configuration options.
// If MessageKey is defined, the Text value from the event always
// takes precedence.
func mergeJSONFields(f *FileEvent, event common.MapStr) {

	// The message key might have been modified by multiline
	if len(f.JSONConfig.MessageKey) > 0 && f.Text != nil {
		f.JSONFields[f.JSONConfig.MessageKey] = *f.Text
	}

	if f.JSONConfig.KeysUnderRoot {
		for k, v := range f.JSONFields {
			if f.JSONConfig.OverwriteKeys {
				if k == "@timestamp" {
					vstr, ok := v.(string)
					if !ok {
						logp.Err("JSON: Won't overwrite @timestamp because value is not string")
						event[jsonErrorKey] = "@timestamp not overwritten (not string)"
						continue
					}
					// @timestamp must be of time common.Time
					ts, err := common.ParseTime(vstr)
					if err != nil {
						logp.Err("JSON: Won't overwrite @timestamp because of parsing error: %v", err)
						event[jsonErrorKey] = fmt.Sprintf("@timestamp not overwritten (parse error on %s)", vstr)
						continue
					}
					event[k] = ts
				} else if k == "type" {
					vstr, ok := v.(string)
					if !ok {
						logp.Err("JSON: Won't overwrite type because value is not string")
						event[jsonErrorKey] = "type not overwritten (not string)"
						continue
					}
					if len(vstr) == 0 || vstr[0] == '_' {
						logp.Err("JSON: Won't overwrite type because value is empty or starts with an underscore")
						event[jsonErrorKey] = fmt.Sprintf("type not overwritten (invalid value [%s])", vstr)
						continue
					}
					event[k] = vstr
				} else {
					event[k] = v
				}
			} else if _, exists := event[k]; !exists {
				event[k] = v
			}
		}
	} else {
		event["json"] = f.JSONFields
	}
}

func (f *FileEvent) ToMapStr() common.MapStr {
	event := common.MapStr{
		common.EventMetadataKey: f.EventMetadata,
		"@timestamp":            common.Time(f.ReadTime),
		"source":                f.Source,
		"offset":                f.Offset, // Offset here is the offset before the starting char.
		"type":                  f.DocumentType,
		"input_type":            f.InputType,
	}

	if f.JSONConfig != nil && len(f.JSONFields) > 0 {
		mergeJSONFields(f, event)
	} else {
		event["message"] = f.Text
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
