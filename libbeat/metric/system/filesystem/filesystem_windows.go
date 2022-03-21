package filesystem

import (
	"github.com/elastic/beats/v7/libbeat/opt"
	"github.com/elastic/gosigar/sys/windows"
	"github.com/pkg/errors"
)

func parseMounts(_ string, filter func(FSStat) bool) ([]FSStat, error) {
	drives, err := windows.GetAccessPaths()
	if err != nil {
		return nil, errors.Wrap(err, "GetAccessPaths failed")
	}

	driveList := []FSStat{}
	for _, drive := range drives {
		fsType, err := windows.GetFilesystemType(drive)
		if err != nil {
			return nil, errors.Wrapf(err, "GetFilesystemType failed")
		}
		if fsType != "" {
			driveList = append(driveList, FSStat{
				Directory: drive,
				Device:    drive,
				Type:      fsType,
			})
		}
	}

	return driveList, nil
}

func (fs *FSStat) getUsage() error {
	freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes, err := windows.GetDiskFreeSpaceEx(fs.Directory)
	if err != nil {
		return errors.Wrap(err, "GetDiskFreeSpaceEx failed")
	}

	fs.Total = opt.UintWith(totalNumberOfBytes)
	fs.Free = opt.UintWith(totalNumberOfFreeBytes)
	fs.Used.Bytes = fs.Total.SubtractOrNone(fs.Free)
	fs.Avail = opt.UintWith(freeBytesAvailable)

	return nil
}
