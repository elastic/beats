package process

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/user"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/shirou/gopsutil/internal/common"
	"github.com/stretchr/testify/assert"
)

var mu sync.Mutex

func skipIfNotImplementedErr(t *testing.T, err error) {
	if err == common.ErrNotImplementedError {
		t.Skip("not implemented")
	}
}

func testGetProcess() Process {
	checkPid := os.Getpid() // process.test
	ret, _ := NewProcess(int32(checkPid))
	return *ret
}

func Test_Pids(t *testing.T) {
	ret, err := Pids()
	skipIfNotImplementedErr(t, err)
	if err != nil {
		t.Errorf("error %v", err)
	}
	if len(ret) == 0 {
		t.Errorf("could not get pids %v", ret)
	}
}

func Test_Pids_Fail(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin only")
	}

	mu.Lock()
	defer mu.Unlock()

	invoke = common.FakeInvoke{Suffix: "fail"}
	ret, err := Pids()
	skipIfNotImplementedErr(t, err)
	invoke = common.Invoke{}
	if err != nil {
		t.Errorf("error %v", err)
	}
	if len(ret) != 9 {
		t.Errorf("wrong getted pid nums: %v/%d", ret, len(ret))
	}
}
func Test_Pid_exists(t *testing.T) {
	checkPid := os.Getpid()

	ret, err := PidExists(int32(checkPid))
	skipIfNotImplementedErr(t, err)
	if err != nil {
		t.Errorf("error %v", err)
	}

	if ret == false {
		t.Errorf("could not get process exists: %v", ret)
	}
}

func Test_NewProcess(t *testing.T) {
	checkPid := os.Getpid()

	ret, err := NewProcess(int32(checkPid))
	skipIfNotImplementedErr(t, err)
	if err != nil {
		t.Errorf("error %v", err)
	}
	empty := &Process{}
	if runtime.GOOS != "windows" { // Windows pid is 0
		if empty == ret {
			t.Errorf("error %v", ret)
		}
	}

}

func Test_Process_memory_maps(t *testing.T) {
	checkPid := os.Getpid()

	ret, err := NewProcess(int32(checkPid))
	skipIfNotImplementedErr(t, err)
	if err != nil {
		t.Errorf("error %v", err)
	}

	// ungrouped memory maps
	mmaps, err := ret.MemoryMaps(false)
	skipIfNotImplementedErr(t, err)
	if err != nil {
		t.Errorf("memory map get error %v", err)
	}
	empty := MemoryMapsStat{}
	for _, m := range *mmaps {
		if m == empty {
			t.Errorf("memory map get error %v", m)
		}
	}

	// grouped memory maps
	mmaps, err = ret.MemoryMaps(true)
	skipIfNotImplementedErr(t, err)
	if err != nil {
		t.Errorf("memory map get error %v", err)
	}
	if len(*mmaps) != 1 {
		t.Errorf("grouped memory maps length (%v) is not equal to 1", len(*mmaps))
	}
	if (*mmaps)[0] == empty {
		t.Errorf("memory map is empty")
	}
}
func Test_Process_MemoryInfo(t *testing.T) {
	p := testGetProcess()

	v, err := p.MemoryInfo()
	skipIfNotImplementedErr(t, err)
	if err != nil {
		t.Errorf("getting memory info error %v", err)
	}
	empty := MemoryInfoStat{}
	if v == nil || *v == empty {
		t.Errorf("could not get memory info %v", v)
	}
}

func Test_Process_CmdLine(t *testing.T) {
	p := testGetProcess()

	v, err := p.Cmdline()
	skipIfNotImplementedErr(t, err)
	if err != nil {
		t.Errorf("getting cmdline error %v", err)
	}
	if !strings.Contains(v, "process.test") {
		t.Errorf("invalid cmd line %v", v)
	}
}

func Test_Process_CmdLineSlice(t *testing.T) {
	p := testGetProcess()

	v, err := p.CmdlineSlice()
	skipIfNotImplementedErr(t, err)
	if err != nil {
		t.Fatalf("getting cmdline slice error %v", err)
	}
	if !reflect.DeepEqual(v, os.Args) {
		t.Errorf("returned cmdline slice not as expected:\nexp: %v\ngot: %v", os.Args, v)
	}
}

func Test_Process_Ppid(t *testing.T) {
	p := testGetProcess()

	v, err := p.Ppid()
	skipIfNotImplementedErr(t, err)
	if err != nil {
		t.Errorf("getting ppid error %v", err)
	}
	if v == 0 {
		t.Errorf("return value is 0 %v", v)
	}
}

func Test_Process_Status(t *testing.T) {
	p := testGetProcess()

	v, err := p.Status()
	skipIfNotImplementedErr(t, err)
	if err != nil {
		t.Errorf("getting status error %v", err)
	}
	if v != "R" && v != "S" {
		t.Errorf("could not get state %v", v)
	}
}

func Test_Process_Terminal(t *testing.T) {
	p := testGetProcess()

	_, err := p.Terminal()
	skipIfNotImplementedErr(t, err)
	if err != nil {
		t.Errorf("getting terminal error %v", err)
	}
}

func Test_Process_IOCounters(t *testing.T) {
	p := testGetProcess()

	v, err := p.IOCounters()
	skipIfNotImplementedErr(t, err)
	if err != nil {
		t.Errorf("getting iocounter error %v", err)
		return
	}
	empty := &IOCountersStat{}
	if v == empty {
		t.Errorf("error %v", v)
	}
}

func Test_Process_NumCtx(t *testing.T) {
	p := testGetProcess()

	_, err := p.NumCtxSwitches()
	skipIfNotImplementedErr(t, err)
	if err != nil {
		t.Errorf("getting numctx error %v", err)
		return
	}
}

func Test_Process_Nice(t *testing.T) {
	p := testGetProcess()

	n, err := p.Nice()
	skipIfNotImplementedErr(t, err)
	if err != nil {
		t.Errorf("getting nice error %v", err)
	}
	if n != 0 && n != 20 && n != 8 {
		t.Errorf("invalid nice: %d", n)
	}
}
func Test_Process_NumThread(t *testing.T) {
	p := testGetProcess()

	n, err := p.NumThreads()
	skipIfNotImplementedErr(t, err)
	if err != nil {
		t.Errorf("getting NumThread error %v", err)
	}
	if n < 0 {
		t.Errorf("invalid NumThread: %d", n)
	}
}

func Test_Process_Threads(t *testing.T) {
	p := testGetProcess()

	n, err := p.NumThreads()
	skipIfNotImplementedErr(t, err)
	if err != nil {
		t.Errorf("getting NumThread error %v", err)
	}
	if n < 0 {
		t.Errorf("invalid NumThread: %d", n)
	}

	ts, err := p.Threads()
	skipIfNotImplementedErr(t, err)
	if err != nil {
		t.Errorf("getting Threads error %v", err)
	}
	if len(ts) != int(n) {
		t.Errorf("unexpected number of threads: %v vs %v", len(ts), n)
	}
}

func Test_Process_Name(t *testing.T) {
	p := testGetProcess()

	n, err := p.Name()
	skipIfNotImplementedErr(t, err)
	if err != nil {
		t.Errorf("getting name error %v", err)
	}
	if !strings.Contains(n, "process.test") {
		t.Errorf("invalid Exe %s", n)
	}
}
func Test_Process_Exe(t *testing.T) {
	p := testGetProcess()

	n, err := p.Exe()
	skipIfNotImplementedErr(t, err)
	if err != nil {
		t.Errorf("getting Exe error %v", err)
	}
	if !strings.Contains(n, "process.test") {
		t.Errorf("invalid Exe %s", n)
	}
}

func Test_Process_CpuPercent(t *testing.T) {
	p := testGetProcess()
	percent, err := p.Percent(0)
	skipIfNotImplementedErr(t, err)
	if err != nil {
		t.Errorf("error %v", err)
	}
	duration := time.Duration(1000) * time.Microsecond
	time.Sleep(duration)
	percent, err = p.Percent(0)
	if err != nil {
		t.Errorf("error %v", err)
	}

	numcpu := runtime.NumCPU()
	//	if percent < 0.0 || percent > 100.0*float64(numcpu) { // TODO
	if percent < 0.0 {
		t.Fatalf("CPUPercent value is invalid: %f, %d", percent, numcpu)
	}
}

func Test_Process_CpuPercentLoop(t *testing.T) {
	p := testGetProcess()
	numcpu := runtime.NumCPU()

	for i := 0; i < 2; i++ {
		duration := time.Duration(100) * time.Microsecond
		percent, err := p.Percent(duration)
		skipIfNotImplementedErr(t, err)
		if err != nil {
			t.Errorf("error %v", err)
		}
		//	if percent < 0.0 || percent > 100.0*float64(numcpu) { // TODO
		if percent < 0.0 {
			t.Fatalf("CPUPercent value is invalid: %f, %d", percent, numcpu)
		}
	}
}

func Test_Process_CreateTime(t *testing.T) {
	if os.Getenv("CIRCLECI") == "true" {
		t.Skip("Skip CI")
	}

	p := testGetProcess()

	c, err := p.CreateTime()
	skipIfNotImplementedErr(t, err)
	if err != nil {
		t.Errorf("error %v", err)
	}

	if c < 1420000000 {
		t.Errorf("process created time is wrong.")
	}

	gotElapsed := time.Since(time.Unix(int64(c/1000), 0))
	maxElapsed := time.Duration(5 * time.Second)

	if gotElapsed >= maxElapsed {
		t.Errorf("this process has not been running for %v", gotElapsed)
	}
}

func Test_Parent(t *testing.T) {
	p := testGetProcess()

	c, err := p.Parent()
	skipIfNotImplementedErr(t, err)
	if err != nil {
		t.Fatalf("error %v", err)
	}
	if c == nil {
		t.Fatalf("could not get parent")
	}
	if c.Pid == 0 {
		t.Fatalf("wrong parent pid")
	}
}

func Test_Connections(t *testing.T) {
	p := testGetProcess()
	ch0 := make(chan string)
	ch1 := make(chan string)
	go func() { // TCP listening goroutine
		addr, err := net.ResolveTCPAddr("tcp", "localhost:0") // dynamically get a random open port from OS
		if err != nil {
			t.Skip("unable to resolve localhost:", err)
		}
		l, err := net.ListenTCP(addr.Network(), addr)
		if err != nil {
			t.Skip(fmt.Sprintf("unable to listen on %v: %v", addr, err))
		}
		defer l.Close()
		ch0 <- l.Addr().String()
		for {
			conn, err := l.Accept()
			if err != nil {
				t.Skip("unable to accept connection:", err)
			}
			ch1 <- l.Addr().String()
			defer conn.Close()
		}
	}()
	go func() { // TCP client goroutine
		tcpServerAddr := <-ch0
		net.Dial("tcp", tcpServerAddr)
	}()

	tcpServerAddr := <-ch1
	tcpServerAddrIP := strings.Split(tcpServerAddr, ":")[0]
	tcpServerAddrPort, err := strconv.ParseUint(strings.Split(tcpServerAddr, ":")[1], 10, 32)
	if err != nil {
		t.Errorf("unable to parse tcpServerAddr port: %v", err)
	}
	c, err := p.Connections()
	skipIfNotImplementedErr(t, err)
	if err != nil {
		t.Errorf("error %v", err)
	}
	if len(c) == 0 {
		t.Errorf("no connections found")
	}
	found := 0
	for _, connection := range c {
		if connection.Status == "ESTABLISHED" && (connection.Laddr.IP == tcpServerAddrIP && connection.Laddr.Port == uint32(tcpServerAddrPort)) || (connection.Raddr.IP == tcpServerAddrIP && connection.Raddr.Port == uint32(tcpServerAddrPort)) {
			found++
		}
	}
	if found != 2 { // two established connections, one for the server, the other for the client
		t.Errorf(fmt.Sprintf("wrong connections: %+v", c))
	}
}

func Test_Children(t *testing.T) {
	p := testGetProcess()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("ping", "localhost", "-n", "4")
	} else {
		cmd = exec.Command("sleep", "3")
	}
	assert.Nil(t, cmd.Start())
	time.Sleep(100 * time.Millisecond)

	c, err := p.Children()
	skipIfNotImplementedErr(t, err)
	if err != nil {
		t.Fatalf("error %v", err)
	}
	if len(c) == 0 {
		t.Fatalf("children is empty")
	}
	if c[0].Pid != int32(cmd.Process.Pid) {
		t.Errorf("could not find child %d", cmd.Process.Pid)
	}
}

func Test_Username(t *testing.T) {
	myPid := os.Getpid()
	currentUser, _ := user.Current()
	myUsername := currentUser.Username

	process, _ := NewProcess(int32(myPid))
	pidUsername, err := process.Username()
	skipIfNotImplementedErr(t, err)
	assert.Equal(t, myUsername, pidUsername)

	t.Log(pidUsername)
}

func Test_CPUTimes(t *testing.T) {
	pid := os.Getpid()
	process, err := NewProcess(int32(pid))
	skipIfNotImplementedErr(t, err)
	assert.Nil(t, err)

	spinSeconds := 0.2
	cpuTimes0, err := process.Times()
	skipIfNotImplementedErr(t, err)
	assert.Nil(t, err)

	// Spin for a duration of spinSeconds
	t0 := time.Now()
	tGoal := t0.Add(time.Duration(spinSeconds*1000) * time.Millisecond)
	assert.Nil(t, err)
	for time.Now().Before(tGoal) {
		// This block intentionally left blank
	}

	cpuTimes1, err := process.Times()
	assert.Nil(t, err)

	if cpuTimes0 == nil || cpuTimes1 == nil {
		t.FailNow()
	}
	measuredElapsed := cpuTimes1.Total() - cpuTimes0.Total()
	message := fmt.Sprintf("Measured %fs != spun time of %fs\ncpuTimes0=%v\ncpuTimes1=%v",
		measuredElapsed, spinSeconds, cpuTimes0, cpuTimes1)
	assert.True(t, measuredElapsed > float64(spinSeconds)/5, message)
	assert.True(t, measuredElapsed < float64(spinSeconds)*5, message)
}

func Test_OpenFiles(t *testing.T) {
	pid := os.Getpid()
	p, err := NewProcess(int32(pid))
	skipIfNotImplementedErr(t, err)
	assert.Nil(t, err)

	v, err := p.OpenFiles()
	skipIfNotImplementedErr(t, err)
	assert.Nil(t, err)
	assert.NotEmpty(t, v) // test always open files.

	for _, vv := range v {
		assert.NotEqual(t, "", vv.Path)
	}
}

func Test_Kill(t *testing.T) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("ping", "localhost", "-n", "4")
	} else {
		cmd = exec.Command("sleep", "3")
	}
	assert.Nil(t, cmd.Start())
	time.Sleep(100 * time.Millisecond)
	p, err := NewProcess(int32(cmd.Process.Pid))
	skipIfNotImplementedErr(t, err)
	assert.Nil(t, err)
	err = p.Kill()
	skipIfNotImplementedErr(t, err)
	assert.Nil(t, err)
	cmd.Wait()
}

func Test_IsRunning(t *testing.T) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("ping", "localhost", "-n", "2")
	} else {
		cmd = exec.Command("sleep", "1")
	}
	cmd.Start()
	p, err := NewProcess(int32(cmd.Process.Pid))
	assert.Nil(t, err)
	running, err := p.IsRunning()
	if err != nil {
		t.Fatalf("IsRunning error: %v", err)
	}
	if !running {
		t.Fatalf("process should be found running")
	}
	cmd.Wait()
	running, err = p.IsRunning()
	if err != nil {
		t.Fatalf("IsRunning error: %v", err)
	}
	if running {
		t.Fatalf("process should NOT be found running")
	}
}
