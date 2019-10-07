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

package file_integrity

import (
	"crypto/sha1"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/auditbeat/core"
	"github.com/elastic/beats/auditbeat/datastore"
	abtest "github.com/elastic/beats/auditbeat/testing"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestData(t *testing.T) {
	defer abtest.SetupDataDir(t)()

	dir, err := ioutil.TempDir("", "audit-file")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	go func() {
		time.Sleep(100 * time.Millisecond)
		file := filepath.Join(dir, "file.data")
		ioutil.WriteFile(file, []byte("hello world"), 0600)
	}()

	ms := mbtest.NewPushMetricSetV2(t, getConfig(dir))
	events := mbtest.RunPushMetricSetV2(10*time.Second, 2, ms)
	for _, e := range events {
		if e.Error != nil {
			t.Fatalf("received error: %+v", e.Error)
		}
	}

	fullEvent := mbtest.StandardizeEvent(ms, events[len(events)-1], core.AddDatasetToEvent)
	mbtest.WriteEventToDataJSON(t, fullEvent, "")
}

func TestActions(t *testing.T) {
	defer abtest.SetupDataDir(t)()

	bucket, err := datastore.OpenBucket(bucketName)
	if err != nil {
		t.Fatal(err)
	}
	defer bucket.Close()

	// First directory
	dir, err := ioutil.TempDir("", "audit-file")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	dir, err = filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Second directory (to be reported with "initial_scan")
	newDir, err := ioutil.TempDir("", "audit-file-new")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(newDir)

	newDir, err = filepath.EvalSymlinks(newDir)
	if err != nil {
		t.Fatal(err)
	}

	createdFilepath := filepath.Join(dir, "created.txt")
	updatedFilepath := filepath.Join(dir, "updated.txt")
	deletedFilepath := filepath.Join(dir, "deleted.txt")

	// Add first directory to db (so that files in it are not reported with "initial_scan")
	e := &Event{
		Timestamp: time.Now().UTC(),
		Path:      dir,
		Action:    InitialScan,
	}
	if err = store(bucket, e); err != nil {
		t.Fatal(err)
	}

	// Add fake event for non-existing file to db to simulate when a file has been deleted
	deletedFileEvent := &Event{
		Timestamp: time.Now().UTC(),
		Path:      deletedFilepath,
		Action:    Created,
	}
	if err = store(bucket, deletedFileEvent); err != nil {
		t.Fatal(err)
	}

	// Insert fake file event into db to simulate when a file has changed
	digest := sha1.New().Sum([]byte("different string"))
	updatedFileEvent := &Event{
		Timestamp: time.Now().UTC(),
		Path:      updatedFilepath,
		Action:    Created,
		Hashes:    map[HashType]Digest{SHA1: digest},
	}
	if err = store(bucket, updatedFileEvent); err != nil {
		t.Fatal(err)
	}

	// Create some files in first directory
	go func() {
		ioutil.WriteFile(createdFilepath, []byte("hello world"), 0600)
		ioutil.WriteFile(updatedFilepath, []byte("hello world"), 0600)
	}()

	ms := mbtest.NewPushMetricSetV2(t, getConfig(dir, newDir))
	events := mbtest.RunPushMetricSetV2(10*time.Second, 5, ms)
	assert.Len(t, events, 5)

	for _, event := range events {
		if event.Error != nil {
			t.Fatalf("received error: %+v", event.Error)
		}

		actions, err := event.MetricSetFields.GetValue("event.action")
		path, err2 := event.MetricSetFields.GetValue("file.path")
		if assert.NoError(t, err) && assert.NoError(t, err2) {
			// Note: Actions reported for a file or directory will be different
			// depending on whether the scanner or the platform-dependent
			// filesystem event listener reported it. The subset of actions we test
			// for here should be consistent across all cases though.
			switch path.(string) {
			case newDir:
				assert.Contains(t, actions, "initial_scan")
			case dir:
				assert.Contains(t, actions, "attributes_modified")
			case deletedFilepath:
				assert.Contains(t, actions, "deleted")
			case createdFilepath:
				assert.Contains(t, actions, "created")
			case updatedFilepath:
				assert.Contains(t, actions, "updated")
			default:
				assert.Fail(t, "unexpected path", "path %v", path)
			}
		}
	}
}

func TestExcludedFiles(t *testing.T) {
	defer abtest.SetupDataDir(t)()

	bucket, err := datastore.OpenBucket(bucketName)
	if err != nil {
		t.Fatal(err)
	}
	defer bucket.Close()

	dir, err := ioutil.TempDir("", "audit-file")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	dir, err = filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}

	ms := mbtest.NewPushMetricSetV2(t, getConfig(dir))

	go func() {
		for _, f := range []string{"FILE.TXT", "FILE.TXT.SWP", "file.txt.swo", ".git/HEAD", ".gitignore"} {
			file := filepath.Join(dir, f)
			ioutil.WriteFile(file, []byte("hello world"), 0600)
		}
	}()

	events := mbtest.RunPushMetricSetV2(10*time.Second, 3, ms)
	for _, e := range events {
		if e.Error != nil {
			t.Fatalf("received error: %+v", e.Error)
		}
	}

	wanted := map[string]bool{
		dir:                              true,
		filepath.Join(dir, "FILE.TXT"):   true,
		filepath.Join(dir, ".gitignore"): true,
	}
	if !assert.Len(t, events, len(wanted)) {
		return
	}
	for _, e := range events {
		event := e.MetricSetFields
		path, err := event.GetValue("file.path")
		if assert.NoError(t, err) {
			_, ok := wanted[path.(string)]
			assert.True(t, ok)
		}
	}
}

func TestIncludedExcludedFiles(t *testing.T) {
	defer abtest.SetupDataDir(t)()

	bucket, err := datastore.OpenBucket(bucketName)
	if err != nil {
		t.Fatal(err)
	}
	defer bucket.Close()

	dir, err := ioutil.TempDir("", "audit-file")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	dir, err = filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}

	err = os.Mkdir(filepath.Join(dir, ".ssh"), 0700)
	if err != nil {
		t.Fatal(err)
	}

	config := getConfig(dir)
	config["include_files"] = []string{`\.ssh/`}
	config["recursive"] = true
	ms := mbtest.NewPushMetricSetV2(t, config)

	go func() {
		for _, f := range []string{"FILE.TXT", ".ssh/known_hosts", ".ssh/known_hosts.swp"} {
			file := filepath.Join(dir, f)
			err := ioutil.WriteFile(file, []byte("hello world"), 0600)
			if err != nil {
				t.Fatal(err)
			}
		}
	}()

	events := mbtest.RunPushMetricSetV2(10*time.Second, 3, ms)
	for _, e := range events {
		if e.Error != nil {
			t.Fatalf("received error: %+v", e.Error)
		}
	}

	wanted := map[string]bool{
		dir:                                    true,
		filepath.Join(dir, ".ssh"):             true,
		filepath.Join(dir, ".ssh/known_hosts"): true,
	}
	if !assert.Len(t, events, len(wanted)) {
		return
	}
	for _, e := range events {
		event := e.MetricSetFields
		path, err := event.GetValue("file.path")
		if assert.NoError(t, err) {
			_, ok := wanted[path.(string)]
			assert.True(t, ok)
		}
	}
}

func getConfig(path ...string) map[string]interface{} {
	return map[string]interface{}{
		"module":        "file_integrity",
		"paths":         path,
		"exclude_files": []string{`(?i)\.sw[nop]$`, `[/\\]\.git([/\\]|$)`},
	}
}
