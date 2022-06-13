// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package proc

import (
	"errors"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	ErrInvalidProcNsPidContent   = errors.New("invalid /proc/ns/pid content")
	ErrInvalidProcNsPidInoNumber = errors.New("invalid /proc/ns/pid ino number")
)

type NamespaceInfo struct {
	Ino string
}

// ReadStat reads process namespace information from /proc/<pid>/ns/pid.
// Due to https://github.com/golang/go/issues/49580 implementing without FS.
func ReadNamespace(root string, pid string) (nsInfo NamespaceInfo, err error) {

	// ReadLink content in order to pull the namespace ino
	nsLink, err := ReadLink(root, pid, filepath.Join("ns", "pid"))
	if err != nil {
		return
	}

	// Parse the namespace ino from link content
	nsIno, err := parseNamespaceIno(nsLink)
	if err != nil {
		return
	}

	// Set the parsed ino
	nsInfo.Ino = nsIno

	return nsInfo, nil
}

func parseNamespaceIno(nsLink string) (string, error) {
	// Proc ns pid link example
	// pid:[4026532605]

	// Split the link content into two parts
	details := strings.Split(nsLink, ":")

	// Fail if more than two parts and ino isn't wrapped as expected
	if len(details) != 2 ||
		!strings.HasSuffix(details[1], "]") ||
		!strings.HasPrefix(details[1], "[") {
		return "", ErrInvalidProcNsPidContent
	}

	// Slice the wrapping from the ino
	ino := details[1][1 : len(details[1])-1]

	// Check if ino is number
	_, err := strconv.Atoi(ino)
	if len(ino) == 0 || err != nil {
		return "", ErrInvalidProcNsPidInoNumber
	}

	return ino, nil
}
