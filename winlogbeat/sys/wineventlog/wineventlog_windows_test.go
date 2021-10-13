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

package wineventlog

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

var sysmonEvtx string

func init() {
	var err error
	sysmonEvtx, err = filepath.Abs("testdata/sysmon-9.01.evtx")
	if err != nil {
		panic(err)
	}

	if _, err = os.Lstat(sysmonEvtx); err != nil {
		panic(err)
	}
}

func TestEvtOpenLog(t *testing.T) {
	h, err := EvtOpenLog(0, sysmonEvtx, EvtOpenFilePath)
	if err != nil {
		t.Fatal(err)
	}
	defer Close(h)
}

func TestEvtQuery(t *testing.T) {
	h, err := EvtQuery(0, sysmonEvtx, "", EvtQueryFilePath)
	if err != nil {
		t.Fatal(err)
	}
	defer Close(h)
}

func TestReadEvtx(t *testing.T) {
	// Open .evtx file.
	h, err := EvtQuery(0, sysmonEvtx, "", EvtQueryFilePath|EvtQueryReverseDirection)
	if err != nil {
		t.Fatal(err)
	}
	defer Close(h)

	// Get handles to events.
	buf := make([]byte, 32*1024)
	out := new(bytes.Buffer)
	count := 0
	for {
		handles, err := EventHandles(h, 8)
		if err == ERROR_NO_MORE_ITEMS {
			t.Log(err)
			break
		}
		if err != nil {
			t.Fatal(err)
		}

		// Read events.
		for _, h := range handles {
			out.Reset()
			if err = RenderEventXML(h, buf, out); err != nil {
				t.Fatal(err)
			}
			Close(h)
			count++
		}
	}

	if count != 32 {
		t.Fatal("expected to read 32 events but got", count, "from", sysmonEvtx)
	}
}

func TestChannels(t *testing.T) {
	channels, err := Channels()
	if err != nil {
		t.Fatal(err)
	}
	assert.NotEmpty(t, channels)

	for _, c := range channels {
		ext := filepath.Ext(c)
		if ext != "" {
			t.Fatal(err)
		}
	}
}

func TestPublishers(t *testing.T) {
	publishers, err := Publishers()
	if err != nil {
		t.Fatal(err)
	}

	assert.NotEmpty(t, publishers)
	for _, p := range publishers {
		t.Log(p)
	}
}
