// +build !integration

package procs

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/logp"
)

type TestProcFile struct {
	Path     string
	Contents string
	IsLink   bool
}

func CreateFakeDirectoryStructure(prefix string, files []TestProcFile) error {

	var err error
	for _, file := range files {
		dir := filepath.Dir(file.Path)
		err = os.MkdirAll(filepath.Join(prefix, dir), 0755)
		if err != nil {
			return err
		}

		if !file.IsLink {
			err = ioutil.WriteFile(filepath.Join(prefix, file.Path),
				[]byte(file.Contents), 0644)
			if err != nil {
				return err
			}
		} else {
			err = os.Symlink(file.Contents, filepath.Join(prefix, file.Path))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func AssertIntArraysAreEqual(t *testing.T, expected []int, result []int) bool {
	for _, ex := range expected {
		found := false
		for _, res := range result {
			if ex == res {
				found = true
				break
			}
		}
		if !found {
			t.Error(fmt.Sprintf("Expected array %v but got %v", expected, result))
			return false
		}
	}
	return true
}

func AssertInt64ArraysAreEqual(t *testing.T, expected []int64, result []int64) bool {
	for _, ex := range expected {
		found := false
		for _, res := range result {
			if ex == res {
				found = true
				break
			}
		}
		if !found {
			t.Error(fmt.Sprintf("Expected array %v but got %v", expected, result))
			return false
		}
	}
	return true
}

func TestFindPidsByCmdlineGrep(t *testing.T) {
	logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{})
	proc := []TestProcFile{
		{Path: "/proc/1/cmdline", Contents: "/sbin/init"},
		{Path: "/proc/1/cgroup", Contents: ""},
		{Path: "/proc/16/cmdline", Contents: ""},
		{Path: "/proc/18/cgroup", Contents: ""},
		{Path: "/proc/766/cmdline", Contents: "nginx: master process /usr/sbin/nginx"},
		{Path: "/proc/768/cmdline", Contents: "nginx: worker process"},
		{Path: "/proc/769/cmdline", Contents: "nginx: cache manager process"},
		{Path: "/proc/1091/cmdline", Contents: "/home/sipscan/env/bin/python\000/home/sipscan/env/bin/gunicorn\000-w\0002\000-b\000127.0.0.1:8001\000sipscan.sipscan:app"},
		{Path: "/proc/9316/cmdline", Contents: "/home/packetbeat/env/bin/python\000/home/packetbeat/env/bin/gunicorn\000-w\0002\000-b\000127.0.0.1:8002\000monar:app"},
	}

	// Create fake proc file system
	path_prefix, err := ioutil.TempDir("/tmp", "")
	if err != nil {
		t.Error("TempDir failed:", err)
		return
	}
	defer os.RemoveAll(path_prefix)

	err = CreateFakeDirectoryStructure(path_prefix, proc)
	if err != nil {
		t.Error("CreateFakeDirectoryStructure failed:", err)
		return
	}

	pids, err := FindPidsByCmdlineGrep(path_prefix, "nginx")
	if err != nil {
		t.Error("FindPidsByCmdline:", err)
		return
	}

	AssertIntArraysAreEqual(t, []int{766, 768, 769}, pids)
}

func TestRefreshPids(t *testing.T) {

	proc := []TestProcFile{
		{Path: "/proc/1/cmdline", Contents: "/sbin/init"},
		{Path: "/proc/1/cgroup", Contents: ""},
		{Path: "/proc/16/cmdline", Contents: ""},
		{Path: "/proc/18/cgroup", Contents: ""},
		{Path: "/proc/766/cmdline", Contents: "nginx: master process /usr/sbin/nginx"},
		{Path: "/proc/768/cmdline", Contents: "nginx: worker process"},
		{Path: "/proc/769/cmdline", Contents: "nginx: cache manager process"},
	}

	// Create fake proc file system
	path_prefix, err := ioutil.TempDir("/tmp", "")
	if err != nil {
		t.Error("TempDir failed:", err)
		return
	}
	defer os.RemoveAll(path_prefix)

	err = CreateFakeDirectoryStructure(path_prefix, proc)
	if err != nil {
		t.Error("CreateFakeDirectoryStructure failed:", err)
		return
	}

	testSignals := make(chan bool)
	var procs ProcessesWatcher = ProcessesWatcher{proc_prefix: path_prefix,
		TestSignals: &testSignals}
	var ch chan time.Time = make(chan time.Time)

	p, err := NewProcess(&procs, "nginx", "nginx", (<-chan time.Time)(ch))
	if err != nil {
		t.Fatalf("NewProcess: %s", err)
	}

	ch <- time.Now()
	<-testSignals

	t.Logf("p and p.Pids: %p %v", p, p.Pids)
	AssertIntArraysAreEqual(t, []int{766, 768, 769}, p.Pids)

	// Add new process
	os.MkdirAll(filepath.Join(path_prefix, "/proc/780"), 0755)
	ioutil.WriteFile(filepath.Join(path_prefix, "/proc/780/cmdline"),
		[]byte("nginx whatever"), 0644)

	ch <- time.Now()
	<-testSignals

	AssertIntArraysAreEqual(t, []int{766, 768, 769, 780}, p.Pids)
}

func TestFindSocketsOfPid(t *testing.T) {
	logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{})

	proc := []TestProcFile{
		{Path: "/proc/766/fd/0", IsLink: true, Contents: "/dev/null"},
		{Path: "/proc/766/fd/1", IsLink: true, Contents: "/dev/null"},
		{Path: "/proc/766/fd/10", IsLink: true, Contents: "/var/log/nginx/packetbeat.error.log"},
		{Path: "/proc/766/fd/11", IsLink: true, Contents: "/var/log/nginx/sipscan.access.log"},
		{Path: "/proc/766/fd/12", IsLink: true, Contents: "/var/log/nginx/sipscan.error.log"},
		{Path: "/proc/766/fd/13", IsLink: true, Contents: "/var/log/nginx/localhost.access.log"},
		{Path: "/proc/766/fd/14", IsLink: true, Contents: "socket:[7619]"},
		{Path: "/proc/766/fd/15", IsLink: true, Contents: "socket:[7620]"},
		{Path: "/proc/766/fd/5", IsLink: true, Contents: "/var/log/nginx/access.log"},
	}

	// Create fake proc file system
	path_prefix, err := ioutil.TempDir("/tmp", "")
	if err != nil {
		t.Error("TempDir failed:", err)
		return
	}
	defer os.RemoveAll(path_prefix)

	err = CreateFakeDirectoryStructure(path_prefix, proc)
	if err != nil {
		t.Error("CreateFakeDirectoryStructure failed:", err)
		return
	}

	inodes, err := FindSocketsOfPid(path_prefix, 766)
	if err != nil {
		t.Fatalf("FindSocketsOfPid: %s", err)
	}

	AssertInt64ArraysAreEqual(t, []int64{7619, 7620}, inodes)
}

func TestParse_Proc_Net_Tcp(t *testing.T) {
	file, err := os.Open("../tests/files/proc_net_tcp.txt")
	if err != nil {
		t.Fatalf("Opening ../tests/files/proc_net_tcp.txt: %s", err)
	}
	socketInfo, err := Parse_Proc_Net_Tcp(file, false)
	if err != nil {
		t.Fatalf("Parse_Proc_Net_Tcp: %s", err)
	}
	if len(socketInfo) != 32 {
		t.Error("expected socket information on 32 sockets but got", len(socketInfo))
	}
	if socketInfo[31].Src_ip.String() != "192.168.2.243" {
		t.Error("Failed to parse source IP address 192.168.2.243")
	}
	if socketInfo[31].Src_port != 41622 {
		t.Error("Failed to parse source port 41622")
	}
}

func TestParse_Proc_Net_Tcp6(t *testing.T) {
	file, err := os.Open("../tests/files/proc_net_tcp6.txt")
	if err != nil {
		t.Fatalf("Opening ../tests/files/proc_net_tcp6.txt: %s", err)
	}
	socketInfo, err := Parse_Proc_Net_Tcp(file, true)
	if err != nil {
		t.Fatalf("Parse_Proc_Net_Tcp: %s", err)
	}
	if len(socketInfo) != 6 {
		t.Error("expected socket information on 6 sockets but got", len(socketInfo))
	}
	if socketInfo[5].Src_ip.String() != "::" {
		t.Error("Failed to parse source IP address ::, got instead", socketInfo[5].Src_ip.String())
	}
	// TODO add an example of a 'real' IPv6 address
	if socketInfo[5].Src_port != 59497 {
		t.Error("Failed to parse source port 59497, got instead", socketInfo[5].Src_port)
	}
}
