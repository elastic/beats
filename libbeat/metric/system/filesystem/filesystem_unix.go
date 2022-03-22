//go:build aix || darwin || freebsd || linux
// +build aix darwin freebsd linux

package filesystem

import (
	"syscall"

	"github.com/elastic/beats/v7/libbeat/opt"
	"github.com/pkg/errors"
)

// GetUsage returns the filesystem usage
func (fs *FSStat) GetUsage() error {
	stat := syscall.Statfs_t{}
	err := syscall.Statfs(fs.Directory, &stat)
	if err != nil {
		return errors.Wrap(err, "error in Statfs syscall")
	}

	fs.Total = opt.UintWith(uint64(stat.Blocks)).MultUint64OrNone(uint64(stat.Bsize))
	fs.Free = opt.UintWith(uint64(stat.Bfree)).MultUint64OrNone(uint64(stat.Bsize))
	fs.Avail = opt.UintWith(uint64(stat.Bavail)).MultUint64OrNone(uint64(stat.Bsize))
	fs.Files = opt.UintWith(stat.Files)
	fs.FreeFiles = opt.UintWith(uint64(stat.Ffree))

	fs.fillMetrics()

	return nil
}
