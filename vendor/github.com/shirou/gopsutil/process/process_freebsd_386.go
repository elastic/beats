// +build freebsd
// +build 386

package process

// copied from sys/sysctl.h
const (
	CTLKern          = 1  // "high kernel": proc, limits
	KernProc         = 14 // struct: process entries
	KernProcPID      = 1  // by process id
	KernProcProc     = 8  // only return procs
	KernProcPathname = 12 // path to executable
	KernProcArgs     = 7  // get/set arguments/proctitle
)

const (
	SIDL   = 1
	SRUN   = 2
	SSLEEP = 3
	SSTOP  = 4
	SZOMB  = 5
	SWAIT  = 6
	SLOCK  = 7
)

const (
	sizeOfKinfoVmentry = 0x244 // TODO: really?
	sizeOfKinfoProc    = 0x220
)

type Timespec struct {
	Sec  int32
	Nsec int32
}

type Timeval struct {
	Sec  int32
	Usec int32
}

type Rusage struct {
	Utime    Timeval
	Stime    Timeval
	Maxrss   int32
	Ixrss    int32
	Idrss    int32
	Isrss    int32
	Minflt   int32
	Majflt   int32
	Nswap    int32
	Inblock  int32
	Oublock  int32
	Msgsnd   int32
	Msgrcv   int32
	Nsignals int32
	Nvcsw    int32
	Nivcsw   int32
}

// copied from sys/user.h
type KinfoProc struct {
	Structsize   int32
	Layout       int32
	Args         int32
	Paddr        int32
	Addr         int32
	Tracep       int32
	Textvp       int32
	Fd           int32
	Vmspace      int32
	Wchan        int32
	Pid          int32
	Ppid         int32
	Pgid         int32
	Tpgid        int32
	Sid          int32
	Tsid         int32
	Jobc         [2]byte
	SpareShort1  [2]byte
	Tdev         int32
	Siglist      [16]byte
	Sigmask      [16]byte
	Sigignore    [16]byte
	Sigcatch     [16]byte
	Uid          int32
	Ruid         int32
	Svuid        int32
	Rgid         int32
	Svgid        int32
	Ngroups      int16
	SpareShort2  [2]byte
	Groups       [64]byte
	Size         int32
	Rssize       int32
	Swrss        int32
	Tsize        int32
	Dsize        int32
	Ssize        int32
	Xstat        [2]byte
	Acflag       [2]byte
	Pctcpu       int32
	Estcpu       int32
	Slptime      int32
	Swtime       int32
	Cow          int32
	Runtime      int64
	Start        [8]byte
	Childtime    [8]byte
	Flag         int32
	Kflag        int32
	Traceflag    int32
	Stat         int8
	Nice         [1]byte
	Lock         [1]byte
	Rqindex      [1]byte
	Oncpu        [1]byte
	Lastcpu      [1]byte
	Ocomm        [17]byte
	Wmesg        [9]byte
	Login        [18]byte
	Lockname     [9]byte
	Comm         [20]int8
	Emul         [17]byte
	Sparestrings [68]byte
	Spareints    [36]byte
	CrFlags      int32
	Jid          int32
	Numthreads   int32
	Tid          int32
	Pri          int32
	Rusage       Rusage
	RusageCh     [72]byte
	Pcb          int32
	Kstack       int32
	Udata        int32
	Tdaddr       int32
	Spareptrs    [24]byte
	Spareint64s  [48]byte
	Sflag        int32
	Tdflags      int32
}
