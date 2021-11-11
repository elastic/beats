package beater

import (
	"os"
	"os/user"
	"strconv"
	"syscall"
)

type Fetcher interface {
	Fetch() (interface{}, error)
	Stop()
}

type FileResult struct {
	Data FileData
	Err  error
	Path string
}

type FileData struct {
	fileName      string
	fileMode      string
	gid           string
	uid           string
	inputType     string
	path     	  string
}

func ExtractFiles(filesPaths []string) ([]FileResult, error) {

	results := make([]FileResult, 0)

	for _, filePath := range filesPaths {
		info, err := os.Stat(filePath)

		result := FileResult{
			Path: filePath,
			Err:  err,
			Data: FromFileInfo(info, filePath),
		}

		results = append(results, result)

	}

	return results, nil
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
			fileName: info.Name(),
			fileMode: mod,
			uid:      usr.Name,
			gid:      group.Name,
			path: path,
		}

	return data
}
