//go:build aix || darwin || freebsd || linux
// +build aix darwin freebsd linux

package filesystem

import (
	"fmt"
	"syscall"

	"github.com/elastic/beats/v7/libbeat/opt"
)

// GetUsage returns the filesystem usage
func (fs *FSStat) GetUsage() error {
	stat := syscall.Statfs_t{}
	err := syscall.Statfs(fs.Directory, &stat)
	if err != nil {
		return fmt.Errorf("error in Statfs syscall: %w", err)
	}

	fs.Total = opt.UintWith(uint64(stat.Blocks)).MultUint64OrNone(uint64(stat.Bsize))
	fs.Free = opt.UintWith(uint64(stat.Bfree)).MultUint64OrNone(uint64(stat.Bsize))
	fs.Avail = opt.UintWith(uint64(stat.Bavail)).MultUint64OrNone(uint64(stat.Bsize))
	fs.Files = opt.UintWith(stat.Files)
	fs.FreeFiles = opt.UintWith(uint64(stat.Ffree))

	fs.fillMetrics()

	return nil
}
