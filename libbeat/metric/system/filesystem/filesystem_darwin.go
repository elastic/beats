package filesystem

/*
#include <stdlib.h>
#include <sys/sysctl.h>
#include <sys/mount.h>
#include <mach/mach_init.h>
#include <mach/mach_host.h>
#include <mach/host_info.h>
#include <libproc.h>
#include <mach/processor_info.h>
#include <mach/vm_map.h>
*/
import "C"

import (
	"bytes"
	"syscall"
)

func parseMounts(path string, filter func(FSStat) bool) ([]FSStat, error) {
	num, err := syscall.Getfsstat(nil, C.MNT_NOWAIT)
	if err != nil {
		return nil, err
	}

	buf := make([]syscall.Statfs_t, num)

	_, err = syscall.Getfsstat(buf, C.MNT_NOWAIT)
	if err != nil {
		return nil, err
	}

	fslist := make([]FSStat, 0, num)

	for i := 0; i < num; i++ {
		fs := FSStat{}
		fs.Directory = byteListToString(buf[i].Mntonname[:])
		fs.Device = byteListToString(buf[i].Mntfromname[:])
		fs.Type = byteListToString(buf[i].Fstypename[:])

		fslist = append(fslist, fs)
	}
	return fslist, nil
}

func byteListToString(raw []int8) string {
	byteList := make([]byte, len(raw))

	for pos, singleByte := range raw {
		byteList[pos] = byte(singleByte)
		if singleByte == 0 {
			break
		}
	}

	return string(bytes.Trim(byteList, "\x00"))
}
