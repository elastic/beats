// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package hostprocesses

type hostProcess struct {
	PID              int64  `osquery:"pid" desc:"Process (or thread) ID"`
	Name             string `osquery:"name" desc:"The process path or shorthand argv[0]"`
	Path             string `osquery:"path" desc:"Path to executed binary"`
	Cmdline          string `osquery:"cmdline" desc:"Complete argv"`
	State            string `osquery:"state" desc:"Process state"`
	Cwd              string `osquery:"cwd" desc:"Process current working directory"`
	Root             string `osquery:"root" desc:"Process virtual root directory"`
	UID              int64  `osquery:"uid" desc:"Unsigned user ID"`
	GID              int64  `osquery:"gid" desc:"Unsigned group ID"`
	EUID             int64  `osquery:"euid" desc:"Unsigned effective user ID"`
	EGID             int64  `osquery:"egid" desc:"Unsigned effective group ID"`
	SUID             int64  `osquery:"suid" desc:"Unsigned saved user ID"`
	SGID             int64  `osquery:"sgid" desc:"Unsigned saved group ID"`
	OnDisk           int    `osquery:"on_disk" desc:"The process path exists yes=1, no=0, unknown=-1"`
	WiredSize        int64  `osquery:"wired_size" desc:"Bytes of unpageable memory used by process"`
	ResidentSize     int64  `osquery:"resident_size" desc:"Bytes of private memory used by process"`
	TotalSize        int64  `osquery:"total_size" desc:"Total virtual memory size"`
	UserTime         int64  `osquery:"user_time" desc:"CPU time in milliseconds spent in user space"`
	SystemTime       int64  `osquery:"system_time" desc:"CPU time in milliseconds spent in kernel space"`
	DiskBytesRead    int64  `osquery:"disk_bytes_read" desc:"Bytes read from disk"`
	DiskBytesWritten int64  `osquery:"disk_bytes_written" desc:"Bytes written to disk"`
	StartTime        int64  `osquery:"start_time" desc:"Process start time in seconds since Epoch, in case of error -1"`
	Parent           int64  `osquery:"parent" desc:"Process parent's PID"`
	Pgroup           int64  `osquery:"pgroup" desc:"Process group"`
	Threads          int    `osquery:"threads" desc:"Number of threads used by process"`
	Nice             int    `osquery:"nice" desc:"Process nice level (-20 to 20, default 0)"`
}
