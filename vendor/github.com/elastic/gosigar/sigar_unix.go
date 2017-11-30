// Copyright (c) 2012 VMware, Inc.

// +build darwin freebsd linux

package gosigar

import "syscall"

func (self *FileSystemUsage) Get(path string) error {
	stat := syscall.Statfs_t{}
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return err
	}

	self.Total = uint64(stat.Blocks) * uint64(stat.Bsize)
	self.Free = uint64(stat.Bfree) * uint64(stat.Bsize)
	self.Avail = uint64(stat.Bavail) * uint64(stat.Bsize)
	self.Used = self.Total - self.Free
	self.Files = stat.Files
	self.FreeFiles = uint64(stat.Ffree)

	return nil
}
