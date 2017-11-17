// +build !windows

package file

import (
	"errors"
	"os"
	"syscall"
)

func stat(name string, statFunc func(name string) (os.FileInfo, error)) (FileInfo, error) {
	info, err := statFunc(name)
	if err != nil {
		return nil, err
	}

	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return nil, errors.New("failed to get uid/gid")
	}

	uid := int(stat.Uid)
	gid := int(stat.Gid)
	return fileInfo{FileInfo: info, uid: &uid, gid: &gid}, nil
}
