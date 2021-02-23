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

// +build integration

package filestream

import (
	"bytes"
	"context"
	"os"
	"runtime"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
)

// test_close_renamed from test_harvester.py
func TestFilestreamCloseRenamed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("renaming files while Filebeat is running is not supported on Windows")
	}

	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"paths":                                []string{env.abspath(testlogName) + "*"},
		"prospector.scanner.check_interval":    "1ms",
		"close.on_state_change.check_interval": "1ms",
		"close.on_state_change.renamed":        "true",
	})

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	testlines := []byte("first log line\n")
	env.mustWriteLinesToFile(testlogName, testlines)

	// first event has made it successfully
	env.waitUntilEventCount(1)
	env.requireOffsetInRegistry(testlogName, len(testlines))

	testlogNameRotated := "test.log.rotated"
	env.mustRenameFile(testlogName, testlogNameRotated)

	newerTestlines := []byte("new first log line\nnew second log line\n")
	env.mustWriteLinesToFile(testlogName, newerTestlines)

	// new two events arrived
	env.waitUntilEventCount(3)

	cancelInput()
	env.waitUntilInputStops()

	env.requireOffsetInRegistry(testlogNameRotated, len(testlines))
	env.requireOffsetInRegistry(testlogName, len(newerTestlines))
}

// test_close_removed from test_harvester.py
func TestFilestreamCloseRemoved(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"paths":                                []string{env.abspath(testlogName) + "*"},
		"prospector.scanner.check_interval":    "24h",
		"close.on_state_change.check_interval": "1ms",
		"close.on_state_change.removed":        "true",
	})

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	testlines := []byte("first log line\n")
	env.mustWriteLinesToFile(testlogName, testlines)

	// first event has made it successfully
	env.waitUntilEventCount(1)
	// check registry
	env.requireOffsetInRegistry(testlogName, len(testlines))

	fi, err := os.Stat(env.abspath(testlogName))
	if err != nil {
		t.Fatalf("cannot stat file: %+v", err)
	}

	env.mustRemoveFile(testlogName)

	// the second log line will not be picked up as scan_interval is set to one day.
	env.mustWriteLinesToFile(testlogName, []byte("first line\nsecond log line\n"))

	// new two events arrived
	env.waitUntilEventCount(1)

	cancelInput()
	env.waitUntilInputStops()

	identifier, _ := newINodeDeviceIdentifier(nil)
	src := identifier.GetSource(loginp.FSEvent{Info: fi, Op: loginp.OpCreate, NewPath: env.abspath(testlogName)})
	env.requireOffsetInRegistryByID(src.Name(), len(testlines))
}

// test_close_eof from test_harvester.py
func TestFilestreamCloseEOF(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"paths":                             []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval": "24h",
		"close.reader.on_eof":               "true",
	})

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	testlines := []byte("first log line\n")
	expectedOffset := len(testlines)
	env.mustWriteLinesToFile(testlogName, testlines)

	// first event has made it successfully
	env.waitUntilEventCount(1)
	env.requireOffsetInRegistry(testlogName, expectedOffset)

	// the second log line will not be picked up as scan_interval is set to one day.
	env.mustWriteLinesToFile(testlogName, []byte("first line\nsecond log line\n"))

	// only one event is read
	env.waitUntilEventCount(1)

	cancelInput()
	env.waitUntilInputStops()

	env.requireOffsetInRegistry(testlogName, expectedOffset)
}

// test_empty_lines from test_harvester.py
func TestFilestreamEmptyLine(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"paths":                             []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval": "1ms",
	})

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	testlines := []byte("first log line\nnext is an empty line\n")
	env.mustWriteLinesToFile(testlogName, testlines)

	env.waitUntilEventCount(2)
	env.requireOffsetInRegistry(testlogName, len(testlines))

	moreTestlines := []byte("\nafter an empty line\n")
	env.mustAppendLinesToFile(testlogName, moreTestlines)

	env.waitUntilEventCount(3)

	cancelInput()
	env.waitUntilInputStops()

	env.requireOffsetInRegistry(testlogName, len(testlines)+len(moreTestlines))
}

// test_empty_lines_only from test_harvester.py
// This test differs from the original because in filestream
// input offset is no longer persisted when the line is empty.
func TestFilestreamEmptyLinesOnly(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"paths":                             []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval": "1ms",
	})

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	testlines := []byte("\n\n\n")
	env.mustWriteLinesToFile(testlogName, testlines)

	cancelInput()
	env.waitUntilInputStops()

	env.requireNoEntryInRegistry(testlogName)
}

// test_exceed_buffer from test_harvester.py
func TestFilestreamExceedBuffer(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"paths":       []string{env.abspath(testlogName)},
		"buffer_size": 10,
	})

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	testline := []byte("a line longer than size allowed in buffer_size\n")
	expectedOffset := len(testline)
	env.mustWriteLinesToFile(testlogName, testline)

	// event arrives to the output in full
	env.waitUntilEventCount(1)
	env.requireEventsReceived([]string{string(testline[:len(testline)-1])})

	cancelInput()
	env.waitUntilInputStops()

	env.requireOffsetInRegistry(testlogName, expectedOffset)
}

// test_truncated_file_open from test_harvester.py
func TestFilestreamTruncatedFileOpen(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"paths":                             []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval": "1ms",
	})

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	testlines := []byte("first line\nsecond line\nthird line\n")
	env.mustWriteLinesToFile(testlogName, testlines)

	env.waitUntilEventCount(3)
	env.requireOffsetInRegistry(testlogName, len(testlines))

	env.mustTruncateFile(testlogName, 0)

	truncatedTestLines := []byte("truncated first line\n")
	env.mustWriteLinesToFile(testlogName, truncatedTestLines)
	env.waitUntilEventCount(4)

	cancelInput()
	env.waitUntilInputStops()
	env.requireOffsetInRegistry(testlogName, len(truncatedTestLines))
}

// test_truncated_file_closed from test_harvester.py
func TestFilestreamTruncatedFileClosed(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"paths":                                []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval":    "1ms",
		"close.on_state_change.check_interval": "1ms",
		"close.on_state_change.inactive":       "50ms",
	})

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	testlines := []byte("first line\nsecond line\nthird line\n")
	env.mustWriteLinesToFile(testlogName, testlines)

	env.waitUntilEventCount(3)
	env.requireOffsetInRegistry(testlogName, len(testlines))

	env.waitUntilHarvesterIsDone()

	env.mustTruncateFile(testlogName, 0)

	truncatedTestLines := []byte("truncated first line\n")
	env.mustWriteLinesToFile(testlogName, truncatedTestLines)
	env.waitUntilEventCount(4)

	cancelInput()
	env.waitUntilInputStops()
	env.requireOffsetInRegistry(testlogName, len(truncatedTestLines))
}

// test_truncated_file_closed from test_harvester.py
func TestFilestreamCloseTimeout(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"paths":                                []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval":    "24h",
		"close.on_state_change.check_interval": "100ms",
		"close.reader.after_interval":          "500ms",
	})

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	testlines := []byte("first line\n")
	env.mustWriteLinesToFile(testlogName, testlines)

	env.waitUntilEventCount(1)
	env.requireOffsetInRegistry(testlogName, len(testlines))
	env.waitUntilHarvesterIsDone()

	env.mustWriteLinesToFile(testlogName, []byte("first line\nsecond log line\n"))

	env.waitUntilEventCount(1)

	cancelInput()
	env.waitUntilInputStops()

	env.requireOffsetInRegistry(testlogName, len(testlines))
}

// test_bom_utf8 from test_harvester.py
func TestFilestreamBOMUTF8(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"paths": []string{env.abspath(testlogName)},
	})

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	// BOM: 0xEF,0xBB,0xBF
	lines := append([]byte{0xEF, 0xBB, 0xBF}, []byte(`#Software: Microsoft Exchange Server
#Version: 14.0.0.0
#Log-type: Message Tracking Log
#Date: 2016-04-05T00:00:02.052Z
#Fields: date-time,client-ip,client-hostname,server-ip,server-hostname,source-context,connector-id,source,event-id,internal-message-id,message-id,recipient-address,recipient-status,total-bytes,recipient-count,related-recipient-address,reference,message-subject,sender-address,return-path,message-info,directionality,tenant-id,original-client-ip,original-server-ip,custom-data
2016-04-05T00:00:02.052Z,,,,,"MDB:61914740-3f1b-4ddb-94e0-557196870cfa, Mailbox:279f077c-216f-4323-a9ee-48e50ffd3cad, Event:269492708, MessageClass:IPM.Note.StorageQuotaWarning.Warning, CreationTime:2016-04-05T00:00:01.022Z, ClientType:System",,STOREDRIVER,NOTIFYMAPI,,,,,,,,,,,,,,,,,S:ItemEntryId=00-00-00-00-37-DB-F9-F9-B5-F2-42-4F-86-62-E6-5D-FC-0C-A1-41-07-00-0E-D6-03-16-80-DC-8C-44-9D-30-07-23-ED-71-B7-F7-00-00-1F-D4-B5-0E-00-00-2E-EF-F2-59-0E-E8-2D-46-BC-31-02-85-0D-67-98-43-00-00-37-4A-A3-B3-00-00
2016-04-05T00:00:02.145Z,,,,,"MDB:61914740-3f1b-4ddb-94e0-557196870cfa, Mailbox:49cb09c6-5b76-415d-a085-da0ad9079682, Event:269492711, MessageClass:IPM.Note.StorageQuotaWarning.Warning, CreationTime:2016-04-05T00:00:01.038Z, ClientType:System",,STOREDRIVER,NOTIFYMAPI,,,,,,,,,,,,,,,,,S:ItemEntryId=00-00-00-00-97-8F-07-43-51-44-61-4A-AD-BD-29-D4-97-4E-20-A0-07-00-0E-D6-03-16-80-DC-8C-44-9D-30-07-23-ED-71-B7-F7-00-8E-8F-BD-EB-57-00-00-3D-FB-CE-26-A4-8D-46-4C-A4-35-0F-A7-9B-FA-D7-B9-00-00-37-44-2F-CA-00-00
`)...)
	env.mustWriteLinesToFile(testlogName, lines)

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
				"paths":    []string{env.abspath(testlogName)},
				"encoding": name,
			})

			ctx, cancelInput := context.WithCancel(context.Background())
			env.startInput(ctx, inp)

			line := []byte("first line\n")
			buf := bytes.NewBuffer(nil)
			writer := transform.NewWriter(buf, encoder)
			writer.Write(line)
			writer.Close()

			env.mustWriteLinesToFile(testlogName, buf.Bytes())

			env.waitUntilEventCount(1)

			env.requireEventsReceived([]string{"first line"})

			cancelInput()
			env.waitUntilInputStops()
		})
	}
}

// test_symlinks_enabled from test_harvester.py
func TestFilestreamSymlinksEnabled(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	symlinkName := "test.log.symlink"
	inp := env.mustCreateInput(map[string]interface{}{
		"paths": []string{
			env.abspath(testlogName),
			env.abspath(symlinkName),
		},
		"prospector.scanner.symlinks": "true",
	})

	testlines := []byte("first line\n")
	env.mustWriteLinesToFile(testlogName, testlines)

	env.mustSymlink(testlogName, symlinkName)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	env.waitUntilEventCount(1)

	cancelInput()
	env.waitUntilInputStops()

	env.requireOffsetInRegistry(testlogName, len(testlines))
	env.requireOffsetInRegistry(symlinkName, len(testlines))
}

// test_symlink_rotated from test_harvester.py
func TestFilestreamSymlinkRotated(t *testing.T) {
	env := newInputTestingEnvironment(t)

	firstTestlogName := "test1.log"
	secondTestlogName := "test2.log"
	symlinkName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"paths": []string{
			env.abspath(firstTestlogName),
			env.abspath(secondTestlogName),
			env.abspath(symlinkName),
		},
		"prospector.scanner.check_interval": "1ms",
		"prospector.scanner.symlinks":       "true",
		"close.on_state_change.removed":     "false",
		"clean_removed":                     "false",
	})

	commonLine := "first line in file "
	for i, path := range []string{firstTestlogName, secondTestlogName} {
		env.mustWriteLinesToFile(path, []byte(commonLine+strconv.Itoa(i)+"\n"))
	}

	env.mustSymlink(firstTestlogName, symlinkName)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	env.waitUntilEventCount(2)

	expectedOffset := len(commonLine) + 2
	env.requireOffsetInRegistry(firstTestlogName, expectedOffset)
	env.requireOffsetInRegistry(secondTestlogName, expectedOffset)

	// rotate symlink
	env.mustRemoveFile(symlinkName)
	env.mustSymlink(secondTestlogName, symlinkName)

	moreLines := "second line in file 2\nthird line in file 2\n"
	env.mustAppendLinesToFile(secondTestlogName, []byte(moreLines))

	env.waitUntilEventCount(4)
	env.requireOffsetInRegistry(firstTestlogName, expectedOffset)
	env.requireOffsetInRegistry(secondTestlogName, expectedOffset+len(moreLines))
	env.requireOffsetInRegistry(symlinkName, expectedOffset+len(moreLines))

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
		"paths": []string{
			env.abspath(testlogName),
			env.abspath(symlinkName),
		},
		"prospector.scanner.check_interval": "1ms",
		"prospector.scanner.symlinks":       "true",
		"close.on_state_change.removed":     "false",
		"clean_removed":                     "false",
	})

	line := []byte("first line\n")
	env.mustWriteLinesToFile(testlogName, line)

	env.mustSymlink(testlogName, symlinkName)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	env.waitUntilEventCount(1)

	env.requireOffsetInRegistry(testlogName, len(line))

	// remove symlink
	env.mustRemoveFile(symlinkName)

	env.mustAppendLinesToFile(testlogName, line)

	env.waitUntilEventCount(2)
	env.requireOffsetInRegistry(testlogName, 2*len(line))

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
		"paths": []string{
			env.abspath(testlogName),
			env.abspath(symlinkName),
		},
		"prospector.scanner.check_interval": "1ms",
		"prospector.scanner.symlinks":       "true",
	})

	lines := []byte("first line\nsecond line\nthird line\n")
	env.mustWriteLinesToFile(testlogName, lines)

	env.mustSymlink(testlogName, symlinkName)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	env.waitUntilEventCount(3)

	env.requireOffsetInRegistry(testlogName, len(lines))

	// remove symlink
	env.mustRemoveFile(symlinkName)
	env.mustTruncateFile(testlogName, 0)

	moreLines := []byte("forth line\nfifth line\n")
	env.mustWriteLinesToFile(testlogName, moreLines)

	env.waitUntilEventCount(5)
	env.requireOffsetInRegistry(testlogName, len(moreLines))

	cancelInput()
	env.waitUntilInputStops()

	env.requireRegistryEntryCount(1)
}
