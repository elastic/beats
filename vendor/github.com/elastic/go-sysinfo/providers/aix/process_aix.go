// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package aix

/*
#cgo LDFLAGS: -L/usr/lib -lperfstat

#include <libperfstat.h>
#include <procinfo.h>
#include <sys/proc.h>

*/
import "C"

import (
	"bytes"
	"encoding/binary"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/pkg/errors"

	"github.com/elastic/go-sysinfo/types"
)

// Processes returns a list of all actives processes.
func (aixSystem) Processes() ([]types.Process, error) {
	// Retrieve processes using /proc instead of calling
	// getprocs which will also retrieve kernel threads.
	files, err := ioutil.ReadDir("/proc")
	if err != nil {
		return nil, errors.Wrap(err, "error while reading /proc")
	}

	processes := make([]types.Process, 0, len(files))
	for _, f := range files {
		// Check that the file is a correct process directory.
		// /proc also contains special files (/proc/version) and threads
		// directories (/proc/pid directory but without any "as" file)
		if _, err := os.Stat("/proc/" + f.Name() + "/as"); err == nil {
			pid, _ := strconv.Atoi(f.Name())
			processes = append(processes, &process{pid: pid})
		}
	}

	return processes, nil
}

// Process returns the process designed by PID.
func (aixSystem) Process(pid int) (types.Process, error) {
	p := process{pid: pid}
	return &p, nil
}

// Self returns the current process.
func (s aixSystem) Self() (types.Process, error) {
	return s.Process(os.Getpid())
}

type process struct {
	pid  int
	info *types.ProcessInfo
	env  map[string]string
}

// PID returns the PID of a process.
func (p *process) PID() int {
	return p.pid
}

// Parent returns the parent of a process.
func (p *process) Parent() (types.Process, error) {
	info, err := p.Info()
	if err != nil {
		return nil, err
	}
	return &process{pid: info.PPID}, nil
}

// Info returns all information about the process.
func (p *process) Info() (types.ProcessInfo, error) {
	if p.info != nil {
		return *p.info, nil
	}

	p.info = &types.ProcessInfo{
		PID: p.pid,
	}

	// Retrieve PPID and StartTime
	info := C.struct_procsinfo64{}
	cpid := C.pid_t(p.pid)

	num, err := C.getprocs(unsafe.Pointer(&info), C.sizeof_struct_procsinfo64, nil, 0, &cpid, 1)
	if num != 1 {
		err = syscall.ESRCH
	}
	if err != nil {
		return types.ProcessInfo{}, errors.Wrap(err, "error while calling getprocs")
	}

	p.info.PPID = int(info.pi_ppid)
	// pi_start is the time in second since the process have started.
	p.info.StartTime = time.Unix(0, int64(uint64(info.pi_start)*1000*uint64(time.Millisecond)))

	// Retrieve arguments and executable name
	// If buffer is not large enough, args are truncated
	buf := make([]byte, 8192)
	var args []string
	if _, err := C.getargs(unsafe.Pointer(&info), C.sizeof_struct_procsinfo64, (*C.char)(&buf[0]), 8192); err != nil {
		return types.ProcessInfo{}, errors.Wrap(err, "error while calling getargs")
	}

	bbuf := bytes.NewBuffer(buf)
	for {
		arg, err := bbuf.ReadBytes(0)
		if err == io.EOF || arg[0] == 0 {
			break
		}
		if err != nil {
			return types.ProcessInfo{}, errors.Wrap(err, "error while reading arguments")
		}

		args = append(args, string(chop(arg)))
	}

	// For some special programs, getargs might return an empty buffer.
	if len(args) == 0 {
		args = append(args, "")
	}

	// The first element of the arguments list is the executable path.
	// There are some exceptions which don't have an executable path
	// but rather a special name directly in args[0].
	if strings.Contains(args[0], "sshd: ") {
		// ssh connections can be named "sshd: root@pts/11".
		// If we are using filepath.Base, the result will only
		// be 11 because of the last "/".
		p.info.Name = args[0]
	} else {
		p.info.Name = filepath.Base(args[0])
	}

	// The process was launched using its absolute path, so we can retrieve
	// the executable path from its "name".
	if filepath.IsAbs(args[0]) {
		p.info.Exe = filepath.Clean(args[0])
	} else {
		// TODO: improve this case. The executable full path can still
		// be retrieve in some cases. Look at os/executable_path.go
		// in the stdlib.
		// For the moment, let's "exe" be the same as "name"
		p.info.Exe = p.info.Name
	}
	p.info.Args = args

	// Get CWD
	cwd, err := os.Readlink("/proc/" + strconv.Itoa(p.pid) + "/cwd")
	if err != nil {
		if !os.IsNotExist(err) {
			return types.ProcessInfo{}, errors.Wrapf(err, "error while reading /proc/%s/cwd", strconv.Itoa(p.pid))
		}
	}

	p.info.CWD = strings.TrimSuffix(cwd, "/")

	return *p.info, nil
}

// Environment returns the environment of a process.
func (p *process) Environment() (map[string]string, error) {
	if p.env != nil {
		return p.env, nil
	}
	p.env = map[string]string{}

	/* If buffer is not large enough, args are truncated */
	buf := make([]byte, 8192)
	info := C.struct_procsinfo64{}
	info.pi_pid = C.pid_t(p.pid)

	if _, err := C.getevars(unsafe.Pointer(&info), C.sizeof_struct_procsinfo64, (*C.char)(&buf[0]), 8192); err != nil {
		return nil, errors.Wrap(err, "error while calling getevars")
	}

	bbuf := bytes.NewBuffer(buf)

	delim := []byte{61} // "="

	for {
		line, err := bbuf.ReadBytes(0)
		if err == io.EOF || line[0] == 0 {
			break
		}
		if err != nil {
			return nil, errors.Wrap(err, "error while calling getevars")
		}

		pair := bytes.SplitN(chop(line), delim, 2)
		if len(pair) != 2 {
			return nil, errors.Wrap(err, "error reading process environment")
		}
		p.env[string(pair[0])] = string(pair[1])
	}

	return p.env, nil
}

// User returns the user IDs of a process.
func (p *process) User() (types.UserInfo, error) {
	var prcred prcred
	if err := p.decodeProcfsFile("cred", &prcred); err != nil {
		return types.UserInfo{}, err
	}
	return types.UserInfo{
		UID:  strconv.Itoa(int(prcred.Ruid)),
		EUID: strconv.Itoa(int(prcred.Euid)),
		SUID: strconv.Itoa(int(prcred.Suid)),
		GID:  strconv.Itoa(int(prcred.Rgid)),
		EGID: strconv.Itoa(int(prcred.Egid)),
		SGID: strconv.Itoa(int(prcred.Sgid)),
	}, nil
}

// Memory returns the current memory usage of a process.
func (p *process) Memory() (types.MemoryInfo, error) {
	var mem types.MemoryInfo
	pagesize := uint64(os.Getpagesize())

	info := C.struct_procsinfo64{}
	cpid := C.pid_t(p.pid)

	num, err := C.getprocs(unsafe.Pointer(&info), C.sizeof_struct_procsinfo64, nil, 0, &cpid, 1)
	if num != 1 {
		err = syscall.ESRCH
	}
	if err != nil {
		return types.MemoryInfo{}, errors.Wrap(err, "error while calling getprocs")
	}

	mem.Resident = uint64(info.pi_drss+info.pi_trss) * pagesize
	mem.Virtual = uint64(info.pi_dvm) * pagesize

	return mem, nil
}

// CPUTime returns the current CPU usage of a process.
func (p *process) CPUTime() (types.CPUTimes, error) {
	var pstatus pstatus
	if err := p.decodeProcfsFile("status", &pstatus); err != nil {
		return types.CPUTimes{}, err
	}
	return types.CPUTimes{
		User:   time.Duration(pstatus.Utime.Sec*1e9 + int64(pstatus.Utime.Nsec)),
		System: time.Duration(pstatus.Stime.Sec*1e9 + int64(pstatus.Stime.Nsec)),
	}, nil
}

func (p *process) decodeProcfsFile(name string, data interface{}) error {
	fileName := "/proc/" + strconv.Itoa(p.pid) + "/" + name

	file, err := os.Open(fileName)
	if err != nil {
		return errors.Wrapf(err, "error while opening %s", fileName)
	}
	defer file.Close()

	if err := binary.Read(file, binary.BigEndian, data); err != nil {
		return errors.Wrapf(err, "error while decoding %s", fileName)
	}

	return nil
}

func chop(buf []byte) []byte {
	return buf[0 : len(buf)-1]
}
