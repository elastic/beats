//go:build aix || darwin || freebsd || linux
// +build aix darwin freebsd linux

package filesystem

import (
	"syscall"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/opt"
	"github.com/pkg/errors"
)

func (fs *FSStat) getUsage() error {
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

	fs.Used.Bytes = fs.Total.SubtractOrNone(fs.Free)

	percTotal := fs.Used.Bytes.ValueOr(0) + fs.Avail.ValueOr(0)
	if percTotal == 0 {
		return nil
	}
	// I'm not sure why this does Used + avail instead of total, but I'm too afraid to change it
	perc := float64(fs.Used.Bytes.ValueOr(0)) / float64(percTotal)
	fs.Used.Pct = opt.FloatWith(common.Round(perc, common.DefaultDecimalPlacesCount))

	return nil
}
