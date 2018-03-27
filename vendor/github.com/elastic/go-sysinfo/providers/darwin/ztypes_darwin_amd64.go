// Created by cgo -godefs - DO NOT EDIT
// cgo -godefs defs_darwin.go

package darwin

type processState uint32

const (
	stateSIDL processState = iota + 1
	stateRun
	stateSleep
	stateStop
	stateZombie
)

const argMax = 0x40000

type bsdInfo struct {
	Pbi_flags        uint32
	Pbi_status       uint32
	Pbi_xstatus      uint32
	Pbi_pid          uint32
	Pbi_ppid         uint32
	Pbi_uid          uint32
	Pbi_gid          uint32
	Pbi_ruid         uint32
	Pbi_rgid         uint32
	Pbi_svuid        uint32
	Pbi_svgid        uint32
	Rfu_1            uint32
	Pbi_comm         [16]int8
	Pbi_name         [32]int8
	Pbi_nfiles       uint32
	Pbi_pgid         uint32
	Pbi_pjobc        uint32
	E_tdev           uint32
	E_tpgid          uint32
	Pbi_nice         int32
	Pbi_start_tvsec  uint64
	Pbi_start_tvusec uint64
}

type procTaskInfo struct {
	Virtual_size      uint64
	Resident_size     uint64
	Total_user        uint64
	Total_system      uint64
	Threads_user      uint64
	Threads_system    uint64
	Policy            int32
	Faults            int32
	Pageins           int32
	Cow_faults        int32
	Messages_sent     int32
	Messages_received int32
	Syscalls_mach     int32
	Syscalls_unix     int32
	Csw               int32
	Threadnum         int32
	Numrunning        int32
	Priority          int32
}

type procTaskAllInfo struct {
	Pbsd   bsdInfo
	Ptinfo procTaskInfo
}

type vinfoStat struct {
	Dev           uint32
	Mode          uint16
	Nlink         uint16
	Ino           uint64
	Uid           uint32
	Gid           uint32
	Atime         int64
	Atimensec     int64
	Mtime         int64
	Mtimensec     int64
	Ctime         int64
	Ctimensec     int64
	Birthtime     int64
	Birthtimensec int64
	Size          int64
	Blocks        int64
	Blksize       int32
	Flags         uint32
	Gen           uint32
	Rdev          uint32
	Qspare        [2]int64
}

type fsid struct {
	Val [2]int32
}

type vnodeInfo struct {
	Stat vinfoStat
	Type int32
	Pad  int32
	Fsid fsid
}

type vnodeInfoPath struct {
	Vi   vnodeInfo
	Path [1024]int8
}

type procVnodePathInfo struct {
	Cdir vnodeInfoPath
	Rdir vnodeInfoPath
}
