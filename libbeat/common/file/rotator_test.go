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

package file_test

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v8/libbeat/common/file"
	"github.com/elastic/beats/v8/libbeat/logp"
)

const logMessage = "Test file rotator.\n"

func TestFileRotator(t *testing.T) {
	logp.TestingSetup()

	dir := t.TempDir()
	logname := "sample"
	c := &testClock{time.Date(2021, 11, 11, 0, 0, 0, 0, time.Local)}

	filename := filepath.Join(dir, logname)
	r, err := file.NewFileRotator(filename,
		file.MaxBackups(2),
		file.WithLogger(logp.NewLogger("rotator").With(logp.Namespace("rotator"))),
		file.WithClock(c),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	firstFile := fmt.Sprintf("%s-%s.ndjson", logname, c.Now().Format(file.DateFormat))

	WriteMsg(t, r)
	AssertDirContents(t, dir, firstFile)

	c.time = time.Date(2021, 11, 12, 0, 0, 0, 0, time.Local)

	Rotate(t, r)
	AssertDirContents(t, dir, firstFile)

	WriteMsg(t, r)

	secondFile := fmt.Sprintf("%s-%s.ndjson", logname, c.Now().Format(file.DateFormat))
	AssertDirContents(t, dir, firstFile, secondFile)

	c.time = time.Date(2021, 11, 13, 0, 0, 0, 0, time.Local)

	Rotate(t, r)
	AssertDirContents(t, dir, firstFile, secondFile)

	WriteMsg(t, r)
	thirdFile := fmt.Sprintf("%s-%s.ndjson", logname, c.Now().Format(file.DateFormat))
	AssertDirContents(t, dir, firstFile, secondFile, thirdFile)

	c.time = time.Date(2021, 11, 14, 0, 0, 0, 0, time.Local)
	Rotate(t, r)
	AssertDirContents(t, dir, secondFile, thirdFile)

	c.time = time.Date(2021, 11, 15, 0, 0, 0, 0, time.Local)
	Rotate(t, r)
	AssertDirContents(t, dir, secondFile, thirdFile)
}

func TestFileRotatorConcurrently(t *testing.T) {
	dir := t.TempDir()

	filename := filepath.Join(dir, "sample")
	r, err := file.NewFileRotator(filename, file.MaxBackups(2))
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	var wg sync.WaitGroup
	wg.Add(1000)
	for i := 0; i < 1000; i++ {
		go func() {
			defer wg.Done()
			WriteMsg(t, r)
		}()
	}
	wg.Wait()
}

func TestDailyRotation(t *testing.T) {
	dir := t.TempDir()

	logname := "daily"
	yesterday := time.Now().AddDate(0, 0, -1).Format(file.DateFormat)
	twoDaysAgo := time.Now().AddDate(0, 0, -2).Format(file.DateFormat)

	// seed directory with existing log files
	files := []string{
		logname + "-" + yesterday + "-1.ndjson",
		logname + "-" + yesterday + "-2.ndjson",
		logname + "-" + yesterday + "-3.ndjson",
		logname + "-" + yesterday + "-4.ndjson",
		logname + "-" + yesterday + "-5.ndjson",
		logname + "-" + yesterday + "-6.ndjson",
		logname + "-" + yesterday + "-7.ndjson",
		logname + "-" + yesterday + "-8.ndjson",
		logname + "-" + yesterday + "-9.ndjson",
		logname + "-" + yesterday + "-10.ndjson",
		logname + "-" + yesterday + "-11.ndjson",
		logname + "-" + yesterday + "-12.ndjson",
		logname + "-" + yesterday + "-13.ndjson",
		logname + "-" + twoDaysAgo + "-1.ndjson",
		logname + "-" + twoDaysAgo + "-2.ndjson",
		logname + "-" + twoDaysAgo + "-3.ndjson",
	}

	for _, f := range files {
		CreateFile(t, filepath.Join(dir, f))
	}

	maxSizeBytes := uint(500)
	filename := filepath.Join(dir, logname)
	r, err := file.NewFileRotator(filename, file.MaxBackups(2), file.Interval(24*time.Hour), file.MaxSizeBytes(maxSizeBytes))
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	// The backups exceeding the max of 2 aren't deleted until the first rotation.
	AssertDirContents(t, dir, files...)

	Rotate(t, r)

	AssertDirContents(t, dir, logname+"-"+yesterday+"-12.ndjson", logname+"-"+yesterday+"-13.ndjson")

	WriteMsg(t, r)

	today := time.Now().Format(file.DateFormat)
	AssertDirContents(t, dir, logname+"-"+yesterday+"-12.ndjson", logname+"-"+yesterday+"-13.ndjson", logname+"-"+today+".ndjson")

	Rotate(t, r)

	AssertDirContents(t, dir, logname+"-"+yesterday+"-13.ndjson", logname+"-"+today+".ndjson")

	WriteMsg(t, r)

	AssertDirContents(t, dir, logname+"-"+yesterday+"-13.ndjson", logname+"-"+today+".ndjson", logname+"-"+today+"-1.ndjson")

	for i := 0; i < (int(maxSizeBytes)/len(logMessage))+1; i++ {
		WriteMsg(t, r)
	}

	AssertDirContents(t, dir, logname+"-"+today+"-1.ndjson", logname+"-"+today+"-2.ndjson", logname+"-"+today+"-3.ndjson")
}

// Tests the FileConfig.RotateOnStartup parameter
func TestRotateOnStartup(t *testing.T) {
	dir := t.TempDir()

	logname := "rotate_on_open"
	c := &testClock{time.Date(2021, 11, 11, 0, 0, 0, 0, time.Local)}
	firstFile := fmt.Sprintf("%s-%s.ndjson", logname, c.Now().Format(file.DateFormat))
	filename := filepath.Join(dir, firstFile)

	// Create an existing log file with this name.
	CreateFile(t, filename)
	AssertDirContents(t, dir, firstFile)

	r, err := file.NewFileRotator(filepath.Join(dir, logname), file.RotateOnStartup(false), file.WithClock(c))
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	WriteMsg(t, r)

	// The line should have been appended to the existing file without rotation.
	AssertDirContents(t, dir, firstFile)

	// Close the first rotator early (the deferred close will be a no-op if
	// we haven't hit an error by now), so it can't interfere with the second one.
	r.Close()

	// Create a second rotator with the default setting of rotateOnStartup=true
	c = &testClock{time.Date(2021, 11, 12, 0, 0, 0, 0, time.Local)}
	r, err = file.NewFileRotator(filepath.Join(dir, logname), file.WithClock(c))
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	// The directory contents shouldn't change until the first Write.
	AssertDirContents(t, dir, firstFile)

	secondFile := fmt.Sprintf("%s-%s.ndjson", logname, c.Now().Format(file.DateFormat))

	WriteMsg(t, r)
	AssertDirContents(t, dir, firstFile, secondFile)
}

func TestRotate(t *testing.T) {
	dir := t.TempDir()

	logname := "beatname"
	filename := filepath.Join(dir, logname)

	c := &testClock{time.Date(2021, 11, 11, 0, 0, 0, 0, time.Local)}
	r, err := file.NewFileRotator(filename, file.MaxBackups(1), file.WithClock(c))
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	WriteMsg(t, r)

	firstFile := fmt.Sprintf("%s-%s.ndjson", logname, c.Now().Format(file.DateFormat))
	AssertDirContents(t, dir, firstFile)

	c.time = time.Date(2021, 11, 13, 0, 0, 0, 0, time.Local)
	secondFile := fmt.Sprintf("%s-%s.ndjson", logname, c.Now().Format(file.DateFormat))

	Rotate(t, r)
	WriteMsg(t, r)

	AssertDirContents(t, dir, firstFile, secondFile)

	c.time = time.Date(2021, 11, 15, 0, 0, 0, 0, time.Local)
	thirdFile := fmt.Sprintf("%s-%s.ndjson", logname, c.Now().Format(file.DateFormat))

	Rotate(t, r)
	WriteMsg(t, r)

	AssertDirContents(t, dir, secondFile, thirdFile)
}

func CreateFile(t *testing.T, filename string) {
	t.Helper()
	f, err := os.Create(filename)
	if err != nil {
		t.Fatal(err)
	}
	err = f.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func AssertDirContents(t *testing.T, dir string, files ...string) {
	t.Helper()

	f, err := os.Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	names, err := f.Readdirnames(-1)
	if err != nil {
		t.Fatal(err)
	}

	assert.ElementsMatch(t, files, names)
}

func WriteMsg(t *testing.T, r *file.Rotator) {
	t.Helper()

	n, err := r.Write([]byte(logMessage))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, len(logMessage), n)
}

func Rotate(t *testing.T, r *file.Rotator) {
	t.Helper()

	if err := r.Rotate(); err != nil {
		t.Fatal(err)
	}
}

type testClock struct {
	time time.Time
}

func (t testClock) Now() time.Time {
	return t.time
}
