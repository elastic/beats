package beater

import (
	"github.com/elastic/beats/v7/libbeat/logp"
	"os"
	"os/user"
	"strconv"
	"syscall"
)

// FileSystemFetcher implement the Fetcher interface
// The FileSystemFetcher meant to fetch file/directories from the file system and ship it
// to the Kubebeat
type FileSystemFetcher struct {
	inputFilePatterns []string // Files and directories paths for the fetcher to extract info from
}

const (
	FileSystemInputType = "file-system"
)

// FileSystemResourceData represents a struct for a system resource data
// This struct is being used by the fileSystemFetcher when
type FileSystemResourceData struct {
	FileName  string `json:"filename"`
	FileMode  string `json:"mode"`
	Gid       string `json:"gid"`
	Uid       string `json:"uid"`
	InputType string `json:"type"`
	Path      string `json:"path"`
}

func NewFileFetcher(filesPaths []string) Fetcher {
	return &FileSystemFetcher{
		inputFilePatterns: filesPaths,
	}
}

func (f *FileSystemFetcher) Fetch() ([]interface{}, error) {
	results := make([]interface{}, 0)

	// Input files might contain glob pattern
	for _, filePattern := range f.inputFilePatterns {
		matchedFiles, err := Glob(filePattern)
		if err != nil {
			logp.Err("Failed to find matched glob for %s, error - %+v", filePattern, err)
		}
		for _, file := range matchedFiles {
			resource := f.fetchSystemResource(file)
			results = append(results, resource)
		}
	}
	return results, nil
}

func (f *FileSystemFetcher) fetchSystemResource(filePath string) interface{} {

	info, err := os.Stat(filePath)
	if err != nil {
		logp.Err("Failed to fetch %s, error - %+v", filePath, err)
		return nil
	}
	file := FromFileInfo(info, filePath)

	return file
}

func FromFileInfo(info os.FileInfo, path string) FileSystemResourceData {

	if info == nil {
		return FileSystemResourceData{}
	}

	stat := info.Sys().(*syscall.Stat_t)
	uid := stat.Uid
	gid := stat.Gid
	u := strconv.FormatUint(uint64(uid), 10)
	g := strconv.FormatUint(uint64(gid), 10)
	usr, _ := user.LookupId(u)
	group, _ := user.LookupGroupId(g)
	mod := strconv.FormatUint(uint64(info.Mode().Perm()), 8)

	data := FileSystemResourceData{
		FileName:  info.Name(),
		FileMode:  mod,
		Uid:       usr.Name,
		Gid:       group.Name,
		Path:      path,
		InputType: FileSystemInputType,
	}

	return data
}

func (f *FileSystemFetcher) Stop() {
}
