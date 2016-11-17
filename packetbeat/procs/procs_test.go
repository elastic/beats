// +build !integration

package procs

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/logp"
)

type testProcFile struct {
	path     string
	contents string
	isLink   bool
}

func createFakeDirectoryStructure(prefix string, files []testProcFile) error {

	var err error
	for _, file := range files {
		dir := filepath.Dir(file.path)
		err = os.MkdirAll(filepath.Join(prefix, dir), 0755)
		if err != nil {
			return err
		}

		if !file.isLink {
			err = ioutil.WriteFile(filepath.Join(prefix, file.path),
				[]byte(file.contents), 0644)
			if err != nil {
				return err
			}
		} else {
			err = os.Symlink(file.contents, filepath.Join(prefix, file.path))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func assertIntArraysAreEqual(t *testing.T, expected []int, result []int) bool {
	for _, ex := range expected {
		found := false
		for _, res := range result {
			if ex == res {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected array %v but got %v", expected, result)
			return false
		}
	}
	return true
}

func assertUint64ArraysAreEqual(t *testing.T, expected []uint64, result []uint64) bool {
	for _, ex := range expected {
		found := false
		for _, res := range result {
			if ex == res {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected array %v but got %v", expected, result)
			return false
		}
	}
	return true
}

func TestFindPidsByCmdlineGrep(t *testing.T) {
	logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{})
	proc := []testProcFile{
		{path: "/proc/1/cmdline", contents: "/sbin/init"},
		{path: "/proc/1/cgroup", contents: ""},
		{path: "/proc/16/cmdline", contents: ""},
		{path: "/proc/18/cgroup", contents: ""},
		{path: "/proc/766/cmdline", contents: "nginx: master process /usr/sbin/nginx"},
		{path: "/proc/768/cmdline", contents: "nginx: worker process"},
		{path: "/proc/769/cmdline", contents: "nginx: cache manager process"},
		{path: "/proc/1091/cmdline", contents: "/home/sipscan/env/bin/python\000/home/sipscan/env/bin/gunicorn\000-w\0002\000-b\000127.0.0.1:8001\000sipscan.sipscan:app"},
		{path: "/proc/9316/cmdline", contents: "/home/packetbeat/env/bin/python\000/home/packetbeat/env/bin/gunicorn\000-w\0002\000-b\000127.0.0.1:8002\000monar:app"},
	}

	// Create fake proc file system
	pathPrefix, err := ioutil.TempDir("/tmp", "")
	if err != nil {
		t.Error("TempDir failed:", err)
		return
	}
	defer os.RemoveAll(pathPrefix)

	err = createFakeDirectoryStructure(pathPrefix, proc)
	if err != nil {
		t.Error("CreateFakeDirectoryStructure failed:", err)
		return
	}

	pids, err := findPidsByCmdlineGrep(pathPrefix, "nginx")
	if err != nil {
		t.Error("FindPidsByCmdline:", err)
		return
	}

	assertIntArraysAreEqual(t, []int{766, 768, 769}, pids)
}

func TestRefreshPids(t *testing.T) {

	proc := []testProcFile{
		{path: "/proc/1/cmdline", contents: "/sbin/init"},
		{path: "/proc/1/cgroup", contents: ""},
		{path: "/proc/16/cmdline", contents: ""},
		{path: "/proc/18/cgroup", contents: ""},
		{path: "/proc/766/cmdline", contents: "nginx: master process /usr/sbin/nginx"},
		{path: "/proc/768/cmdline", contents: "nginx: worker process"},
		{path: "/proc/769/cmdline", contents: "nginx: cache manager process"},
	}

	// Create fake proc file system
	pathPrefix, err := ioutil.TempDir("/tmp", "")
	if err != nil {
		t.Error("TempDir failed:", err)
		return
	}
	defer os.RemoveAll(pathPrefix)

	err = createFakeDirectoryStructure(pathPrefix, proc)
	if err != nil {
		t.Error("CreateFakeDirectoryStructure failed:", err)
		return
	}

	testSignals := make(chan bool)
	procs := ProcessesWatcher{
		procPrefix:  pathPrefix,
		testSignals: &testSignals,
	}
	ch := make(chan time.Time)

	p, err := newProcess(&procs, "nginx", "nginx", (<-chan time.Time)(ch))
	if err != nil {
		t.Fatalf("NewProcess: %s", err)
	}

	ch <- time.Now()
	<-testSignals

	t.Logf("p and p.Pids: %p %v", p, p.pids)
	assertIntArraysAreEqual(t, []int{766, 768, 769}, p.pids)

	// Add new process
	os.MkdirAll(filepath.Join(pathPrefix, "/proc/780"), 0755)
	ioutil.WriteFile(filepath.Join(pathPrefix, "/proc/780/cmdline"),
		[]byte("nginx whatever"), 0644)

	ch <- time.Now()
	<-testSignals

	assertIntArraysAreEqual(t, []int{766, 768, 769, 780}, p.pids)
}

func TestFindSocketsOfPid(t *testing.T) {
	logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{})

	proc := []testProcFile{
		{path: "/proc/766/fd/0", isLink: true, contents: "/dev/null"},
		{path: "/proc/766/fd/1", isLink: true, contents: "/dev/null"},
		{path: "/proc/766/fd/10", isLink: true, contents: "/var/log/nginx/packetbeat.error.log"},
		{path: "/proc/766/fd/11", isLink: true, contents: "/var/log/nginx/sipscan.access.log"},
		{path: "/proc/766/fd/12", isLink: true, contents: "/var/log/nginx/sipscan.error.log"},
		{path: "/proc/766/fd/13", isLink: true, contents: "/var/log/nginx/localhost.access.log"},
		{path: "/proc/766/fd/14", isLink: true, contents: "socket:[7619]"},
		{path: "/proc/766/fd/15", isLink: true, contents: "socket:[7620]"},
		{path: "/proc/766/fd/5", isLink: true, contents: "/var/log/nginx/access.log"},
	}

	// Create fake proc file system
	pathPrefix, err := ioutil.TempDir("/tmp", "")
	if err != nil {
		t.Error("TempDir failed:", err)
		return
	}
	defer os.RemoveAll(pathPrefix)

	err = createFakeDirectoryStructure(pathPrefix, proc)
	if err != nil {
		t.Error("CreateFakeDirectoryStructure failed:", err)
		return
	}

	inodes, err := findSocketsOfPid(pathPrefix, 766)
	if err != nil {
		t.Fatalf("FindSocketsOfPid: %s", err)
	}

	assertUint64ArraysAreEqual(t, []uint64{7619, 7620}, inodes)
}

func TestParse_Proc_Net_Tcp(t *testing.T) {
	file, err := os.Open("../tests/files/proc_net_tcp.txt")
	if err != nil {
		t.Fatalf("Opening ../tests/files/proc_net_tcp.txt: %s", err)
	}
	socketInfo, err := parseProcNetTCP(file, false)
	if err != nil {
		t.Fatalf("Parse_Proc_Net_Tcp: %s", err)
	}
	if len(socketInfo) != 32 {
		t.Error("expected socket information on 32 sockets but got", len(socketInfo))
	}
	if socketInfo[31].srcIP.String() != "192.168.2.243" {
		t.Error("Failed to parse source IP address 192.168.2.243")
	}
	if socketInfo[31].srcPort != 41622 {
		t.Error("Failed to parse source port 41622")
	}
}

func TestParse_Proc_Net_Tcp6(t *testing.T) {
	file, err := os.Open("../tests/files/proc_net_tcp6.txt")
	if err != nil {
		t.Fatalf("Opening ../tests/files/proc_net_tcp6.txt: %s", err)
	}
	socketInfo, err := parseProcNetTCP(file, true)
	if err != nil {
		t.Fatalf("Parse_Proc_Net_Tcp: %s", err)
	}
	if len(socketInfo) != 6 {
		t.Error("expected socket information on 6 sockets but got", len(socketInfo))
	}
	if socketInfo[5].srcIP.String() != "::" {
		t.Error("Failed to parse source IP address ::, got instead", socketInfo[5].srcIP.String())
	}
	// TODO add an example of a 'real' IPv6 address
	if socketInfo[5].srcPort != 59497 {
		t.Error("Failed to parse source port 59497, got instead", socketInfo[5].srcPort)
	}
}
