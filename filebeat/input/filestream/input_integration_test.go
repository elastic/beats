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

//go:build integration

package filestream

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// test_close_renamed from test_harvester.py
func TestFilestreamCloseRenamed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("renaming files while Filebeat is running is not supported on Windows")
	}

	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	// prospector.scanner.check_interval must be set to a bigger interval
	// than close.on_state_change.check_interval to make sure
	// the Harvester detects the rename first thus allowing
	// the output to receive the event and then close the source file.
	inp := env.mustCreateInput(map[string]interface{}{
		"id":                                   "fake-ID",
		"paths":                                []string{env.abspath(testlogName) + "*"},
		"prospector.scanner.check_interval":    "10ms",
		"close.on_state_change.check_interval": "1ms",
		"close.on_state_change.renamed":        "true",
	})

	testlines := []byte("first log line\n")
	env.mustWriteToFile(testlogName, testlines)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	// first event has made it successfully
	env.waitUntilEventCount(1)
	env.requireOffsetInRegistry(testlogName, "fake-ID", len(testlines))

	testlogNameRotated := "test.log.rotated"
	env.mustRenameFile(testlogName, testlogNameRotated)

	newerTestlines := []byte("new first log line\nnew second log line\n")
	env.mustWriteToFile(testlogName, newerTestlines)

	env.waitUntilEventCount(3)

	cancelInput()
	env.waitUntilInputStops()

	env.requireOffsetInRegistry(testlogNameRotated, "fake-ID", len(testlines))
	env.requireOffsetInRegistry(testlogName, "fake-ID", len(newerTestlines))
}

func TestFilestreamMetadataUpdatedOnRename(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("renaming files while Filebeat is running is not supported on Windows")
	}

	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"id":                                "fake-ID",
		"paths":                             []string{env.abspath(testlogName) + "*"},
		"prospector.scanner.check_interval": "1ms",
	})

	testline := []byte("log line\n")
	env.mustWriteToFile(testlogName, testline)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	env.waitUntilEventCount(1)
	env.waitUntilMetaInRegistry(testlogName, "fake-ID", fileMeta{Source: env.abspath(testlogName), IdentifierName: "native"})
	env.requireOffsetInRegistry(testlogName, "fake-ID", len(testline))

	testlogNameRenamed := "test.log.renamed"
	env.mustRenameFile(testlogName, testlogNameRenamed)

	// check if the metadata is updated and cursor data stays the same
	env.waitUntilMetaInRegistry(testlogNameRenamed, "fake-ID", fileMeta{Source: env.abspath(testlogNameRenamed), IdentifierName: "native"})
	env.requireOffsetInRegistry(testlogNameRenamed, "fake-ID", len(testline))

	env.mustAppendToFile(testlogNameRenamed, testline)

	env.waitUntilEventCount(2)
	env.requireOffsetInRegistry(testlogNameRenamed, "fake-ID", len(testline)*2)

	cancelInput()
	env.waitUntilInputStops()
}

// test_close_removed from test_harvester.py
func TestFilestreamCloseRemoved(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"id":                                   "fake-ID",
		"paths":                                []string{env.abspath(testlogName) + "*"},
		"prospector.scanner.check_interval":    "24h",
		"close.on_state_change.check_interval": "1ms",
		"close.on_state_change.removed":        "true",
	})

	testlines := []byte("first log line\n")
	env.mustWriteToFile(testlogName, testlines)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	// first event has made it successfully
	env.waitUntilEventCount(1)

	env.requireOffsetInRegistry(testlogName, "fake-ID", len(testlines))

	fi, err := os.Stat(env.abspath(testlogName))
	if err != nil {
		t.Fatalf("cannot stat file: %+v", err)
	}

	env.mustRemoveFile(testlogName)

	env.waitUntilHarvesterIsDone()

	cancelInput()
	env.waitUntilInputStops()

	id := getIDFromPath(env.abspath(testlogName), "fake-ID", fi)
	env.requireOffsetInRegistryByID(id, len(testlines))
}

// test_close_eof from test_harvester.py
func TestFilestreamCloseEOF(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"id":                                "fake-ID",
		"paths":                             []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval": "24h",
		"close.reader.on_eof":               "true",
	})

	testlines := []byte("first log line\n")
	expectedOffset := len(testlines)
	env.mustWriteToFile(testlogName, testlines)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	// first event has made it successfully
	env.waitUntilEventCount(1)
	env.requireOffsetInRegistry(testlogName, "fake-ID", expectedOffset)

	// the second log line will not be picked up as scan_interval is set to one day.
	env.mustWriteToFile(testlogName, []byte("first line\nsecond log line\n"))

	// only one event is read
	env.waitUntilEventCount(1)

	cancelInput()
	env.waitUntilInputStops()

	env.requireOffsetInRegistry(testlogName, "fake-ID", expectedOffset)
}

// test_empty_lines from test_harvester.py
func TestFilestreamEmptyLine(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"id":                                "fake-ID",
		"paths":                             []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval": "1ms",
	})

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	testlines := []byte("first log line\nnext is an empty line\n")
	env.mustWriteToFile(testlogName, testlines)

	env.waitUntilEventCount(2)
	env.requireOffsetInRegistry(testlogName, "fake-ID", len(testlines))

	moreTestlines := []byte("\nafter an empty line\n")
	env.mustAppendToFile(testlogName, moreTestlines)

	env.waitUntilEventCount(3)
	env.requireEventsReceived([]string{
		"first log line",
		"next is an empty line",
		"after an empty line",
	})

	cancelInput()
	env.waitUntilInputStops()

	env.requireOffsetInRegistry(testlogName, "fake-ID", len(testlines)+len(moreTestlines))
}

// test_empty_lines_only from test_harvester.py
// This test differs from the original because in filestream
// input offset is no longer persisted when the line is empty.
func TestFilestreamEmptyLinesOnly(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"id":                                "fake-ID",
		"paths":                             []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval": "1ms",
	})

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	testlines := []byte("\n\n\n")
	env.mustWriteToFile(testlogName, testlines)

	cancelInput()
	env.waitUntilInputStops()

	env.requireNoEntryInRegistry(testlogName, "fake-ID")
}

// test_bom_utf8 from test_harvester.py
func TestFilestreamBOMUTF8(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"id":    "fake-ID",
		"paths": []string{env.abspath(testlogName)},
	})

	// BOM: 0xEF,0xBB,0xBF
	lines := append([]byte{0xEF, 0xBB, 0xBF}, []byte(`#Software: Microsoft Exchange Server
#Version: 14.0.0.0
#Log-type: Message Tracking Log
#Date: 2016-04-05T00:00:02.052Z
#Fields: date-time,client-ip,client-hostname,server-ip,server-hostname,source-context,connector-id,source,event-id,internal-message-id,message-id,recipient-address,recipient-status,total-bytes,recipient-count,related-recipient-address,reference,message-subject,sender-address,return-path,message-info,directionality,tenant-id,original-client-ip,original-server-ip,custom-data
2016-04-05T00:00:02.052Z,,,,,"MDB:61914740-3f1b-4ddb-94e0-557196870cfa, Mailbox:279f077c-216f-4323-a9ee-48e50ffd3cad, Event:269492708, MessageClass:IPM.Note.StorageQuotaWarning.Warning, CreationTime:2016-04-05T00:00:01.022Z, ClientType:System",,STOREDRIVER,NOTIFYMAPI,,,,,,,,,,,,,,,,,S:ItemEntryId=00-00-00-00-37-DB-F9-F9-B5-F2-42-4F-86-62-E6-5D-FC-0C-A1-41-07-00-0E-D6-03-16-80-DC-8C-44-9D-30-07-23-ED-71-B7-F7-00-00-1F-D4-B5-0E-00-00-2E-EF-F2-59-0E-E8-2D-46-BC-31-02-85-0D-67-98-43-00-00-37-4A-A3-B3-00-00
2016-04-05T00:00:02.145Z,,,,,"MDB:61914740-3f1b-4ddb-94e0-557196870cfa, Mailbox:49cb09c6-5b76-415d-a085-da0ad9079682, Event:269492711, MessageClass:IPM.Note.StorageQuotaWarning.Warning, CreationTime:2016-04-05T00:00:01.038Z, ClientType:System",,STOREDRIVER,NOTIFYMAPI,,,,,,,,,,,,,,,,,S:ItemEntryId=00-00-00-00-97-8F-07-43-51-44-61-4A-AD-BD-29-D4-97-4E-20-A0-07-00-0E-D6-03-16-80-DC-8C-44-9D-30-07-23-ED-71-B7-F7-00-8E-8F-BD-EB-57-00-00-3D-FB-CE-26-A4-8D-46-4C-A4-35-0F-A7-9B-FA-D7-B9-00-00-37-44-2F-CA-00-00
`)...)
	env.mustWriteToFile(testlogName, lines)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	env.waitUntilEventCount(7)

	cancelInput()
	env.waitUntilInputStops()

	messages := env.getOutputMessages()
	require.Equal(t, messages[0], "#Software: Microsoft Exchange Server")
}

// test_boms from test_harvester.py
func TestFilestreamUTF16BOMs(t *testing.T) {
	encodings := map[string]encoding.Encoding{
		"utf-16be-bom": unicode.UTF16(unicode.BigEndian, unicode.UseBOM),
		"utf-16le-bom": unicode.UTF16(unicode.LittleEndian, unicode.UseBOM),
	}

	for name, enc := range encodings {
		name := name
		encoder := enc.NewEncoder()
		t.Run(name, func(t *testing.T) {
			env := newInputTestingEnvironment(t)

			testlogName := "test.log"
			inp := env.mustCreateInput(map[string]interface{}{
				"id":       "fake-ID",
				"paths":    []string{env.abspath(testlogName)},
				"encoding": name,
			})

			line := []byte("first line\n")
			buf := bytes.NewBuffer(nil)
			writer := transform.NewWriter(buf, encoder)
			writer.Write(line)
			writer.Close()

			env.mustWriteToFile(testlogName, buf.Bytes())

			ctx, cancelInput := context.WithCancel(context.Background())
			env.startInput(ctx, inp)

			env.waitUntilEventCount(1)

			env.requireEventsReceived([]string{"first line"})

			cancelInput()
			env.waitUntilInputStops()
		})
	}
}

// test_close_timeout from test_harvester.py
func TestFilestreamCloseTimeout(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"id":                                   "fake-ID",
		"paths":                                []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval":    "24h",
		"close.on_state_change.check_interval": "100ms",
		"close.reader.after_interval":          "500ms",
	})

	testlines := []byte("first line\n")
	env.mustWriteToFile(testlogName, testlines)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	env.waitUntilEventCount(1)
	env.requireOffsetInRegistry(testlogName, "fake-ID", len(testlines))
	env.waitUntilHarvesterIsDone()

	env.mustWriteToFile(testlogName, []byte("first line\nsecond log line\n"))

	env.waitUntilEventCount(1)

	cancelInput()
	env.waitUntilInputStops()

	env.requireOffsetInRegistry(testlogName, "fake-ID", len(testlines))
}

// test_close_inactive from test_input.py
func TestFilestreamCloseAfterInterval(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"id":                                   "fake-ID",
		"paths":                                []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval":    "24h",
		"close.on_state_change.check_interval": "100ms",
		"close.on_state_change.inactive":       "2s",
	})

	testlines := []byte("first line\nsecond line\nthird line\n")
	env.mustWriteToFile(testlogName, testlines)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	env.waitUntilEventCount(3)
	env.requireOffsetInRegistry(testlogName, "fake-ID", len(testlines))
	env.waitUntilHarvesterIsDone()

	cancelInput()
	env.waitUntilInputStops()
}

// test_close_inactive_file_removal from test_input.py
func TestFilestreamCloseAfterIntervalRemoved(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"id":                                   "fake-ID",
		"paths":                                []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval":    "24h",
		"close.on_state_change.check_interval": "10ms",
		"close.on_state_change.inactive":       "100ms",
		// reader is not stopped when file is removed to see if the reader can still detect
		// if the file has been inactive even if it have been removed in the meantime
		"close.on_state_change.removed": "false",
	})

	testlines := []byte("first line\nsecond line\nthird line\n")
	env.mustWriteToFile(testlogName, testlines)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	env.waitUntilEventCount(3)
	env.requireOffsetInRegistry(testlogName, "fake-ID", len(testlines))

	env.mustRemoveFile(testlogName)

	env.waitUntilHarvesterIsDone()

	cancelInput()
	env.waitUntilInputStops()
}

func TestFilestreamCloseAfterIntervalRenamed(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"id":                                   "fake-ID",
		"paths":                                []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval":    "24h",
		"close.on_state_change.check_interval": "10ms",
		"close.on_state_change.inactive":       "100ms",
		// reader is not stopped when file is removed to see if the reader can still detect
		// if the file has been inactive even if it have been removed in the meantime
		"close.on_state_change.removed": "false",
	})

	testlines := []byte("first line\nsecond line\nthird line\n")
	env.mustWriteToFile(testlogName, testlines)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	env.waitUntilEventCount(3)
	env.requireOffsetInRegistry(testlogName, "fake-ID", len(testlines))

	newFileName := "test_rotated.log"
	env.mustRenameFile(testlogName, newFileName)

	env.waitUntilHarvesterIsDone()

	cancelInput()
	env.waitUntilInputStops()
}

// test_close_inactive_file_rotation_and_removal from test_input.py
func TestFilestreamCloseAfterIntervalRotatedAndRemoved(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"id":                                   "fake-ID",
		"paths":                                []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval":    "24h",
		"close.on_state_change.check_interval": "10ms",
		"close.on_state_change.inactive":       "100ms",
		// reader is not stopped when file is removed to see if the reader can still detect
		// if the file has been inactive even if it have been removed in the meantime
		"close.on_state_change.removed": "false",
	})

	testlines := []byte("first line\nsecond line\nthird line\n")
	env.mustWriteToFile(testlogName, testlines)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	env.waitUntilEventCount(3)
	env.requireOffsetInRegistry(testlogName, "fake-ID", len(testlines))

	newFileName := "test_rotated.log"
	env.mustRenameFile(testlogName, newFileName)
	env.mustRemoveFile(newFileName)

	env.waitUntilHarvesterIsDone()

	cancelInput()
	env.waitUntilInputStops()
}

// test_close_inactive_file_rotation_and_removal2 from test_input.py
func TestFilestreamCloseAfterIntervalRotatedAndNewRemoved(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"id":                                   "fake-ID",
		"paths":                                []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval":    "1ms",
		"close.on_state_change.check_interval": "10ms",
		"close.on_state_change.inactive":       "100ms",
		// reader is not stopped when file is removed to see if the reader can still detect
		// if the file has been inactive even if it have been removed in the meantime
		"close.on_state_change.removed": "false",
	})

	testlines := []byte("first line\nsecond line\nthird line\n")
	env.mustWriteToFile(testlogName, testlines)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	env.waitUntilEventCount(3)
	env.requireOffsetInRegistry(testlogName, "fake-ID", len(testlines))

	newFileName := "test_rotated.log"
	env.mustRenameFile(testlogName, newFileName)

	env.waitUntilHarvesterIsDone()

	newTestlines := []byte("rotated first line\nrotated second line\nrotated third line\n")
	env.mustWriteToFile(testlogName, newTestlines)

	env.waitUntilEventCount(6)

	env.mustRemoveFile(newFileName)

	env.waitUntilHarvesterIsDone()

	cancelInput()
	env.waitUntilInputStops()
}

// test_truncated_file_open from test_harvester.py
func TestFilestreamTruncatedFileOpen(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"id":                                 "fake-ID",
		"paths":                              []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval":  "1ms",
		"prospector.scanner.resend_on_touch": "true",
	})

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	testlines := []byte("first line\nsecond line\nthird line\n")
	env.mustWriteToFile(testlogName, testlines)

	env.waitUntilEventCount(3)
	env.requireOffsetInRegistry(testlogName, "fake-ID", len(testlines))

	env.mustTruncateFile(testlogName, 0)
	time.Sleep(5 * time.Millisecond)

	truncatedTestLines := []byte("truncated first line\n")
	env.mustWriteToFile(testlogName, truncatedTestLines)
	env.waitUntilEventCount(4)

	cancelInput()
	env.waitUntilInputStops()
	env.requireOffsetInRegistry(testlogName, "fake-ID", len(truncatedTestLines))
}

// test_truncated_file_closed from test_harvester.py
func TestFilestreamTruncatedFileClosed(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"id":                                 "fake-ID",
		"paths":                              []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval":  "1ms",
		"prospector.scanner.resend_on_touch": "true",
		"close.reader.on_eof":                "true",
	})

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	testlines := []byte("first line\nsecond line\nthird line\n")
	env.mustWriteToFile(testlogName, testlines)

	env.waitUntilEventCount(3)
	env.requireOffsetInRegistry(testlogName, "fake-ID", len(testlines))

	env.waitUntilHarvesterIsDone()

	env.mustTruncateFile(testlogName, 0)
	time.Sleep(5 * time.Millisecond)

	truncatedTestLines := []byte("truncated first line\n")
	env.mustWriteToFile(testlogName, truncatedTestLines)
	env.waitUntilEventCount(4)

	cancelInput()
	env.waitUntilInputStops()
	env.requireOffsetInRegistry(testlogName, "fake-ID", len(truncatedTestLines))
}

// test_truncate from test_harvester.py
func TestFilestreamTruncateWithSymlink(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	symlinkName := "test.log.symlink"
	inp := env.mustCreateInput(map[string]interface{}{
		"id": "fake-ID",
		"paths": []string{
			env.abspath(testlogName),
			env.abspath(symlinkName),
		},
		"prospector.scanner.check_interval":  "1ms",
		"prospector.scanner.resend_on_touch": "true",
		"prospector.scanner.symlinks":        "true",
	})

	lines := []byte("first line\nsecond line\nthird line\n")
	env.mustWriteToFile(testlogName, lines)

	env.mustSymlink(testlogName, symlinkName)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	env.waitUntilEventCount(3)

	env.requireOffsetInRegistry(testlogName, "fake-ID", len(lines))

	// remove symlink
	env.mustRemoveFile(symlinkName)
	env.mustTruncateFile(testlogName, 0)
	env.waitUntilOffsetInRegistry(testlogName, "fake-ID", 0, 10*time.Second)

	moreLines := []byte("forth line\nfifth line\n")
	env.mustWriteToFile(testlogName, moreLines)

	env.waitUntilEventCount(5)
	env.requireOffsetInRegistry(testlogName, "fake-ID", len(moreLines))

	cancelInput()
	env.waitUntilInputStops()

	env.requireRegistryEntryCount(1)
}

func TestFilestreamTruncateBigScannerInterval(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"id":                                 "fake-ID",
		"paths":                              []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval":  "5s",
		"prospector.scanner.resend_on_touch": "true",
	})

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	testlines := []byte("first line\nsecond line\nthird line\n")
	env.mustWriteToFile(testlogName, testlines)

	env.waitUntilEventCount(3)
	env.requireOffsetInRegistry(testlogName, "fake-ID", len(testlines))

	env.mustTruncateFile(testlogName, 0)

	truncatedTestLines := []byte("truncated first line\n")
	env.mustWriteToFile(testlogName, truncatedTestLines)

	env.waitUntilEventCount(3)

	cancelInput()
	env.waitUntilInputStops()
}

func TestFilestreamTruncateCheckOffset(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"id":                                 "fake-ID",
		"paths":                              []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval":  "1ms",
		"prospector.scanner.resend_on_touch": "true",
	})

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	testlines := []byte("first line\nsecond line\nthird line\n")
	env.mustWriteToFile(testlogName, testlines)

	env.waitUntilEventCount(3)
	env.requireOffsetInRegistry(testlogName, "fake-ID", len(testlines))

	env.mustTruncateFile(testlogName, 0)

	env.waitUntilOffsetInRegistry(testlogName, "fake-ID", 0, 10*time.Second)

	cancelInput()
	env.waitUntilInputStops()
}

func TestFilestreamTruncateBlockedOutput(t *testing.T) {
	env := newInputTestingEnvironment(t)
	env.pipeline = &mockPipelineConnector{blocking: true}

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"id":                                "fake-ID",
		"paths":                             []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval": "200ms",
	})

	testlines := []byte("first line\nsecond line\n")
	env.mustWriteToFile(testlogName, testlines)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	for env.pipeline.clientsCount() != 1 {
		time.Sleep(10 * time.Millisecond)
	}
	env.pipeline.clients[0].waitUntilPublishingHasStarted()
	env.pipeline.clients[0].canceler()

	env.waitUntilEventCount(2)
	env.requireOffsetInRegistry(testlogName, "fake-ID", len(testlines))

	// extra lines are appended after first line is processed
	// so it can interfere with the truncation of the file
	env.mustAppendToFile(testlogName, []byte("third line\n"))

	env.mustTruncateFile(testlogName, 0)

	env.waitUntilOffsetInRegistry(testlogName, "fake-ID", 0, 10*time.Second)

	// all newly started client has to be cancelled so events can be processed
	env.pipeline.cancelAllClients()
	// if a new client shows up, it should not block
	env.pipeline.invertBlocking()

	truncatedTestLines := []byte("truncated line\n")
	env.mustWriteToFile(testlogName, truncatedTestLines)

	env.waitUntilEventCount(3)
	env.waitUntilOffsetInRegistry(testlogName, "fake-ID", len(truncatedTestLines), 10*time.Second)

	cancelInput()
	env.waitUntilInputStops()
}

// test_symlinks_enabled from test_harvester.py
func TestFilestreamSymlinksEnabled(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	symlinkName := "test.log.symlink"
	inp := env.mustCreateInput(map[string]interface{}{
		"id": "fake-ID",
		"paths": []string{
			env.abspath(symlinkName),
		},
		"prospector.scanner.symlinks": "true",
	})

	testlines := []byte("first line\n")
	env.mustWriteToFile(testlogName, testlines)

	env.mustSymlink(testlogName, symlinkName)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	env.waitUntilEventCount(1)

	cancelInput()
	env.waitUntilInputStops()

	env.requireOffsetInRegistry(testlogName, "fake-ID", len(testlines))
}

// test_symlink_rotated from test_harvester.py
func TestFilestreamSymlinkRotated(t *testing.T) {
	env := newInputTestingEnvironment(t)

	firstTestlogName := "test1.log"
	secondTestlogName := "test2.log"
	symlinkName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"id": "fake-ID",
		"paths": []string{
			env.abspath(symlinkName),
		},
		"prospector.scanner.check_interval": "1ms",
		"prospector.scanner.symlinks":       "true",
		"close.on_state_change.removed":     "false",
		"clean_removed":                     "false",
	})

	commonLine := "first line in file "
	for i, path := range []string{firstTestlogName, secondTestlogName} {
		env.mustWriteToFile(path, []byte(commonLine+strconv.Itoa(i)+"\n"))
	}

	env.mustSymlink(firstTestlogName, symlinkName)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	env.waitUntilEventCount(1)

	expectedOffset := len(commonLine) + 2
	env.requireOffsetInRegistry(firstTestlogName, "fake-ID", expectedOffset)

	// rotate symlink
	env.mustRemoveFile(symlinkName)
	env.mustSymlink(secondTestlogName, symlinkName)

	moreLines := "second line in file 2\nthird line in file 2\n"
	env.mustAppendToFile(secondTestlogName, []byte(moreLines))

	env.waitUntilEventCount(4)
	env.requireOffsetInRegistry(firstTestlogName, "fake-ID", expectedOffset)
	env.requireOffsetInRegistry(secondTestlogName, "fake-ID", expectedOffset+len(moreLines))

	cancelInput()
	env.waitUntilInputStops()

	env.requireRegistryEntryCount(2)
}

// test_symlink_removed from test_harvester.py
func TestFilestreamSymlinkRemoved(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	symlinkName := "test.log.symlink"
	inp := env.mustCreateInput(map[string]interface{}{
		"id": "fake-ID",
		"paths": []string{
			env.abspath(symlinkName),
		},
		"prospector.scanner.check_interval": "1ms",
		"prospector.scanner.symlinks":       "true",
		"close.on_state_change.removed":     "false",
		"clean_removed":                     "false",
	})

	line := []byte("first line\n")
	env.mustWriteToFile(testlogName, line)

	env.mustSymlink(testlogName, symlinkName)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	env.waitUntilEventCount(1)

	env.requireOffsetInRegistry(testlogName, "fake-ID", len(line))

	// remove symlink
	env.mustRemoveFile(symlinkName)

	env.mustAppendToFile(testlogName, line)

	env.waitUntilEventCount(2)
	env.requireOffsetInRegistry(testlogName, "fake-ID", 2*len(line))

	cancelInput()
	env.waitUntilInputStops()

	env.requireRegistryEntryCount(1)
}

// test_truncate from test_harvester.py
func TestFilestreamTruncate(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	symlinkName := "test.log.symlink"
	inp := env.mustCreateInput(map[string]interface{}{
		"id": "fake-ID",
		"paths": []string{
			env.abspath("*"),
		},
		"prospector.scanner.check_interval":  "1ms",
		"prospector.scanner.resend_on_touch": "true",
		"prospector.scanner.symlinks":        "true",
	})

	lines := []byte("first line\nsecond line\nthird line\n")
	env.mustWriteToFile(testlogName, lines)

	env.mustSymlink(testlogName, symlinkName)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	env.waitUntilEventCount(3)

	env.requireOffsetInRegistry(testlogName, "fake-ID", len(lines))

	// remove symlink
	env.mustRemoveFile(symlinkName)
	env.mustTruncateFile(testlogName, 0)
	env.waitUntilOffsetInRegistry(testlogName, "fake-ID", 0, 10*time.Second)

	// recreate symlink
	env.mustSymlink(testlogName, symlinkName)

	moreLines := []byte("forth line\nfifth line\n")
	env.mustWriteToFile(testlogName, moreLines)

	env.waitUntilOffsetInRegistry(testlogName, "fake-ID", len(moreLines), 10*time.Second)

	cancelInput()
	env.waitUntilInputStops()

	env.requireRegistryEntryCount(1)
}

func TestFilestreamHarvestAllFilesWhenHarvesterLimitExceeded(t *testing.T) {
	env := newInputTestingEnvironment(t)

	logFiles := []struct {
		path  string
		lines []string
	}{
		{path: "log-a.log",
			lines: []string{"1-aaaaaaaaaa", "2-aaaaaaaaaa"}},
		{path: "log-b.log",
			lines: []string{"1-bbbbbbbbbb", "2-bbbbbbbbbb"}},
	}
	for _, lf := range logFiles {
		env.mustWriteToFile(
			lf.path, []byte(strings.Join(lf.lines, "\n")+"\n"))
	}

	inp := env.mustCreateInput(map[string]interface{}{
		"id":                  "TestFilestreamHarvestAllFilesWhenHarvesterLimitExceeded",
		"harvester_limit":     1,
		"close.reader.on_eof": true,
		"paths": []string{
			env.abspath(logFiles[0].path),
			env.abspath(logFiles[1].path)},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)

	env.startInput(ctx, inp)

	env.waitUntilEventCountCtx(ctx, 4)

	cancel()
	env.waitUntilInputStops()
}

func TestGlobalIDCannotBeUsed(t *testing.T) {
	env := newInputTestingEnvironment(t)
	testlogName := "test.log"
	_, err := env.createInput(map[string]interface{}{
		"id":    ".global",
		"paths": []string{env.abspath(testlogName) + "*"},
	})
	if err == nil {
		t.Fatal("expecting an error because '.global' cannot be used as input ID")
	}
}

// test_rotating_close_inactive_larger_write_rate from test_input.py
func TestRotatingCloseInactiveLargerWriteRate(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"id": "my-id",
		"paths": []string{
			env.abspath("*"),
		},
		"prospector.scanner.check_interval":    "100ms",
		"close.on_state_change.check_interval": "1s",
		"close.on_state_change.inactive":       "5s",
		"ignore_older":                         "10s",
	})

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	time.Sleep(1 * time.Second)

	rotations := 2
	iterations := 3
	r := 0
	for r <= rotations {
		f, err := os.Create(env.abspath(testlogName))
		if err != nil {
			t.Fatalf("failed to open log file: %+v", err)
		}
		n := 0
		for n <= iterations {
			f.Write([]byte(fmt.Sprintf("hello world %d\n", r*iterations+n)))
			n += 1
			time.Sleep(100 * time.Millisecond)
		}
		env.mustRenameFile(testlogName, testlogName+time.Now().Format("2006-01-02T15:04:05.99999999"))
		r += 1
	}

	// allow for events to be send multiple times due to log rotation
	env.waitUntilAtLeastEventCount(rotations * iterations)

	cancelInput()
	env.waitUntilInputStops()
}

// test_rotating_close_inactive_low_write_rate from test_input.py
func TestRotatingCloseInactiveLowWriteRate(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"id": "my-id",
		"paths": []string{
			env.abspath("*"),
		},
		"prospector.scanner.check_interval":    "1ms",
		"close.on_state_change.check_interval": "1ms",
		"close.on_state_change.inactive":       "1s",
		"ignore_older":                         "10s",
	})

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	time.Sleep(1 * time.Second)

	env.mustWriteToFile(testlogName, []byte("Line 1\n"))
	env.waitUntilEventCount(1)

	env.mustRenameFile(testlogName, testlogName+".1")

	env.waitUntilHarvesterIsDone()
	time.Sleep(2 * time.Second)

	env.mustWriteToFile(testlogName, []byte("Line 2\n"))

	// allow for events to be send multiple times due to log rotation
	env.waitUntilAtLeastEventCount(2)

	cancelInput()
	env.waitUntilInputStops()
}

func FuzzFilestreamID(f *testing.F) {
	f.Add("foo")
	f.Add("my-ID")
	f.Add("id with a space")
	f.Add("Hello, 世界")
	f.Add("foo::bar::")
	f.Add("::")
	f.Add(" ")
	f.Fuzz(func(t *testing.T, id string) {
		// Execute some testing to avoid "invalid" strings
		if id == "" {
			return
		}

		// Skip if id is not a valid UTF-8 string
		if !utf8.ValidString(id) {
			return
		}

		// Skip if any rune is not valid or the null character
		for _, r := range []rune(id) {
			if r == 0 || !utf8.ValidRune(r) {
				return
			}
		}

		// Test starts here
		env := newInputTestingEnvironment(t)
		testlogName := "test.log"
		inp := env.mustCreateInput(map[string]interface{}{
			"id":    id,
			"paths": []string{env.abspath(testlogName) + "*"},
		})

		testlines := []byte("first log line\n")
		env.mustWriteToFile(testlogName, testlines)

		ctx, cancelInput := context.WithCancel(context.Background())
		env.startInput(ctx, inp)

		// first event has made it successfully
		env.waitUntilEventCount(1)
		env.requireOffsetInRegistry(testlogName, id, len(testlines))

		newerTestlines := []byte("new first log line\nnew second log line\n")
		env.mustWriteToFile(testlogName, newerTestlines)

		env.waitUntilEventCount(3)

		cancelInput()
		env.waitUntilInputStops()

		env.requireOffsetInRegistry(testlogName, id, len(newerTestlines))
	})
}
