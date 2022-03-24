package filesystem

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/opt"
	"github.com/elastic/gosigar/sys/windows"
)

func parseMounts(_ string, filter func(FSStat) bool) ([]FSStat, error) {
	drives, err := windows.GetAccessPaths()
	if err != nil {
		return nil, fmt.Errorf("GetAccessPaths failed: %w", err)
	}

	driveList := []FSStat{}
	for _, drive := range drives {
		fsType, err := windows.GetFilesystemType(drive)
		if err != nil {
			return nil, fmt.Errorf("GetFilesystemType failed: %w", err)
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

func (fs *FSStat) GetUsage() error {
	freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes, err := windows.GetDiskFreeSpaceEx(fs.Directory)
	if err != nil {
		return errors.Wrap(err, "GetDiskFreeSpaceEx failed")
	}

	fs.Total = opt.UintWith(totalNumberOfBytes)
	fs.Free = opt.UintWith(totalNumberOfFreeBytes)
	fs.Avail = opt.UintWith(freeBytesAvailable)

	fs.fillMetrics()

	return nil
}
