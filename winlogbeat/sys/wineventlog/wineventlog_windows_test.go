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
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/winlogbeat/sys/winevent"
)

var updateXML = flag.Bool("update", false, "update XML golden files from evtx files in testdata")

func TestWinEventLog(t *testing.T) {
	for _, test := range []struct {
		path   string
		events int
	}{
		{path: "application-windows-error-reporting.evtx", events: 1},
		{path: "sysmon-9.01.evtx", events: 32},
		{path: "ec1.evtx", events: 1},          // eventcreate /id 1000 /t error /l application /d "My custom error event for the application log"
		{path: "ec2.evtx", events: 1},          // eventcreate /id 999 /t error /l application /so WinWord /d "Winword event 999 happened due to low diskspace"
		{path: "ec3.evtx", events: 1},          // eventcreate /id 5 /t error /l system /d "Catastrophe!"
		{path: "ec4.evtx", events: 1},          // eventcreate /id 5 /t error /l system /so Backup /d "Backup failure"
		{path: "ec3and4.evtx", events: 2},      // ec3 and ec3 exported as a single evtx.
		{path: "original.evtx", events: 5},     // a capture from a short generation of the eventlog WindowsEventLogAPI test.
		{path: "experimental.evtx", events: 5}, // a capture from a short generation of the eventlog WindowsEventLogAPIExperimental test.
	} {
		t.Run(test.path, func(t *testing.T) {
			evtx, err := filepath.Abs(filepath.Join("testdata", test.path))
			if err != nil {
				t.Fatal(err)
			}
			xmlPath := evtx[:len(evtx)-len("evtx")] + "xml"

			if _, err = os.Lstat(evtx); err != nil {
				t.Fatal(err)
			}

			t.Run("EvtOpenLog", func(t *testing.T) {
				h, err := EvtOpenLog(0, evtx, EvtOpenFilePath)
				if err != nil {
					t.Fatal(err)
				}
				defer Close(h) //nolint:errcheck // This is just a resource release.
			})

			t.Run("EvtQuery", func(t *testing.T) {
				h, err := EvtQuery(0, evtx, "", EvtQueryFilePath)
				if err != nil {
					t.Fatal(err)
				}
				defer Close(h) //nolint:errcheck // This is just a resource release.
			})

			t.Run("ReadEvtx", func(t *testing.T) {
				// Open .evtx file.
				h, err := EvtQuery(0, evtx, "", EvtQueryFilePath|EvtQueryReverseDirection)
				if err != nil {
					t.Fatal(err)
				}
				defer Close(h) //nolint:errcheck // This is just a resource release.

				// Get handles to events.
				buf := make([]byte, 32*1024)
				var out io.Writer
				if *updateXML {
					f, err := os.Create(xmlPath)
					if err != nil {
						t.Fatalf("failed to create golden file: %v", err)
					}
					defer f.Close()
					out = f
				} else {
					out = &bytes.Buffer{}
				}
				var count int
				for {
					handles, err := EventHandles(h, 8)
					if err == ERROR_NO_MORE_ITEMS { //nolint:errorlint // This is never wrapped.
						t.Log(err)
						break
					}
					if err != nil {
						t.Fatal(err)
					}

					// Read events.
					for _, h := range handles {
						if err = RenderEventXML(h, buf, out); err != nil {
							t.Fatal(err)
						}
						Close(h) //nolint:errcheck // This is just a resource release.
						fmt.Fprintln(out)
						count++
					}
				}
				if !*updateXML {
					f, err := os.Open(xmlPath)
					if err != nil {
						t.Fatalf("failed to read golden file: %v", err)
					}
					want, err := unmarshalXMLEvents(f)
					if err != nil {
						t.Fatalf("failed to unmarshal golden events: %v", err)
					}
					got, err := unmarshalXMLEvents(out.(*bytes.Buffer))
					if err != nil {
						t.Fatalf("failed to unmarshal obtained events: %v", err)
					}
					if !reflect.DeepEqual(want, got) {
						t.Errorf("unexpected result for %s: got:- want:+\n%s", test.path, cmp.Diff(want, got))
					}
				}

				if count != test.events {
					t.Errorf("expected to read %d events but got %d from %s", test.events, count, test.path)
				}
			})
		})
	}
}

// unmarshalXMLEvents unmarshals a complete set of events from the XML data
// in the provided io.Reader. GUID values are canonicalised to lowercase.
func unmarshalXMLEvents(r io.Reader) ([]winevent.Event, error) {
	var events []winevent.Event
	decoder := xml.NewDecoder(r)
	for {
		var e winevent.Event
		err := decoder.Decode(&e)
		if err != nil {
			if err != io.EOF { //nolint:errorlint // This is never wrapped.
				return nil, err
			}
			break
		}
		events = append(events, canonical(e))
	}
	return events, nil
}

// canonical return e with its GUID values canonicalised to lower case.
// Different versions of Windows render these values in different cases; ¯\_(ツ)_/¯
func canonical(e winevent.Event) winevent.Event {
	e.Provider.GUID = strings.ToLower(e.Provider.GUID)
	for i, kv := range e.EventData.Pairs {
		if strings.Contains(strings.ToLower(kv.Key), "guid") {
			e.EventData.Pairs[i].Value = strings.ToLower(kv.Value)
		}
	}
	return e
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
