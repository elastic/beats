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

//go:build linux && cgo && withjournald
// +build linux,cgo,withjournald

package main_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"path"
	"reflect"
	"strings"
	"testing"
	"text/template"
	"time"
)

func TestFilebeatParsers(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "filebeat-tests")
	if err != nil {
		t.Fatal("could not create temp directory:", err)
	}

	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	t.Log("temp dir:", tmpDir)

	// TODO: Question: should we compile in the temp dir?
	// This will require copying the journal file there
	compileCmd := exec.Command("go", "build", "-tags=linux,cgo,withjournald", ".")
	if err := compileCmd.Run(); err != nil {
		log.Fatal(err)
	}

	t.Cleanup(func() {
		os.Remove("filebeat")
	})

	// Generate the configuration file and save it in our
	// temporary directory
	tmpl, err := template.ParseFiles("testdata/journald_multiline_parser.tmpl")
	if err != nil {
		t.Fatal("could not parse template file", err)
	}

	configFile, err := os.Create(path.Join(tmpDir, "filebeat.yml"))
	if err != nil {
		t.Fatal("cannot create config file:", err)
	}

	if err := tmpl.Execute(configFile, struct{ TempDir string }{TempDir: tmpDir}); err != nil {
		t.Fatal("could not render/write config file:", err)
	}

	configFile.Close()

	filebeatCmd := exec.Command("./filebeat", "-c", configFile.Name(), "-e")
	// Make sure we stop Filebeat before any cleanup
	defer func() {
		if err := filebeatCmd.Process.Kill(); err != nil {
			t.Log("could not kill filebeat process: ", err)
		}
	}()

	filebeatOutput := bytes.Buffer{}
	filebeatCmd.Stderr = &filebeatOutput
	filebeatCmd.Stdout = &filebeatOutput
	if err := filebeatCmd.Start(); err != nil {
		t.Fatal("could not start Filebeat: ", err)
	}

	// Wait Filebeat do its job
	buff := bufio.NewScanner(&filebeatOutput)
	time.Sleep(2 * time.Second)
	for buff.Scan() {
		line := buff.Text()
		if strings.Contains(line, "Input journald starting") {
			time.Sleep(time.Second) // Give Filebeat a wee bit more time to run
			break
		}
	}

	outputFile, err := os.Open(path.Join(tmpDir, "output.ndjson"))
	if err != nil {
		t.Fatal("error opening output file: ", err)
	}

	scanner := bufio.NewScanner(outputFile)
	linesCount := 0
	got := []string{}
	for scanner.Scan() {
		linesCount++
		m := map[string]interface{}{}
		json.Unmarshal(scanner.Bytes(), &m)
		got = append(got, m["message"].(string))
	}

	if linesCount != 2 {
		t.Error("exprcting 2 lines, got:", linesCount)
	}

	expected := []string{
		"1st line\n2nd line\n3rd line",
		"4th line\n5th line\n6th line",
	}

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("expecting: %#v, got: %#v", expected, got)
	}
}
