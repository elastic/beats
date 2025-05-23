// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cpio

import (
	"io/fs"
	"strconv"
)

// https://manpages.ubuntu.com/manpages/bionic/en/man5/cpio.5.html
//
// mode    The mode specifies both the regular permissions and the file type.  It consists of
// several bit fields as follows:
// 0170000  This masks the file type bits.
// 0140000  File type value for sockets.
// 0120000  File type value for symbolic links.  For symbolic links, the link body is
// 		 stored as file data.
// 0100000  File type value for regular files.
// 0060000  File type value for block special devices.
// 0040000  File type value for directories.
// 0020000  File type value for character special devices.
// 0010000  File type value for named pipes or FIFOs.
// 0004000  SUID bit.
// 0002000  SGID bit.
// 0001000  Sticky bit.  On some systems, this modifies the behavior of executables
// 		 and/or directories.
// 0000777  The lower 9 bits specify read/write/execute permissions for world, group,
// 		 and user following standard POSIX conventions.

const (
	fileTypeMask     uint64 = 0170000
	fileTypeSocket   uint64 = 0140000
	fileTypeSymlink  uint64 = 0120000
	fileTypeBlockDev uint64 = 0060000
	fileTypeDir      uint64 = 0040000
	fileTypeCharDev  uint64 = 0020000
	fileTypePipe     uint64 = 0010000
	suidBit          uint64 = 0004000
	sgidBit          uint64 = 0002000
	stickyBit        uint64 = 0001000
)

func parseFileMode(m [6]byte) (fm fs.FileMode, err error) {
	mode, err := strconv.ParseUint(string(m[:]), 8, 64)
	if err != nil {
		return fm, err
	}

	fm = fs.FileMode(mode).Perm()

	if mode&suidBit != 0 {
		fm |= fs.ModeSetuid
	}

	if mode&sgidBit != 0 {
		fm |= fs.ModeSetgid
	}

	if mode&stickyBit != 0 {
		fm |= fs.ModeSticky
	}

	mode &= fileTypeMask

	if mode == fileTypeSocket {
		fm |= fs.ModeSocket
	}

	if mode == fileTypeSymlink {
		fm |= fs.ModeSymlink
	}

	if mode == fileTypeBlockDev {
		fm |= fs.ModeDevice
	}

	if mode == fileTypeDir {
		fm |= fs.ModeDir
	}

	if mode == fileTypeCharDev {
		fm |= fs.ModeCharDevice
	}

	if mode == fileTypePipe {
		fm |= fs.ModeNamedPipe
	}

	return fm, nil
}
