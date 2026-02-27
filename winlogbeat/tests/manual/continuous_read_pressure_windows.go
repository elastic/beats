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

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const eventLogKeyName = `SYSTEM\CurrentControlSet\Services\EventLog`

func main() {
	var (
		provider       = flag.String("provider", "WinlogbeatPressure", "Event Log channel/provider name")
		source         = flag.String("source", "WinlogbeatPressureSource", "Event Log source name under provider")
		duration       = flag.Duration("duration", 5*time.Minute, "How long to emit events")
		rate           = flag.Int("rate", 25, "Events per second")
		messageSize    = flag.Int("message-size", 12000, "Message size in bytes/chars")
		eventID        = flag.Uint("event-id", 10, "Windows event ID")
		outputDir      = flag.String("output-dir", "", "Optional Winlogbeat output.file path for verification")
		verifyTimeout  = flag.Duration("verify-timeout", 2*time.Minute, "Max wait before reading output-dir")
		cleanup        = flag.Bool("cleanup", false, "Remove source/provider registry keys after run")
		reportProgress = flag.Duration("progress-every", 10*time.Second, "Progress print interval")
		runWinlogbeat  = flag.Bool("run-winlogbeat", false, "Start winlogbeat process automatically for this run")
		winlogbeatBin  = flag.String("winlogbeat-bin", "", "Path to winlogbeat binary (required when -run-winlogbeat)")
		winlogbeatCfg  = flag.String("winlogbeat-config", "winlogbeat/tests/manual/winlogbeat-continuous-pressure.yml", "Winlogbeat config path")
		winlogbeatData = flag.String("winlogbeat-data", "C:/tmp/winlogbeat-pressure-data", "Winlogbeat data path")
		startDelay     = flag.Duration("start-delay", 3*time.Second, "Delay after starting winlogbeat before writing")
		stopTimeout    = flag.Duration("stop-timeout", 20*time.Second, "Graceful stop timeout for auto-started winlogbeat")
	)
	flag.Parse()

	if *rate <= 0 {
		fatalf("invalid -rate: %d", *rate)
	}
	if *messageSize < 64 {
		fatalf("invalid -message-size: %d (must be >= 64)", *messageSize)
	}
	if *duration <= 0 {
		fatalf("invalid -duration: %s", *duration)
	}
	if *runWinlogbeat && *winlogbeatBin == "" {
		fatalf("-winlogbeat-bin is required when -run-winlogbeat is set")
	}
	if *outputDir == "" {
		*outputDir = "C:/tmp/winlogbeat-pressure-output"
	}

	if _, err := installAsEventCreate(*provider, *source); err != nil {
		fatalf("failed to register source/provider: %v", err)
	}
	if *cleanup {
		defer func() {
			_ = removeSource(*provider, *source)
			_ = removeProvider(*provider)
		}()
	}

	sid, err := getCurrentUserSID()
	if err != nil {
		fatalf("failed to resolve current user SID: %v", err)
	}

	var winlogbeatCmd *exec.Cmd
	if *runWinlogbeat {
		var err error
		winlogbeatCmd, err = startWinlogbeat(*winlogbeatBin, *winlogbeatCfg, *winlogbeatData, *provider, *outputDir)
		if err != nil {
			fatalf("failed to start winlogbeat: %v", err)
		}
		fmt.Printf("started_winlogbeat pid=%d\n", winlogbeatCmd.Process.Pid)
		time.Sleep(*startDelay)
	}

	runID := time.Now().UTC().Format("20060102T150405.000Z")
	fmt.Printf("run_id=%s provider=%s source=%s duration=%s rate=%d message_size=%d\n",
		runID, *provider, *source, duration.String(), *rate, *messageSize)

	start := time.Now()
	deadline := start.Add(*duration)
	progressAt := start.Add(*reportProgress)
	interval := time.Second / time.Duration(*rate)
	if interval < time.Millisecond {
		interval = time.Millisecond
	}

	var sent int
	for now := time.Now(); now.Before(deadline); now = time.Now() {
		msg := buildMessage(runID, sent, *messageSize)
		if err := reportEvent(*source, windows.EVENTLOG_INFORMATION_TYPE, uint32(*eventID), sid, msg); err != nil {
			fatalf("failed writing event %d: %v", sent, err)
		}
		sent++

		if now.After(progressAt) {
			fmt.Printf("progress sent=%d elapsed=%s\n", sent, time.Since(start).Round(time.Second))
			progressAt = now.Add(*reportProgress)
		}
		time.Sleep(interval)
	}

	fmt.Printf("writer_done sent=%d elapsed=%s\n", sent, time.Since(start).Round(time.Second))

	if winlogbeatCmd != nil {
		if err := stopProcess(winlogbeatCmd, *stopTimeout); err != nil {
			fatalf("failed to stop winlogbeat cleanly: %v", err)
		}
		fmt.Println("winlogbeat_stopped")
	}

	fmt.Printf("waiting %s before reading output from %s\n", verifyTimeout.String(), *outputDir)
	time.Sleep(*verifyTimeout)

	seen, err := readRunSequences(*outputDir, runID)
	if err != nil {
		fatalf("failed reading output-dir: %v", err)
	}

	reportSequences(sent, seen)
}

func startWinlogbeat(bin, cfg, dataPath, provider, outputDir string) (*exec.Cmd, error) {
	cmd := exec.CommandContext(context.Background(), bin, "-e", "-c", cfg, "--path.data", dataPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		"PRESSURE_PROVIDER="+provider,
		"PRESSURE_OUTPUT_DIR="+outputDir,
	)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}

func stopProcess(cmd *exec.Cmd, timeout time.Duration) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	_ = cmd.Process.Signal(os.Interrupt)
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		if err == nil {
			return nil
		}
		// If process already exited due to signal, do not fail the run.
		if errorsIsExit(err) {
			return nil
		}
		return err
	case <-time.After(timeout):
		_ = cmd.Process.Kill()
		<-done
		return fmt.Errorf("process did not exit within %s", timeout)
	}
}

func errorsIsExit(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*exec.ExitError)
	return ok
}

func buildMessage(runID string, seq, size int) string {
	msg := fmt.Sprintf("pressure run=%s seq=%06d ", runID, seq)
	if len(msg) < size {
		msg += strings.Repeat("X", size-len(msg))
	}
	return msg
}

func reportSequences(expected int, seen map[int]struct{}) {
	if expected == 0 {
		fmt.Println("no events expected")
		return
	}

	missing := make([]int, 0)
	for i := 0; i < expected; i++ {
		if _, ok := seen[i]; !ok {
			missing = append(missing, i)
		}
	}

	fmt.Printf("verification expected=%d seen=%d missing=%d\n", expected, len(seen), len(missing))
	if len(missing) == 0 {
		fmt.Println("verification_result=PASS (no sequence gaps)")
		return
	}

	sort.Ints(missing)
	preview := missing
	if len(preview) > 20 {
		preview = preview[:20]
	}
	fmt.Printf("verification_result=FAIL first_missing=%v\n", preview)
}

func readRunSequences(outputDir, runID string) (map[int]struct{}, error) {
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return nil, err
	}

	pattern := regexp.MustCompile(`pressure run=` + regexp.QuoteMeta(runID) + ` seq=(\d{6})`)
	seen := make(map[int]struct{})

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".ndjson") {
			continue
		}

		filePath := filepath.Join(outputDir, entry.Name())
		f, err := os.Open(filePath)
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", filePath, err)
		}

		scanner := bufio.NewScanner(f)
		// Increase scanner buffer so very large messages can be parsed.
		buf := make([]byte, 0, 1024*1024)
		scanner.Buffer(buf, 16*1024*1024)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			var evt map[string]interface{}
			if err := json.Unmarshal([]byte(line), &evt); err != nil {
				continue
			}
			msg, ok := getStringField(evt, "message")
			if !ok {
				continue
			}
			matches := pattern.FindStringSubmatch(msg)
			if len(matches) != 2 {
				continue
			}
			var seq int
			if _, err := fmt.Sscanf(matches[1], "%d", &seq); err != nil {
				continue
			}
			seen[seq] = struct{}{}
		}
		if err := scanner.Err(); err != nil {
			_ = f.Close()
			return nil, fmt.Errorf("scan %s: %w", filePath, err)
		}
		_ = f.Close()
	}

	return seen, nil
}

func getStringField(evt map[string]interface{}, key string) (string, bool) {
	v, ok := evt[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

func getCurrentUserSID() (*windows.SID, error) {
	var token windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token); err != nil {
		return nil, fmt.Errorf("OpenProcessToken: %w", err)
	}
	defer token.Close()

	tokenUser, err := token.GetTokenUser()
	if err != nil {
		return nil, fmt.Errorf("GetTokenUser: %w", err)
	}
	return tokenUser.User.Sid, nil
}

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

// installAsEventCreate registers an event log source backed by EventCreate.exe.
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

func removeProvider(provider string) error {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, eventLogKeyName, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return registry.DeleteKey(k, provider)
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
