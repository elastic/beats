package beater

import (
	"github.com/elastic/beats/v7/libbeat/logp"
	"os"
	"os/user"
	"strconv"
	"syscall"
)

type FileSystemFetcher struct {
	filesPaths []string // Files and directories paths for the fetcher to extract info from
}

const (
	FILE_SYSTEM_INPUT_TYPE = "file-system"
)

type FileSystemResourceData struct {
	FileName  string `json:"fileName"`
	FileMode  string `json:"fileMode"`
	Gid       string `json:"gid"`
	Uid       string `json:"uid"`
	InputType string `json:"inputType"`
	Path      string `json:"path"`
}

func NewFileFetcher(filesPaths []string) Fetcher {
	return &FileSystemFetcher{
		filesPaths: filesPaths,
	}
}

func (f *FileSystemFetcher) Fetch() (interface{}, error) {
	results := make([]FileSystemResourceData, 0)

	for _, filePath := range f.filesPaths {
		info, err := os.Stat(filePath)

		// If errors occur during file system resource, just skip on the file and log the error
		if err != nil {
			logp.Err("Failed to fetch %s, error - %+v", filePath, err)
			continue
		}

		result := FromFileInfo(info, filePath)
		results = append(results, result)
	}

	return results, nil
}

func (f *FileSystemFetcher) Stop() {
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

	data :=
		FileSystemResourceData{
			FileName: info.Name(),
			FileMode: mod,
			Uid:      usr.Name,
			Gid:      group.Name,
			Path:     path,
			InputType: FILE_SYSTEM_INPUT_TYPE,
		}

	return data
}
