// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux

package login

import (
	"encoding/binary"
	"github.com/elastic/beats/v7/auditbeat/ab"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/auditbeat/core"
	abtest "github.com/elastic/beats/v7/auditbeat/testing"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestData(t *testing.T) {
	if byteOrder != binary.LittleEndian {
		t.Skip("Test only works on little-endian systems - skipping.")
	}

	defer abtest.SetupDataDir(t)()

	config := getBaseConfig()
	config["login.wtmp_file_pattern"] = "./testdata/wtmp"
	config["login.btmp_file_pattern"] = ""
	f := mbtest.NewReportingMetricSetV2WithRegistry(t, config, ab.Registry)
	defer f.(*MetricSet).utmpReader.bucket.DeleteBucket()

	events, errs := mbtest.ReportingFetchV2(f)
	if len(errs) > 0 {
		t.Fatalf("received error: %+v", errs[0])
	}

	if len(events) == 0 {
		t.Fatal("no events were generated")
	} else if len(events) != 1 {
		t.Fatalf("only one event expected, got %d", len(events))
	}

	events[0].RootFields.Put("event.origin", "/var/log/wtmp")
	fullEvent := mbtest.StandardizeEvent(f, events[0], core.AddDatasetToEvent)
	mbtest.WriteEventToDataJSON(t, fullEvent, "")
}

func TestWtmp(t *testing.T) {
	if byteOrder != binary.LittleEndian {
		t.Skip("Test only works on little-endian systems - skipping.")
	}

	defer abtest.SetupDataDir(t)()

	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	wtmpFilepath := filepath.Join(dir, "wtmp")

	config := getBaseConfig()
	config["login.wtmp_file_pattern"] = wtmpFilepath
	config["login.btmp_file_pattern"] = ""
	f := mbtest.NewReportingMetricSetV2(t, config)
	defer f.(*MetricSet).utmpReader.bucket.DeleteBucket()

	events, errs := mbtest.ReportingFetchV2(f)
	if len(errs) > 0 {
		t.Fatalf("received error: %+v", errs[0])
	}

	if len(events) == 0 {
		t.Fatal("no events were generated")
	} else if len(events) != 1 {
		t.Fatalf("only one event expected, got %d", len(events))
	}

	// utmpdump: [7] [14962] [ts/2] [vagrant ] [pts/2       ] [10.0.2.2            ] [10.0.2.2       ] [2019-01-24T09:51:51,367964+00:00]
	checkFieldValue(t, events[0].RootFields, "event.kind", "event")
	checkFieldValue(t, events[0].RootFields, "event.category", []string{"authentication"})
	checkFieldValue(t, events[0].RootFields, "event.type", []string{"start", "authentication_success"})
	checkFieldValue(t, events[0].RootFields, "event.action", "user_login")
	checkFieldValue(t, events[0].RootFields, "event.outcome", "success")
	checkFieldValue(t, events[0].RootFields, "process.pid", 14962)
	checkFieldValue(t, events[0].RootFields, "source.ip", "10.0.2.2")
	checkFieldValue(t, events[0].RootFields, "user.name", "vagrant")
	checkFieldValue(t, events[0].RootFields, "user.terminal", "pts/2")
	assert.True(t, events[0].Timestamp.Equal(time.Date(2019, 1, 24, 9, 51, 51, 367964000, time.UTC)),
		"Timestamp is not equal: %+v", events[0].Timestamp)

	// Append logout event to wtmp file and check that it's read
	wtmpFile, err := os.OpenFile(wtmpFilepath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("error opening %v: %v", wtmpFilepath, err)
	}

	wtmpFileInfo, err := os.Stat(wtmpFilepath)
	if err != nil {
		t.Fatalf("error performing stat on %v: %v", wtmpFilepath, err)
	}

	size := wtmpFileInfo.Size()

	loginUtmp := utmpC{
		Type: DEAD_PROCESS,
	}
	copy(loginUtmp.Device[:], "pts/2")

	err = binary.Write(wtmpFile, byteOrder, loginUtmp)
	if err != nil {
		t.Fatalf("error writing to %v: %v", wtmpFilepath, err)
	}

	events, errs = mbtest.ReportingFetchV2(f)
	if len(errs) > 0 {
		t.Fatalf("received error: %+v", errs[0])
	}

	if len(events) == 0 {
		t.Fatal("no events were generated")
	} else if len(events) != 1 {
		t.Fatalf("only one event expected, got %d: %v", len(events), events)
	}

	checkFieldValue(t, events[0].RootFields, "event.kind", "event")
	checkFieldValue(t, events[0].RootFields, "event.category", []string{"authentication"})
	checkFieldValue(t, events[0].RootFields, "event.type", []string{"end"})
	checkFieldValue(t, events[0].RootFields, "event.action", "user_logout")
	checkFieldValue(t, events[0].RootFields, "process.pid", 14962)
	checkFieldValue(t, events[0].RootFields, "source.ip", "10.0.2.2")
	checkFieldValue(t, events[0].RootFields, "related.ip", []string{"10.0.2.2"})
	checkFieldValue(t, events[0].RootFields, "user.name", "vagrant")
	checkFieldValue(t, events[0].RootFields, "related.user", []string{"vagrant"})
	checkFieldValue(t, events[0].RootFields, "user.terminal", "pts/2")

	// We truncate to the previous size to force a full re-read, simulating an inode reuse.
	if err := wtmpFile.Truncate(size); err != nil {
		t.Fatalf("error truncating %v: %v", wtmpFilepath, err)
	}

	events, errs = mbtest.ReportingFetchV2(f)
	if len(errs) > 0 {
		t.Fatalf("received error: %+v", errs[0])
	}

	if len(events) == 0 {
		t.Fatal("no events were generated")
	} else if len(events) != 1 {
		t.Fatalf("only one event expected, got %d", len(events))
	}

	// utmpdump: [7] [14962] [ts/2] [vagrant ] [pts/2       ] [10.0.2.2            ] [10.0.2.2       ] [2019-01-24T09:51:51,367964+00:00]
	checkFieldValue(t, events[0].RootFields, "event.kind", "event")
	checkFieldValue(t, events[0].RootFields, "event.category", []string{"authentication"})
	checkFieldValue(t, events[0].RootFields, "event.type", []string{"start", "authentication_success"})
	checkFieldValue(t, events[0].RootFields, "event.action", "user_login")
	checkFieldValue(t, events[0].RootFields, "event.outcome", "success")
	checkFieldValue(t, events[0].RootFields, "process.pid", 14962)
	checkFieldValue(t, events[0].RootFields, "source.ip", "10.0.2.2")
	checkFieldValue(t, events[0].RootFields, "user.name", "vagrant")
	checkFieldValue(t, events[0].RootFields, "user.terminal", "pts/2")
	assert.True(t, events[0].Timestamp.Equal(time.Date(2019, 1, 24, 9, 51, 51, 367964000, time.UTC)),
		"Timestamp is not equal: %+v", events[0].Timestamp)
}

func TestBtmp(t *testing.T) {
	if byteOrder != binary.LittleEndian {
		t.Skip("Test only works on little-endian systems - skipping.")
	}

	defer abtest.SetupDataDir(t)()

	config := getBaseConfig()
	config["login.wtmp_file_pattern"] = ""
	config["login.btmp_file_pattern"] = "./testdata/btmp*"
	f := mbtest.NewReportingMetricSetV2(t, config)
	defer f.(*MetricSet).utmpReader.bucket.DeleteBucket()

	events, errs := mbtest.ReportingFetchV2(f)
	if len(errs) > 0 {
		t.Fatalf("received error: %+v", errs[0])
	}

	if len(events) == 0 {
		t.Fatal("no events were generated")
	} else if len(events) != 4 {
		t.Fatalf("expected 4 events, got %d", len(events))
	}

	// utmpdump: [6] [03307] [    ] [root    ] [ssh:notty   ] [10.0.2.2            ] [10.0.2.2       ] [2019-02-20T17:42:26,000000+0000]
	checkFieldValue(t, events[0].RootFields, "event.kind", "event")
	checkFieldValue(t, events[0].RootFields, "event.category", []string{"authentication"})
	checkFieldValue(t, events[0].RootFields, "event.type", []string{"start", "authentication_failure"})
	checkFieldValue(t, events[0].RootFields, "event.action", "user_login")
	checkFieldValue(t, events[0].RootFields, "event.outcome", "failure")
	checkFieldValue(t, events[0].RootFields, "process.pid", 3307)
	checkFieldValue(t, events[0].RootFields, "source.ip", "10.0.2.2")
	checkFieldValue(t, events[0].RootFields, "user.id", 0)
	checkFieldValue(t, events[0].RootFields, "user.name", "root")
	checkFieldValue(t, events[0].RootFields, "user.terminal", "ssh:notty")
	assert.True(t, events[0].Timestamp.Equal(time.Date(2019, 2, 20, 17, 42, 26, 0, time.UTC)),
		"Timestamp is not equal: %+v", events[0].Timestamp)

	// The second UTMP entry in the btmp test file is a duplicate of the first, this is what Ubuntu 18.04 generates.
	// utmpdump: [6] [03307] [    ] [root    ] [ssh:notty   ] [10.0.2.2            ] [10.0.2.2       ] [2019-02-20T17:42:26,000000+0000]
	checkFieldValue(t, events[1].RootFields, "event.kind", "event")
	checkFieldValue(t, events[0].RootFields, "event.category", []string{"authentication"})
	checkFieldValue(t, events[0].RootFields, "event.type", []string{"start", "authentication_failure"})
	checkFieldValue(t, events[1].RootFields, "event.action", "user_login")
	checkFieldValue(t, events[1].RootFields, "event.outcome", "failure")
	checkFieldValue(t, events[1].RootFields, "process.pid", 3307)
	checkFieldValue(t, events[1].RootFields, "source.ip", "10.0.2.2")
	checkFieldValue(t, events[1].RootFields, "user.id", 0)
	checkFieldValue(t, events[1].RootFields, "user.name", "root")
	checkFieldValue(t, events[1].RootFields, "user.terminal", "ssh:notty")
	assert.True(t, events[1].Timestamp.Equal(time.Date(2019, 2, 20, 17, 42, 26, 0, time.UTC)),
		"Timestamp is not equal: %+v", events[1].Timestamp)

	// utmpdump: [7] [03788] [/0  ] [elastic ] [pts/0       ] [                    ] [0.0.0.0        ] [2019-02-20T17:45:08,447344+0000]
	checkFieldValue(t, events[2].RootFields, "event.kind", "event")
	checkFieldValue(t, events[0].RootFields, "event.category", []string{"authentication"})
	checkFieldValue(t, events[0].RootFields, "event.type", []string{"start", "authentication_failure"})
	checkFieldValue(t, events[2].RootFields, "event.action", "user_login")
	checkFieldValue(t, events[2].RootFields, "event.outcome", "failure")
	checkFieldValue(t, events[2].RootFields, "process.pid", 3788)
	checkFieldValue(t, events[2].RootFields, "source.ip", "0.0.0.0")
	checkFieldValue(t, events[2].RootFields, "user.name", "elastic")
	checkFieldValue(t, events[2].RootFields, "user.terminal", "pts/0")
	assert.True(t, events[2].Timestamp.Equal(time.Date(2019, 2, 20, 17, 45, 8, 447344000, time.UTC)),
		"Timestamp is not equal: %+v", events[2].Timestamp)

	// utmpdump: [7] [03788] [/0  ] [UNKNOWN ] [pts/0       ] [                    ] [0.0.0.0        ] [2019-02-20T17:45:15,765318+0000]
	checkFieldValue(t, events[3].RootFields, "event.kind", "event")
	checkFieldValue(t, events[0].RootFields, "event.category", []string{"authentication"})
	checkFieldValue(t, events[0].RootFields, "event.type", []string{"start", "authentication_failure"})
	checkFieldValue(t, events[3].RootFields, "event.action", "user_login")
	checkFieldValue(t, events[3].RootFields, "event.outcome", "failure")
	checkFieldValue(t, events[3].RootFields, "process.pid", 3788)
	checkFieldValue(t, events[3].RootFields, "source.ip", "0.0.0.0")
	contains, err := events[3].RootFields.HasKey("user.id")
	if assert.NoError(t, err) {
		assert.False(t, contains)
	}
	checkFieldValue(t, events[3].RootFields, "user.name", "UNKNOWN")
	checkFieldValue(t, events[3].RootFields, "user.terminal", "pts/0")
	assert.True(t, events[3].Timestamp.Equal(time.Date(2019, 2, 20, 17, 45, 15, 765318000, time.UTC)),
		"Timestamp is not equal: %+v", events[3].Timestamp)
}

func checkFieldValue(t *testing.T, mapstr mapstr.M, fieldName string, fieldValue interface{}) {
	value, err := mapstr.GetValue(fieldName)
	if assert.NoError(t, err) {
		switch v := value.(type) {
		case *net.IP:
			assert.Equal(t, fieldValue, v.String())
		default:
			assert.Equal(t, fieldValue, v)
		}
	}
}

func getBaseConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":   system.ModuleName,
		"datasets": []string{"login"},
	}
}

// setupTestDir creates a temporary directory, copies the test files into it,
// and returns the path.
func setupTestDir(t *testing.T) string {
	tmp, err := ioutil.TempDir("", "auditbeat-login-test-dir")
	if err != nil {
		t.Fatal("failed to create temp dir")
	}

	copyDir(t, "./testdata", tmp)

	return tmp
}

func copyDir(t *testing.T, src, dst string) {
	files, err := ioutil.ReadDir(src)
	if err != nil {
		t.Fatalf("failed to read %v", src)
	}

	for _, file := range files {
		srcFile := filepath.Join(src, file.Name())
		dstFile := filepath.Join(dst, file.Name())
		copyFile(t, srcFile, dstFile)
	}
}

func copyFile(t *testing.T, src, dst string) {
	in, err := os.Open(src)
	if err != nil {
		t.Fatalf("failed to open %v", src)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		t.Fatalf("failed to open %v", dst)
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		t.Fatalf("failed to copy %v to %v", src, dst)
	}
}
