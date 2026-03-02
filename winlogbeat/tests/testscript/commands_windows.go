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

//go:build windows

package scripttest

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/rogpeppe/go-internal/testscript"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"github.com/elastic/beats/v7/winlogbeat/sys/wineventlog"
)

const eventLogKeyName = `SYSTEM\CurrentControlSet\Services\EventLog`

func customCommands() map[string]func(*testscript.TestScript, bool, []string) {
	return map[string]func(*testscript.TestScript, bool, []string){
		"envsubst":                   cmdEnvsubst,
		"write-event":                cmdWriteEvent,
		"write-multiline-event":      cmdWriteMultilineEvent,
		"clear-event-log":            cmdClearEventLog,
		"check-event-count":          cmdCheckEventCount,
		"check-event-field":          cmdCheckEventField,
		"check-event-field-exists":   cmdCheckEventFieldExists,
		"check-event-field-absent":   cmdCheckEventFieldAbsent,
		"check-event-field-contains": cmdCheckEventFieldContains,
		"sleep":                      cmdSleep,
		"wait-for-event-count":       cmdWaitForEventCount,
		"wait-for-event-log":         cmdWaitForEventLog,
	}
}

// setupTest is called once per txtar test file. It derives a unique provider
// name from the work directory, registers the Windows event log sources, sets
// environment variables, and registers cleanup to remove the sources.
func setupTest(env *testscript.Env) error {
	// Use the work directory as uniqueness input — it is unique per test run.
	hash := sha256.Sum256([]byte(env.Cd))
	suffix := fmt.Sprintf("%x", hash[:3]) // 6 hex chars

	provider := "WinlogbeatTest_" + suffix
	appSrc := "WinlogbeatApp_" + suffix
	otherSrc := "WinlogbeatOther_" + suffix

	if _, err := installAsEventCreate(provider, appSrc); err != nil {
		return fmt.Errorf("install event source %s: %w", appSrc, err)
	}
	if _, err := installAsEventCreate(provider, otherSrc); err != nil {
		removeSource(provider, appSrc) //nolint:errcheck // best-effort cleanup on setup failure
		return fmt.Errorf("install event source %s: %w", otherSrc, err)
	}

	env.Setenv("PROVIDER", provider)
	env.Setenv("APP_SOURCE", appSrc)
	env.Setenv("OTHER_APP_SOURCE", otherSrc)

	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("get hostname: %w", err)
	}
	env.Setenv("HOSTNAME", hostname)

	userSID, err := getCurrentUserSID()
	if err != nil {
		return fmt.Errorf("get current user SID: %w", err)
	}
	env.Setenv("USER_SID", userSID)

	// Register cleanup via Cleanup if the underlying T supports it.
	type cleanupT interface {
		Cleanup(func())
	}
	if c, ok := env.T().(cleanupT); ok {
		c.Cleanup(func() {
			wineventlog.EvtClearLog(wineventlog.NilHandle, provider, "") //nolint:errcheck // best-effort cleanup
			removeSource(provider, appSrc)                               //nolint:errcheck // best-effort cleanup
			removeSource(provider, otherSrc)                             //nolint:errcheck // best-effort cleanup
			removeProvider(provider)                                     //nolint:errcheck // best-effort cleanup
		})
	}

	return nil
}

// cmdWriteEvent implements:
//
//	write-event <message> [--source=name] [--id=N] [--level=information|warning|error] [--sid=S]
func cmdWriteEvent(script *testscript.TestScript, neg bool, args []string) {
	if len(args) < 1 {
		script.Fatalf("usage: write-event <message> [--source=name] [--id=N] [--level=information|warning|error] [--sid=S]")
	}

	msg := args[0]
	source := script.Getenv("APP_SOURCE")
	eventID := uint32(10)
	level := "information"
	sidStr := ""

	for _, arg := range args[1:] {
		switch {
		case strings.HasPrefix(arg, "--source="):
			source = arg[len("--source="):]
		case strings.HasPrefix(arg, "--id="):
			n, err := strconv.ParseUint(arg[len("--id="):], 10, 32)
			if err != nil {
				script.Fatalf("write-event: invalid --id: %v", err)
			}
			eventID = uint32(n)
		case strings.HasPrefix(arg, "--level="):
			level = arg[len("--level="):]
		case strings.HasPrefix(arg, "--sid="):
			sidStr = arg[len("--sid="):]
		default:
			script.Fatalf("write-event: unknown argument: %s", arg)
		}
	}

	var eventType uint16
	switch strings.ToLower(level) {
	case "information", "info":
		eventType = windows.EVENTLOG_INFORMATION_TYPE
	case "warning", "warn":
		eventType = windows.EVENTLOG_WARNING_TYPE
	case "error":
		eventType = windows.EVENTLOG_ERROR_TYPE
	case "success":
		eventType = windows.EVENTLOG_SUCCESS
	default:
		script.Fatalf("write-event: unknown level: %s", level)
	}

	var sid *windows.SID
	switch {
	case sidStr != "":
		var err error
		sid, err = windows.StringToSid(sidStr)
		if err != nil {
			script.Fatalf("write-event: invalid SID %q: %v", sidStr, err)
		}
	default:
		// Default to the current user's SID, matching the Python test
		// behavior where write_event_log() always attached the process
		// user's SID unless explicitly overridden.
		sidStr, err := getCurrentUserSID()
		if err != nil {
			script.Fatalf("write-event: get current user SID: %v", err)
		}
		sid, err = windows.StringToSid(sidStr)
		if err != nil {
			script.Fatalf("write-event: invalid current user SID %q: %v", sidStr, err)
		}
	}

	if err := reportEvent(source, eventType, eventID, sid, msg); err != nil {
		script.Fatalf("write-event: %v", err)
	}
}

// cmdWriteMultilineEvent writes the specific multiline message used by the
// test_multiline_events test (contains newlines and control characters).
func cmdWriteMultilineEvent(script *testscript.TestScript, neg bool, args []string) {
	source := script.Getenv("APP_SOURCE")
	msg := "\nA trusted logon process has been registered with the Local Security Authority.\n" +
		"This logon process will be trusted to submit logon requests.\n\nSubject:\n\n" +
		"Security ID:  SYSTEM\nAccount Name:  MS4\x1e$\nAccount Domain:  WORKGROUP\n" +
		"Logon ID:  0x3e7\nLogon Process Name:  IKE"
	if err := reportEvent(source, windows.EVENTLOG_INFORMATION_TYPE, 10, nil, msg); err != nil {
		script.Fatalf("write-multiline-event: %v", err)
	}
}

// cmdClearEventLog implements: clear-event-log
func cmdClearEventLog(script *testscript.TestScript, neg bool, args []string) {
	provider := script.Getenv("PROVIDER")
	if err := wineventlog.EvtClearLog(wineventlog.NilHandle, provider, ""); err != nil {
		script.Fatalf("clear-event-log: %v", err)
	}
}

// cmdCheckEventCount implements: check-event-count <dir> <N>
func cmdCheckEventCount(script *testscript.TestScript, neg bool, args []string) {
	if len(args) != 2 {
		script.Fatalf("usage: check-event-count <dir> <N>")
	}
	dir := script.MkAbs(args[0])
	want, err := strconv.Atoi(args[1])
	if err != nil {
		script.Fatalf("check-event-count: invalid count %q: %v", args[1], err)
	}

	events := readNDJSON(script, dir)
	if neg {
		if len(events) == want {
			script.Fatalf("check-event-count: got %d events, did not want %d", len(events), want)
		}
		return
	}
	if len(events) != want {
		script.Fatalf("check-event-count: got %d events, want %d", len(events), want)
	}
}

// cmdCheckEventField implements: check-event-field <dir> <index> <field> <value>
func cmdCheckEventField(script *testscript.TestScript, neg bool, args []string) {
	if len(args) != 4 {
		script.Fatalf("usage: check-event-field <dir> <index> <field> <value>")
	}
	dir := script.MkAbs(args[0])
	idx, err := strconv.Atoi(args[1])
	if err != nil {
		script.Fatalf("check-event-field: invalid index %q: %v", args[1], err)
	}
	field := args[2]
	want := args[3]

	events := readNDJSON(script, dir)
	if idx < 0 || idx >= len(events) {
		script.Fatalf("check-event-field: index %d out of range (have %d events)", idx, len(events))
	}

	got, ok := getField(events[idx], field)
	gotStr := fmt.Sprintf("%v", got)

	if neg {
		if ok && gotStr == want {
			script.Fatalf("check-event-field: event[%d].%s = %q, should not equal %q", idx, field, gotStr, want)
		}
		return
	}
	if !ok {
		script.Fatalf("check-event-field: event[%d] missing field %q", idx, field)
	}
	if gotStr != want {
		script.Fatalf("check-event-field: event[%d].%s = %q, want %q", idx, field, gotStr, want)
	}
}

// cmdCheckEventFieldExists implements: check-event-field-exists <dir> <index> <field>
func cmdCheckEventFieldExists(script *testscript.TestScript, neg bool, args []string) {
	if len(args) != 3 {
		script.Fatalf("usage: check-event-field-exists <dir> <index> <field>")
	}
	dir := script.MkAbs(args[0])
	idx, err := strconv.Atoi(args[1])
	if err != nil {
		script.Fatalf("check-event-field-exists: invalid index %q: %v", args[1], err)
	}
	field := args[2]

	events := readNDJSON(script, dir)
	if idx < 0 || idx >= len(events) {
		script.Fatalf("check-event-field-exists: index %d out of range (have %d events)", idx, len(events))
	}

	_, ok := getField(events[idx], field)
	if neg {
		if ok {
			script.Fatalf("check-event-field-exists: event[%d].%s exists, but should not", idx, field)
		}
		return
	}
	if !ok {
		script.Fatalf("check-event-field-exists: event[%d] missing field %q", idx, field)
	}
}

// cmdCheckEventFieldAbsent implements: check-event-field-absent <dir> <index> <field>
func cmdCheckEventFieldAbsent(script *testscript.TestScript, neg bool, args []string) {
	if len(args) != 3 {
		script.Fatalf("usage: check-event-field-absent <dir> <index> <field>")
	}
	dir := script.MkAbs(args[0])
	idx, err := strconv.Atoi(args[1])
	if err != nil {
		script.Fatalf("check-event-field-absent: invalid index %q: %v", args[1], err)
	}
	field := args[2]

	events := readNDJSON(script, dir)
	if idx < 0 || idx >= len(events) {
		script.Fatalf("check-event-field-absent: index %d out of range (have %d events)", idx, len(events))
	}

	if _, ok := getField(events[idx], field); ok {
		script.Fatalf("check-event-field-absent: event[%d].%s exists, but should not", idx, field)
	}
}

// cmdWaitForEventCount implements: wait-for-event-count <dir> <count> [<timeout>]
// Polls the output directory until at least <count> events appear, or times out.
func cmdWaitForEventCount(script *testscript.TestScript, neg bool, args []string) {
	if len(args) < 2 || len(args) > 3 {
		script.Fatalf("usage: wait-for-event-count <dir> <count> [<timeout>]")
	}
	dir := script.MkAbs(args[0])
	want, err := strconv.Atoi(args[1])
	if err != nil {
		script.Fatalf("wait-for-event-count: invalid count %q: %v", args[1], err)
	}
	timeout := 30 * time.Second
	if len(args) == 3 {
		timeout, err = time.ParseDuration(args[2])
		if err != nil {
			script.Fatalf("wait-for-event-count: invalid timeout %q: %v", args[2], err)
		}
	}
	deadline := time.Now().Add(timeout)
	for {
		events := readNDJSON(script, dir)
		if len(events) >= want {
			return
		}
		if time.Now().After(deadline) {
			script.Fatalf("wait-for-event-count: timed out after %v; got %d events, want %d",
				timeout, len(events), want)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// cmdWaitForEventLog implements: wait-for-event-log <count> [<timeout>]
// Polls the Windows event log for $PROVIDER until at least <count> events
// are visible, or times out. This ensures events written by write-event are
// committed and readable before winlogbeat starts.
func cmdWaitForEventLog(script *testscript.TestScript, neg bool, args []string) {
	if len(args) < 1 || len(args) > 2 {
		script.Fatalf("usage: wait-for-event-log <count> [<timeout>]")
	}
	want, err := strconv.Atoi(args[0])
	if err != nil {
		script.Fatalf("wait-for-event-log: invalid count %q: %v", args[0], err)
	}
	timeout := 30 * time.Second
	if len(args) == 2 {
		timeout, err = time.ParseDuration(args[1])
		if err != nil {
			script.Fatalf("wait-for-event-log: invalid timeout %q: %v", args[1], err)
		}
	}

	provider := script.Getenv("PROVIDER")
	if provider == "" {
		script.Fatalf("wait-for-event-log: $PROVIDER not set")
	}

	deadline := time.Now().Add(timeout)
	for {
		n, err := countEventLogRecords(provider)
		if err != nil {
			script.Fatalf("wait-for-event-log: %v", err)
		}
		if n >= want {
			return
		}
		if time.Now().After(deadline) {
			script.Fatalf("wait-for-event-log: timed out after %v; got %d events in %q, want %d",
				timeout, n, provider, want)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// countEventLogRecords queries the Windows event log channel and returns the
// number of event records it contains.
func countEventLogRecords(channel string) (int, error) {
	h, err := wineventlog.EvtQuery(wineventlog.NilHandle, channel, "", wineventlog.EvtQueryChannelPath|wineventlog.EvtQueryForwardDirection)
	if err != nil {
		return 0, fmt.Errorf("EvtQuery(%q): %w", channel, err)
	}
	defer wineventlog.Close(h)

	var count int
	for {
		handles, err := wineventlog.EventHandles(h, 100)
		if err != nil {
			if err == wineventlog.ERROR_NO_MORE_ITEMS { //nolint:errorlint // errno comparison
				break
			}
			return 0, fmt.Errorf("EventHandles: %w", err)
		}
		count += len(handles)
		for _, eh := range handles {
			wineventlog.Close(eh)
		}
	}
	return count, nil
}

// cmdSleep implements: sleep <duration>
func cmdSleep(script *testscript.TestScript, neg bool, args []string) {
	if len(args) != 1 {
		script.Fatalf("usage: sleep <duration>")
	}
	d, err := time.ParseDuration(args[0])
	if err != nil {
		script.Fatalf("sleep: invalid duration %q: %v", args[0], err)
	}
	time.Sleep(d)
}

// getCurrentUserSID returns the SID string for the current process user.
func getCurrentUserSID() (string, error) {
	var token windows.Token
	err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token)
	if err != nil {
		return "", fmt.Errorf("OpenProcessToken: %w", err)
	}
	defer token.Close()

	tokenUser, err := token.GetTokenUser()
	if err != nil {
		return "", fmt.Errorf("GetTokenUser: %w", err)
	}
	return tokenUser.User.Sid.String(), nil
}

// cmdCheckEventFieldContains implements: check-event-field-contains <dir> <index> <field> <substring>
// It converts the field value to a string and checks whether it contains the substring.
func cmdCheckEventFieldContains(script *testscript.TestScript, neg bool, args []string) {
	if len(args) != 4 {
		script.Fatalf("usage: check-event-field-contains <dir> <index> <field> <substring>")
	}
	dir := script.MkAbs(args[0])
	idx, err := strconv.Atoi(args[1])
	if err != nil {
		script.Fatalf("check-event-field-contains: invalid index %q: %v", args[1], err)
	}
	field := args[2]
	want := args[3]

	events := readNDJSON(script, dir)
	if idx < 0 || idx >= len(events) {
		script.Fatalf("check-event-field-contains: index %d out of range (have %d events)", idx, len(events))
	}

	got, ok := getField(events[idx], field)
	if !ok {
		if neg {
			return
		}
		script.Fatalf("check-event-field-contains: event[%d] missing field %q", idx, field)
	}

	gotStr := fmt.Sprintf("%v", got)
	contains := strings.Contains(gotStr, want)
	if neg {
		if contains {
			script.Fatalf("check-event-field-contains: event[%d].%s = %q, should not contain %q", idx, field, gotStr, want)
		}
		return
	}
	if !contains {
		script.Fatalf("check-event-field-contains: event[%d].%s = %q, want substring %q", idx, field, gotStr, want)
	}
}

// reportEvent writes a single event to the Windows event log using the
// low-level ReportEvent syscall, supporting custom event IDs and SIDs.
func reportEvent(source string, eventType uint16, eventID uint32, sid *windows.SID, msg string) error {
	sourcePtr, err := windows.UTF16PtrFromString(source)
	if err != nil {
		return fmt.Errorf("UTF16PtrFromString(%q): %w", source, err)
	}
	h, err := windows.RegisterEventSource(nil, sourcePtr)
	if err != nil {
		return fmt.Errorf("RegisterEventSource(%q): %w", source, err)
	}
	defer windows.DeregisterEventSource(h) //nolint:errcheck // best-effort cleanup

	msgPtr, err := windows.UTF16PtrFromString(msg)
	if err != nil {
		return fmt.Errorf("UTF16PtrFromString(msg): %w", err)
	}

	var sidPtr uintptr
	if sid != nil {
		sidPtr = uintptr(unsafe.Pointer(sid))
	}

	deadline := time.Now().Add(10 * time.Second)
	for {
		err = windows.ReportEvent(h, eventType, 0, eventID, sidPtr, 1, 0, &msgPtr, nil)
		if err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("ReportEvent: %w", err)
		}
	}
}

// readNDJSON reads all *.ndjson files in dir and returns the parsed events.
func readNDJSON(script *testscript.TestScript, dir string) []map[string]interface{} {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		script.Fatalf("readNDJSON: ReadDir(%q): %v", dir, err)
	}

	var events []map[string]interface{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".ndjson") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			script.Fatalf("readNDJSON: ReadFile: %v", err)
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			var evt map[string]interface{}
			if err := json.Unmarshal([]byte(line), &evt); err != nil {
				script.Fatalf("readNDJSON: Unmarshal: %v\nline: %s", err, line)
			}
			events = append(events, evt)
		}
	}
	return events
}

// getField retrieves a value from a nested map using dot-separated field names.
func getField(evt map[string]interface{}, field string) (interface{}, bool) {
	parts := strings.SplitN(field, ".", 2)
	val, ok := evt[parts[0]]
	if !ok {
		return nil, false
	}
	if len(parts) == 1 {
		return val, true
	}
	nested, ok := val.(map[string]interface{})
	if !ok {
		return nil, false
	}
	return getField(nested, parts[1])
}

// installAsEventCreate registers an event log source backed by EventCreate.exe,
// which supports event IDs 1–1000.
func installAsEventCreate(provider, src string) (alreadyExisted bool, _ error) {
	eventLogKey, err := registry.OpenKey(registry.LOCAL_MACHINE, eventLogKeyName, registry.CREATE_SUB_KEY)
	if err != nil {
		return false, err
	}
	defer eventLogKey.Close()

	pk, _, err := registry.CreateKey(eventLogKey, provider, registry.SET_VALUE)
	if err != nil {
		return false, err
	}
	defer pk.Close()

	sk, alreadyExist, err := registry.CreateKey(pk, src, registry.SET_VALUE)
	if err != nil {
		return false, err
	}
	defer sk.Close()
	if alreadyExist {
		return true, nil
	}

	if err = sk.SetDWordValue("CustomSource", 1); err != nil {
		return false, err
	}
	if err = sk.SetExpandStringValue("EventMessageFile", "%SystemRoot%\\System32\\EventCreate.exe"); err != nil {
		return false, err
	}
	typesSupported := uint32(windows.EVENTLOG_ERROR_TYPE | windows.EVENTLOG_WARNING_TYPE | windows.EVENTLOG_INFORMATION_TYPE)
	if err = sk.SetDWordValue("TypesSupported", typesSupported); err != nil {
		return false, err
	}
	return false, nil
}

// removeSource deletes the registry sub-key for an event logging source.
func removeSource(provider, src string) error {
	pk, err := registry.OpenKey(registry.LOCAL_MACHINE,
		fmt.Sprintf("%s\\%s", eventLogKeyName, provider),
		registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer pk.Close()
	return registry.DeleteKey(pk, src)
}

// removeProvider deletes the registry key for an event logging provider.
func removeProvider(provider string) error {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, eventLogKeyName, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return registry.DeleteKey(k, provider)
}
