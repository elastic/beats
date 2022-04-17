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

//go:build linux
// +build linux

package procs

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/menderesk/beats/v7/libbeat/logp"
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
		err = os.MkdirAll(filepath.Join(prefix, dir), 0o755)
		if err != nil {
			return err
		}

		if !file.isLink {
			err = ioutil.WriteFile(filepath.Join(prefix, file.path),
				[]byte(file.contents), 0o644)
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

func TestFindSocketsOfPid(t *testing.T) {
	logp.TestingSetup()

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
	pathPrefix, err := ioutil.TempDir("", "find-sockets")
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
	socketInfo, err := socketsFromProc("../tests/files/proc_net_tcp.txt", false)
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
	socketInfo, err := socketsFromProc("../tests/files/proc_net_tcp6.txt", true)
	if err != nil {
		t.Fatalf("Parse_Proc_Net_Tcp: %s", err)
	}
	if len(socketInfo) != 6 {
		t.Error("expected socket information on 6 sockets but got", len(socketInfo))
	}
	if socketInfo[5].srcIP.String() != "::" {
		t.Error("Failed to parse source IP address ::, got instead", socketInfo[5].srcIP.String())
	}
	if socketInfo[5].srcPort != 59497 {
		t.Error("Failed to parse source port 59497, got instead", socketInfo[5].srcPort)
	}
	if socketInfo[4].srcIP.String() != "2001:db8::123:ffff:89ab:cdef" {
		t.Error("Failed to parse source IP address 2001:db8::123:ffff:89ab:cdef, got instead", socketInfo[4].srcIP.String())
	}
}
