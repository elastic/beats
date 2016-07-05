package gosigar_test

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"testing"

	sigar "github.com/elastic/gosigar"
	"github.com/stretchr/testify/assert"
)

var procinfo map[string]string

func setUp(t testing.TB) {
	out, err := exec.Command("/bin/ps", "-p1", "-c", "-opid,comm,stat,ppid,pgid,tty,pri,ni").Output()
	if err != nil {
		t.Fatal(err)
	}
	rdr := bufio.NewReader(bytes.NewReader(out))
	_, err = rdr.ReadString('\n') // skip header
	if err != nil {
		t.Fatal(err)
	}
	data, err := rdr.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	procinfo = make(map[string]string, 8)
	fields := strings.Fields(data)
	procinfo["pid"] = fields[0]
	procinfo["name"] = fields[1]
	procinfo["stat"] = fields[2]
	procinfo["ppid"] = fields[3]
	procinfo["pgid"] = fields[4]
	procinfo["tty"] = fields[5]
	procinfo["prio"] = fields[6]
	procinfo["nice"] = fields[7]

}

func tearDown(t testing.TB) {
}

/* ProcState.Get() call task_info, which on Mac OS X requires root
   or a signed executable. Skip the test if not running as root
   to accommodate automated tests, but let users test locally using
   `sudo -E go test`
*/

func TestDarwinProcState(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	state := sigar.ProcState{}
	usr, err := user.Current()
	if err == nil && usr.Username == "root" {
		if assert.NoError(t, state.Get(1)) {

			ppid, _ := strconv.Atoi(procinfo["ppid"])
			pgid, _ := strconv.Atoi(procinfo["pgid"])

			assert.Equal(t, procinfo["name"], state.Name)
			assert.Equal(t, ppid, state.Ppid)
			assert.Equal(t, pgid, state.Pgid)
			assert.Equal(t, 1, state.Pgid)
			assert.Equal(t, 0, state.Ppid)
		}
	} else {
		fmt.Println("Skipping ProcState test; run as root to test")
	}
}
