package beater

import (
	"github.com/elastic/beats/v7/libbeat/logp"
	"os"
	"os/user"
	"strconv"
	"syscall"
)

type Fetcher interface {
	Fetch() (interface{}, error)
	Stop()
}

type FilesFetcher struct {
	filesPaths []string // Files and directories paths for the fetcher to extract info from
}


type FileData struct {
	FileName  string `json:"fileName"`
	FileMode  string `json:"fileMode"`
	Gid       string `json:"gid"`
	Uid       string `json:"uid"`
	InputType string `json:"inputType"`
	Path      string `json:"path"`
}

func NewFileFetcher(filesPaths []string) Fetcher {
	return &FilesFetcher{
		filesPaths: filesPaths,
	}
}

func (f *FilesFetcher) Fetch() (interface{}, error) {
	results := make([]FileData, 0)

	for _, filePath := range f.filesPaths {
		info, err := os.Stat(filePath)

		// If errors occur during files read, just skip on the file
		if err != nil {
			logp.Err("Failed to fetch %s, error - %+v", filePath, err)
			continue
		}

		result := FromFileInfo(info, filePath)
		results = append(results, result)
	}

	return results, nil
}

func (f *FilesFetcher) Stop() {
}

func FromFileInfo(info os.FileInfo, path string) FileData {

	if info == nil {
		return FileData{}
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
		FileData{
			FileName: info.Name(),
			FileMode: mod,
			Uid:      usr.Name,
			Gid:      group.Name,
			Path:     path,
		}

	return data
}
