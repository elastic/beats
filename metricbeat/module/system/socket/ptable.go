package socket

import (
	"os"
	"strconv"
	"strings"

	"github.com/joeshaw/multierror"
	"github.com/prometheus/procfs"
)

// process tools

// Proc contains static process information.
type Proc struct {
	PID        int
	Command    string
	Executable string
	CmdLine    string
}

// ProcTable contains all of the active processes (if the current user is root).
type ProcTable struct {
	fs     procfs.FS
	procs  map[int]*Proc
	inodes map[uint32]*Proc
	euid   int
}

// NewProcTable returns a new ProcTable that reads data from the /proc
// directory by default. An alternative proc filesystem mountpoint can be
// specified through the mountpoint parameter.
func NewProcTable(mountpoint string) (*ProcTable, error) {
	if mountpoint == "" {
		mountpoint = procfs.DefaultMountPoint
	}

	fs, err := procfs.NewFS(mountpoint)
	if err != nil {
		return nil, err
	}

	p := &ProcTable{fs: fs, euid: os.Geteuid()}
	p.Refresh()
	return p, nil
}

// Refresh updates the process table with new processes and removes processes
// that have exited. It collects the PID, command, and socket inode information.
// If running as non-root, only information from the current process will be
// collected.
func (t *ProcTable) Refresh() error {
	var err error
	var procs []procfs.Proc
	if t.euid == 0 {
		procs, err = t.fs.AllProcs()
		if err != nil {
			return err
		}
	} else {
		proc, err := t.fs.Self()
		if err != nil {
			return err
		}
		procs = append(procs, proc)
	}

	var errs multierror.Errors
	inodes := map[uint32]*Proc{}
	cachedProcs := make(map[int]*Proc, len(procs))
	for _, p := range procs {
		proc := t.procs[p.PID]

		// Cache miss.
		if proc == nil {
			proc = &Proc{PID: p.PID}

			if proc.Executable, err = p.Executable(); err != nil {
				errs = append(errs, err)
			}
			if proc.Command, err = p.Comm(); err != nil {
				errs = append(errs, err)
			}
			if cmdline, err := p.CmdLine(); err != nil {
				errs = append(errs, err)
			} else {
				proc.CmdLine = strings.Join(cmdline, " ")
			}
		}
		cachedProcs[proc.PID] = proc

		// Always update map socket inode to Proc.
		socketInodes, err := socketInodes(&p)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for _, inode := range socketInodes {
			inodes[inode] = proc
		}
	}

	t.procs = cachedProcs
	t.inodes = inodes
	return errs.Err()
}

func socketInodes(p *procfs.Proc) ([]uint32, error) {
	fds, err := p.FileDescriptorTargets()
	if err != nil {
		return nil, err
	}

	var inodes []uint32
	for _, fd := range fds {
		if strings.HasPrefix(fd, "socket:[") {
			inode, err := strconv.ParseInt(fd[8:len(fd)-1], 10, 64)
			if err != nil {
				continue
			}

			inodes = append(inodes, uint32(inode))
		}
	}

	return inodes, nil
}

// ProcessBySocketInode returns the Proc associated with the given socket
// inode.
func (t *ProcTable) ProcessBySocketInode(inode uint32) *Proc {
	return t.inodes[inode]
}
