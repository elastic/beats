// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

//nolint:gosec // This file is used for testing ETW functionality and does not handle sensitive data.
package etw

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Test Provider Constants
const (
	testProviderName = "Beats-ETW-TestProvider"
	testProviderGUID = "{06f0e00e-422a-4987-922b-2ec13e281ea4}"
)

// ETW Event Constants
const (
	// Event IDs
	allDataTypesEventID = 1
	complexDataEventID  = 2

	// Channel IDs
	CHANNEL_OPERATIONAL = 16

	// Keywords
	READ_KEYWORD        = 0x1
	WRITE_KEYWORD       = 0x2
	LOCAL_KEYWORD       = 0x4
	REMOTE_KEYWORD      = 0x8
	PERFORMANCE_KEYWORD = 0x10
	SECURITY_KEYWORD    = 0x20
	NETWORK_KEYWORD     = 0x40
	STORAGE_KEYWORD     = 0x80
	DEBUG_KEYWORD       = 0x100
	AUDIT_KEYWORD       = 0x200
	ERROR_KEYWORD       = 0x400
	CRITICAL_KEYWORD    = 0x800
	SESSION_KEYWORD     = 0x1000
	PROCESS_KEYWORD     = 0x2000
	THREAD_KEYWORD      = 0x4000
	MEMORY_KEYWORD      = 0x8000

	// Keyword combinations
	LOCAL_READ_KEYWORDS   = LOCAL_KEYWORD | READ_KEYWORD
	REMOTE_WRITE_KEYWORDS = REMOTE_KEYWORD | WRITE_KEYWORD

	// Tasks
	TASK_DISCONNECT = 1
	TASK_CONNECT    = 2

	// Opcodes
	OPCODE_STOP       = 20
	OPCODE_INITIALIZE = 12

	// Levels
	LEVEL_ERROR       = 2
	LEVEL_INFORMATION = 4
)

// Test flags
var (
	regenerateETL = flag.Bool("regenerate-etl", false, "regenerate the ETL test data file")
)

// Common test setup and utilities

// setupProviderManager initializes provider manager with cleanup
func setupProviderManager(t testing.TB) {
	pm, err := NewTestProviderManager()
	if err != nil {
		t.Fatalf("Failed to create test provider manager: %v", err)
		return
	}

	if err := pm.ensureProviderRegistered(); err != nil {
		t.Fatalf("Failed to ensure provider is registered: %v", err)
		return
	}

	t.Cleanup(func() {
		if err := pm.cleanup(); err != nil {
			t.Logf("Failed to clean up provider: %v", err)
		}
	})
}

// TestProviderManager handles registration/unregistration of the test provider
type TestProviderManager struct {
	testDataDir  string
	manifestPath string
	dllPath      string
	scriptPath   string
}

// NewTestProviderManager creates a new provider manager
func NewTestProviderManager() (*TestProviderManager, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return nil, fmt.Errorf("failed to get current file path")
	}

	testDataDir := filepath.Join(filepath.Dir(currentFile), "testdata")

	return &TestProviderManager{
		testDataDir:  testDataDir,
		manifestPath: filepath.Join(testDataDir, "sample.man"),
		dllPath:      filepath.Join(testDataDir, "sample.dll"),
		scriptPath:   filepath.Join(testDataDir, "manage-etw-provider.ps1"),
	}, nil
}

// Provider management methods
func (pm *TestProviderManager) checkProviderStatus() (bool, error) {
	cmd := exec.CommandContext(context.Background(), "powershell.exe", "-ExecutionPolicy", "Bypass", "-File", pm.scriptPath, "-Action", "Status")
	cmd.Dir = pm.testDataDir

	err := cmd.Run()
	if err != nil {
		errExit := &exec.ExitError{}
		if errors.As(err, &errExit) && errExit.ExitCode() == 1 {
			return false, nil
		}
		return false, fmt.Errorf("failed to check provider status: %w", err)
	}
	return true, nil
}

func (pm *TestProviderManager) registerProvider() error {
	cmd := exec.CommandContext(context.Background(), "powershell.exe", "-ExecutionPolicy", "Bypass", "-File", pm.scriptPath, "-Action", "Register")
	cmd.Dir = pm.testDataDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to register provider: %w, output: %s", err, string(output))
	}
	return nil
}

func (pm *TestProviderManager) unregisterProvider() error {
	cmd := exec.CommandContext(context.Background(), "powershell.exe", "-ExecutionPolicy", "Bypass", "-File", pm.scriptPath, "-Action", "Unregister")
	cmd.Dir = pm.testDataDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to unregister provider: %w, output: %s", err, string(output))
	}
	return nil
}

func (pm *TestProviderManager) ensureProviderRegistered() error {
	registered, err := pm.checkProviderStatus()
	if err != nil {
		return fmt.Errorf("failed to check provider status: %w", err)
	}

	if registered {
		return nil
	}
	return pm.registerProvider()
}

func (pm *TestProviderManager) cleanup() error {
	return pm.unregisterProvider()
}

// Windows API declarations
var (
	eventRegister   = advapi32.NewProc("EventRegister")
	eventUnregister = advapi32.NewProc("EventUnregister")
	eventWrite      = advapi32.NewProc("EventWrite")
)

// EventDataDescriptor structure for passing event data
type EventDataDescriptor struct {
	Ptr      uint64
	Size     uint32
	Reserved uint32
}

// Windows API wrappers
func _EventRegister(providerId *windows.GUID, enableCallback uintptr, callbackContext uintptr, regHandle *windows.Handle) error {
	ret, _, _ := eventRegister.Call(
		uintptr(unsafe.Pointer(providerId)),
		enableCallback,
		callbackContext,
		uintptr(unsafe.Pointer(regHandle)),
	)
	if ret != 0 {
		return syscall.Errno(ret)
	}
	return nil
}

func _EventUnregister(regHandle windows.Handle) error {
	ret, _, _ := eventUnregister.Call(uintptr(regHandle))
	if ret != 0 {
		return syscall.Errno(ret)
	}
	return nil
}

// TestRegenerateTestdataETL regenerates the ETL file in the testdata directory
func TestRegenerateTestdataETL(t *testing.T) {
	if !*regenerateETL {
		t.Skip("Skipping ETL regeneration test. Use -regenerate-etl flag to run.")
	}
	setupProviderManager(t)
	const (
		eventCount = 30
	)

	etlPath := filepath.Join("testdata", "sample-test-events.etl")

	// Clean up existing file
	_ = os.Remove(etlPath)

	sessionName := uniqueSessionName("BeatsETWTestSession")
	// Start ETW session
	if err := startETWSession(sessionName, etlPath); err != nil {
		t.Fatalf("Failed to start ETW session: %v", err)
	}
	t.Cleanup(func() { _ = stopETWSession(sessionName) })

	// Generate events using the generator
	generator, err := NewETWEventGenerator()
	if err != nil {
		t.Fatalf("Failed to create event generator: %v", err)
	}
	t.Cleanup(func() { generator.close() })

	if err := generator.generateEvents(eventCount, time.Millisecond); err != nil {
		t.Fatalf("Failed to generate events: %v", err)
	}

	// Wait for events to be flushed
	time.Sleep(5 * time.Second)

	// Verify file exists
	if _, err := os.Stat(etlPath); err != nil {
		t.Fatalf("ETL file was not created: %v", err)
	}

	t.Logf("ETL file created: %s", etlPath)
}

// Helper functions for ETL session management
func startETWSession(sessionName, etlPath string) error {
	_ = stopETWSession(sessionName)
	cmd := exec.CommandContext(context.Background(), "logman", "start", sessionName, "-ets",
		"-o", etlPath,
		"-p", testProviderGUID, "0xFFFFFFFFFFFFFFFF", "0xFF")
	return cmd.Run()
}

func stopETWSession(sessionName string) error {
	cmd := exec.CommandContext(context.Background(), "logman", "stop", sessionName, "-ets")
	return cmd.Run()
}

// ETWEventGenerator generates random ETW events for the test provider
// covering both ALL_DATA_TYPES_EVENT and COMPLEX_DATA_EVENT as defined in sample.man.
type ETWEventGenerator struct {
	providerGUID windows.GUID
	regHandle    windows.Handle
}

// NewETWEventGenerator creates and registers a new ETWEventGenerator
func NewETWEventGenerator() (*ETWEventGenerator, error) {
	guid, err := windows.GUIDFromString(testProviderGUID)
	if err != nil {
		return nil, err
	}
	var regHandle windows.Handle
	err = _EventRegister(&guid, 0, 0, &regHandle)
	if err != nil {
		return nil, err
	}
	return &ETWEventGenerator{
		providerGUID: guid,
		regHandle:    regHandle,
	}, nil
}

// Close unregisters the ETW provider
func (g *ETWEventGenerator) close() error {
	return _EventUnregister(g.regHandle)
}

// generateEvents generates n random events, alternating between the two event types
func (g *ETWEventGenerator) generateEvents(n int, wait time.Duration) error {
	for i := 0; i < n; i++ {
		if i%2 == 0 {
			if err := g.writeAllDataTypesEvent(); err != nil {
				return err
			}
		} else {
			if err := g.writeComplexDataEvent(); err != nil {
				return err
			}
		}
	}
	time.Sleep(wait)
	return nil
}

// StartGenerating generates events in batches with a delay between batches until stop is signaled.
func (g *ETWEventGenerator) StartGenerating(eventsPerBatch int, batchInterval time.Duration, stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)
	batch := 0
	for {
		select {
		case <-stop:
			return
		default:
			_ = g.generateEvents(eventsPerBatch, batchInterval)
			batch++
		}
	}
}

func (g *ETWEventGenerator) writeEvent(eventID int, desc []EventDataDescriptor) error {
	d := EventDescriptor{
		Id:      uint16(eventID),
		Version: 0,
		Channel: CHANNEL_OPERATIONAL,
		Level:   LEVEL_INFORMATION,
		Opcode:  OPCODE_STOP,
		Task:    TASK_CONNECT,
		Keyword: LOCAL_READ_KEYWORDS,
	}
	if eventID == complexDataEventID {
		d.Level = LEVEL_ERROR
		d.Opcode = OPCODE_INITIALIZE
		d.Task = TASK_DISCONNECT
		d.Keyword = REMOTE_WRITE_KEYWORDS
	}
	ret, _, _ := eventWrite.Call(
		uintptr(g.regHandle),
		uintptr(unsafe.Pointer(&d)),
		uintptr(uint32(len(desc))),
		uintptr(unsafe.Pointer(&desc[0])),
	)
	if ret != 0 {
		return fmt.Errorf("EventWrite failed: %w", syscall.Errno(ret))
	}
	return nil
}

type simpleRand struct{ seed uint64 }

func (r *simpleRand) next() uint64 { r.seed = r.seed*6364136223846793005 + 1; return r.seed }

func (r *simpleRand) randBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(r.next() >> 8)
	}
	return b
}

// Write ALL_DATA_TYPES_EVENT (event ID 1)
func (g *ETWEventGenerator) writeAllDataTypesEvent() error {
	r := &simpleRand{seed: uint64(uintptr(unsafe.Pointer(g))) ^ uint64(time.Now().UnixNano())}
	// Predefined sets for more realistic data
	stringSet := []string{"alpha", "bravo", "charlie", "delta", "echo"}

	unicodeStr := stringSet[int(r.next()%uint64(len(stringSet)))]
	ansiStr := stringSet[int(r.next()%uint64(len(stringSet)))]
	int8v := int8(r.next())
	uint8v := uint8(r.next())
	int16v := int16(r.next())
	uint16v := uint16(r.next())
	int32v := int32(r.next())
	uint32v := uint32(r.next())
	int64v := int64(r.next())
	uint64v := r.next()
	floatv := float32(r.next()%10000)/100 + float32(r.next()%100)/100
	doublev := float64(r.next()%1000000)/1000 + float64(r.next()%1000)/1000
	boolv := int32(r.next() % 2)
	binSize := uint32(7 + r.next()%8)
	binVal := r.randBytes(int(binSize))
	guid, _ := windows.GenerateGUID()
	ptrVal := uintptr(r.next() & 0xFFFFFFFF)
	// Generate a valid FILETIME (random time in the last 10 years)
	now := time.Now()
	randomDuration := time.Duration(r.next()%uint64(10*365*24*60*60)) * time.Second
	randomTime := now.Add(-randomDuration)
	fileTime := uint64(randomTime.UnixNano()/100 + 116444736000000000)
	// Generate a valid SYSTEMTIME
	var sysTime windows.Systemtime
	sysTime.Year = uint16(randomTime.Year())
	sysTime.Month = uint16(randomTime.Month())
	sysTime.DayOfWeek = uint16(randomTime.Weekday())
	sysTime.Day = uint16(randomTime.Day())
	sysTime.Hour = uint16(randomTime.Hour())
	sysTime.Minute = uint16(randomTime.Minute())
	sysTime.Second = uint16(randomTime.Second())
	sysTime.Milliseconds = uint16(randomTime.Nanosecond() / 1e6)
	sid := []byte{0x01, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05, 0x15, 0x00, 0x00, 0x00,
		byte(r.next()), byte(r.next()), byte(r.next()), byte(r.next()),
		byte(r.next()), byte(r.next()), byte(r.next()), byte(r.next()),
		byte(r.next()), byte(r.next()), byte(r.next()), byte(r.next())}
	hex32 := int32(0xDEADBEEF ^ r.next())
	hex64 := int64(0xDEADBEEFDEADBEEF ^ r.next())

	var desc []EventDataDescriptor
	// UnicodeString
	utf16Str := windows.StringToUTF16(unicodeStr)
	desc = append(desc, newEventDataDescriptorPtr(unsafe.Pointer(&utf16Str[0]), len(utf16Str)*2))
	ansiBytes := append([]byte(ansiStr), 0)
	desc = append(desc, newEventDataDescriptorPtr(unsafe.Pointer(&ansiBytes[0]), len(ansiBytes)))
	desc = append(desc, newEventDataDescriptorPtr(unsafe.Pointer(&int8v), int(unsafe.Sizeof(int8v))))
	desc = append(desc, newEventDataDescriptorPtr(unsafe.Pointer(&uint8v), int(unsafe.Sizeof(uint8v))))
	desc = append(desc, newEventDataDescriptorPtr(unsafe.Pointer(&int16v), int(unsafe.Sizeof(int16v))))
	desc = append(desc, newEventDataDescriptorPtr(unsafe.Pointer(&uint16v), int(unsafe.Sizeof(uint16v))))
	desc = append(desc, newEventDataDescriptorPtr(unsafe.Pointer(&int32v), int(unsafe.Sizeof(int32v))))
	desc = append(desc, newEventDataDescriptorPtr(unsafe.Pointer(&uint32v), int(unsafe.Sizeof(uint32v))))
	desc = append(desc, newEventDataDescriptorPtr(unsafe.Pointer(&int64v), int(unsafe.Sizeof(int64v))))
	desc = append(desc, newEventDataDescriptorPtr(unsafe.Pointer(&uint64v), int(unsafe.Sizeof(uint64v))))
	desc = append(desc, newEventDataDescriptorPtr(unsafe.Pointer(&floatv), int(unsafe.Sizeof(floatv))))
	desc = append(desc, newEventDataDescriptorPtr(unsafe.Pointer(&doublev), int(unsafe.Sizeof(doublev))))
	desc = append(desc, newEventDataDescriptorPtr(unsafe.Pointer(&boolv), int(unsafe.Sizeof(boolv))))
	desc = append(desc, newEventDataDescriptorPtr(unsafe.Pointer(&binSize), int(unsafe.Sizeof(binSize))))
	desc = append(desc, newEventDataDescriptorPtr(unsafe.Pointer(&binVal[0]), len(binVal)))
	desc = append(desc, newEventDataDescriptorPtr(unsafe.Pointer(&guid), int(unsafe.Sizeof(guid))))
	desc = append(desc, newEventDataDescriptorPtr(unsafe.Pointer(&ptrVal), int(unsafe.Sizeof(ptrVal))))
	desc = append(desc, newEventDataDescriptorPtr(unsafe.Pointer(&fileTime), int(unsafe.Sizeof(fileTime))))
	desc = append(desc, newEventDataDescriptorPtr(unsafe.Pointer(&sysTime), int(unsafe.Sizeof(sysTime))))
	desc = append(desc, newEventDataDescriptorPtr(unsafe.Pointer(&sid[0]), len(sid)))
	desc = append(desc, newEventDataDescriptorPtr(unsafe.Pointer(&hex32), int(unsafe.Sizeof(hex32))))
	desc = append(desc, newEventDataDescriptorPtr(unsafe.Pointer(&hex64), int(unsafe.Sizeof(hex64))))

	runtime.KeepAlive(desc)
	return g.writeEvent(allDataTypesEventID, desc)
}

// Write COMPLEX_DATA_EVENT (event ID 2)
func (g *ETWEventGenerator) writeComplexDataEvent() error {
	r := &simpleRand{seed: uint64(uintptr(unsafe.Pointer(g))) ^ 0xDEADBEEF ^ uint64(time.Now().UnixNano())}
	stringSet := []string{
		"alpha",
		"bravo",
		"charlie",
		"delta",
		"echo",
	}
	filePathSet := []string{
		"C:/Windows/System32/drivers/etc/hosts",
		"C:/Program Files/Example/file.txt",
		"D:/Data/report.csv",
		"C:/Temp/test.log",
		"C:/Users/Public/Documents/readme.md",
	}

	transferName := stringSet[int(r.next()%uint64(len(stringSet)))]
	errorCode := int32(50000 + r.next()%15536)
	filesCount := uint16(2 + r.next()%3)
	files := make([]string, filesCount)
	for i := range files {
		files[i] = filePathSet[int(r.next()%uint64(len(filePathSet)))]
	}
	bufferSize := uint32(5 + r.next()%10)
	buffer := r.randBytes(int(bufferSize))
	certificate := r.randBytes(11)
	isLocal := int32(r.next() % 2)
	path := filePathSet[int(r.next()%uint64(len(filePathSet)))]
	valuesCount := uint16(2 + r.next()%3)
	values := make([]struct {
		Value uint16
		Name  string
	}, valuesCount)
	for i := range values {
		values[i].Value = uint16(r.next() % 1000)
		values[i].Name = stringSet[int(r.next()%uint64(len(stringSet)))]
	}
	desc := make([]EventDataDescriptor, 0, 20)
	utf16TransferName := windows.StringToUTF16(transferName)
	desc = append(desc, newEventDataDescriptorPtr(unsafe.Pointer(&utf16TransferName[0]), len(utf16TransferName)*2))
	desc = append(desc, newEventDataDescriptorPtr(unsafe.Pointer(&errorCode), int(unsafe.Sizeof(errorCode))))
	desc = append(desc, newEventDataDescriptorPtr(unsafe.Pointer(&filesCount), int(unsafe.Sizeof(filesCount))))
	var filesBlob bytes.Buffer
	for _, f := range files {
		utf16Bytes := windows.StringToUTF16(f)
		if err := binary.Write(&filesBlob, binary.LittleEndian, utf16Bytes); err != nil {
			return fmt.Errorf("failed to write file path %s: %w", f, err)
		}
	}
	packedFiles := filesBlob.Bytes()
	desc = append(desc, newEventDataDescriptorPtr(unsafe.Pointer(&packedFiles[0]), len(packedFiles)))
	desc = append(desc, newEventDataDescriptorPtr(unsafe.Pointer(&bufferSize), int(unsafe.Sizeof(bufferSize))))
	desc = append(desc, newEventDataDescriptorPtr(unsafe.Pointer(&buffer[0]), len(buffer)))
	desc = append(desc, newEventDataDescriptorPtr(unsafe.Pointer(&certificate[0]), len(certificate)))
	desc = append(desc, newEventDataDescriptorPtr(unsafe.Pointer(&isLocal), int(unsafe.Sizeof(isLocal))))
	utf16Path := windows.StringToUTF16(path)
	desc = append(desc, newEventDataDescriptorPtr(unsafe.Pointer(&utf16Path[0]), len(utf16Path)*2))
	desc = append(desc, newEventDataDescriptorPtr(unsafe.Pointer(&valuesCount), int(unsafe.Sizeof(valuesCount))))
	var valuesBlob bytes.Buffer
	for _, v := range values {
		if err := binary.Write(&valuesBlob, binary.LittleEndian, v.Value); err != nil {
			return fmt.Errorf("failed to write value %d: %w", v.Value, err)
		}
		utf16Slice := windows.StringToUTF16(v.Name)
		if err := binary.Write(&valuesBlob, binary.LittleEndian, utf16Slice); err != nil {
			return fmt.Errorf("failed to write name %s: %w", v.Name, err)
		}
	}
	packedValues := valuesBlob.Bytes()
	desc = append(desc, newEventDataDescriptorPtr(unsafe.Pointer(&packedValues[0]), len(packedValues)))
	runtime.KeepAlive(desc)
	return g.writeEvent(complexDataEventID, desc)
}

func newEventDataDescriptorPtr(ptr unsafe.Pointer, size int) EventDataDescriptor {
	return EventDataDescriptor{
		Ptr:  uint64(uintptr(ptr)),
		Size: uint32(size),
	}
}

// uniqueSessionName generates a unique session name for tests
func uniqueSessionName(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}
