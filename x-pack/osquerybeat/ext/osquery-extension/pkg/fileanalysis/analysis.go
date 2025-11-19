// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fileanalysis

type fileAnalysis struct {
	Path         string `osquery:"path" desc:"Path to the file"`
	Mode         string `osquery:"mode" desc:"File mode (permissions)"`
	UID          int64  `osquery:"uid" desc:"User ID of the file owner"`
	GID          int64  `osquery:"gid" desc:"Group ID of the file"`
	Size         int64  `osquery:"size" desc:"Size of the file in bytes"`
	Mtime        int64  `osquery:"mtime" desc:"Last modification time in Unix timestamp"`
	FileType     string `osquery:"file_type" desc:"File type information from 'file' command"`
	CodeSign     string `osquery:"code_sign" desc:"Code signing information from 'codesign' command"`
	Dependencies string `osquery:"dependencies" desc:"Library dependencies from 'otool' command"`
	Symbols      string `osquery:"symbols" desc:"Symbol table from 'nm' command"`
	Strings      string `osquery:"strings" desc:"Printable strings from the file"`
}
