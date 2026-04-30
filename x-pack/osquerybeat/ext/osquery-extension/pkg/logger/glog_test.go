// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package logger

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
)

// TestNew tests the logger constructor
func TestNew(t *testing.T) {
	tests := []struct {
		name      string
		verbose   bool
		wantLevel Level
	}{
		{
			name:      "verbose enabled",
			verbose:   true,
			wantLevel: LevelInfo,
		},
		{
			name:      "verbose disabled",
			verbose:   false,
			wantLevel: LevelWarning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			log := New(&buf, tt.verbose)

			if log == nil {
				t.Fatal("New() returned nil")
			}

			if log.minLevel != tt.wantLevel {
				t.Errorf("New() minLevel = %v, want %v", log.minLevel, tt.wantLevel)
			}

			if log.w != &buf {
				t.Error("New() writer not set correctly")
			}
		})
	}
}

// TestFormatLog tests the glog format
func TestFormatLog(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, true)

	// Test Info level
	formatted := log.formatLog(LevelInfo, "test message")

	// Verify format: I0314 15:24:36.123456 12345 file.go:123] test message
	// Accept both .go and .s files (runtime files may be .s)
	pattern := `^I\d{4} \d{2}:\d{2}:\d{2}\.\d{6} \d+ \w+\.(go|s):\d+\] test message\n$`
	matched, err := regexp.MatchString(pattern, formatted)
	if err != nil {
		t.Fatalf("regex error: %v", err)
	}
	if !matched {
		t.Errorf("formatLog() output doesn't match expected pattern.\nGot: %q\nPattern: %q", formatted, pattern)
	}

	// Verify it contains the message
	if !strings.Contains(formatted, "test message") {
		t.Errorf("formatLog() output doesn't contain message: %q", formatted)
	}

	// Verify it contains a filename (either .go or .s)
	if !strings.Contains(formatted, ".go:") && !strings.Contains(formatted, ".s:") {
		t.Errorf("formatLog() output doesn't contain filename: %q", formatted)
	}
}

// TestLevelCharacters tests that correct level characters are used
func TestLevelCharacters(t *testing.T) {
	tests := []struct {
		name     string
		level    Level
		wantChar byte
	}{
		{
			name:     "info level",
			level:    LevelInfo,
			wantChar: 'I',
		},
		{
			name:     "warning level",
			level:    LevelWarning,
			wantChar: 'W',
		},
		{
			name:     "error level",
			level:    LevelError,
			wantChar: 'E',
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			log := New(&buf, true)

			formatted := log.formatLog(tt.level, "test")

			if formatted[0] != tt.wantChar {
				t.Errorf("formatLog() level char = %c, want %c", formatted[0], tt.wantChar)
			}
		})
	}
}

// TestInfo tests Info logging
func TestInfo(t *testing.T) {
	tests := []struct {
		name      string
		verbose   bool
		message   string
		shouldLog bool
	}{
		{
			name:      "verbose enabled - should log",
			verbose:   true,
			message:   "info message",
			shouldLog: true,
		},
		{
			name:      "verbose disabled - should not log",
			verbose:   false,
			message:   "info message",
			shouldLog: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			log := New(&buf, tt.verbose)

			log.Info(tt.message)

			output := buf.String()
			if tt.shouldLog {
				if output == "" {
					t.Error("Info() didn't log when it should have")
				}
				if !strings.Contains(output, tt.message) {
					t.Errorf("Info() output doesn't contain message: %q", output)
				}
				if output[0] != 'I' {
					t.Errorf("Info() didn't use 'I' level character: %q", output)
				}
			} else {
				if output != "" {
					t.Errorf("Info() logged when it shouldn't have: %q", output)
				}
			}
		})
	}
}

// TestInfof tests formatted Info logging
func TestInfof(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, true)

	log.Infof("formatted %s %d", "message", 42)

	output := buf.String()
	if !strings.Contains(output, "formatted message 42") {
		t.Errorf("Infof() output doesn't contain formatted message: %q", output)
	}
	if output[0] != 'I' {
		t.Errorf("Infof() didn't use 'I' level character: %q", output)
	}
}

// TestWarning tests Warning logging
func TestWarning(t *testing.T) {
	tests := []struct {
		name    string
		verbose bool
		message string
	}{
		{
			name:    "verbose enabled",
			verbose: true,
			message: "warning message",
		},
		{
			name:    "verbose disabled",
			verbose: false,
			message: "warning message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			log := New(&buf, tt.verbose)

			log.Warning(tt.message)

			output := buf.String()
			if output == "" {
				t.Error("Warning() didn't log")
			}
			if !strings.Contains(output, tt.message) {
				t.Errorf("Warning() output doesn't contain message: %q", output)
			}
			if output[0] != 'W' {
				t.Errorf("Warning() didn't use 'W' level character: %q", output)
			}
		})
	}
}

// TestWarningf tests formatted Warning logging
func TestWarningf(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, true)

	log.Warningf("formatted %s %d", "warning", 99)

	output := buf.String()
	if !strings.Contains(output, "formatted warning 99") {
		t.Errorf("Warningf() output doesn't contain formatted message: %q", output)
	}
	if output[0] != 'W' {
		t.Errorf("Warningf() didn't use 'W' level character: %q", output)
	}
}

// TestError tests Error logging
func TestError(t *testing.T) {
	tests := []struct {
		name    string
		verbose bool
		message string
	}{
		{
			name:    "verbose enabled",
			verbose: true,
			message: "error message",
		},
		{
			name:    "verbose disabled",
			verbose: false,
			message: "error message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			log := New(&buf, tt.verbose)

			log.Error(tt.message)

			output := buf.String()
			if output == "" {
				t.Error("Error() didn't log")
			}
			if !strings.Contains(output, tt.message) {
				t.Errorf("Error() output doesn't contain message: %q", output)
			}
			if output[0] != 'E' {
				t.Errorf("Error() didn't use 'E' level character: %q", output)
			}
		})
	}
}

// TestErrorf tests formatted Error logging
func TestErrorf(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, true)

	log.Errorf("formatted %s %d", "error", 500)

	output := buf.String()
	if !strings.Contains(output, "formatted error 500") {
		t.Errorf("Errorf() output doesn't contain formatted message: %q", output)
	}
	if output[0] != 'E' {
		t.Errorf("Errorf() didn't use 'E' level character: %q", output)
	}
}

// TestFatal tests Fatal logging (without actually exiting)
func TestFatal(t *testing.T) {
	// We can't actually test os.Exit being called without using a subprocess
	// but we can test the message is logged
	var buf bytes.Buffer
	log := New(&buf, true)

	// Replace os.Exit with a no-op for testing
	oldOsExit := osExit
	defer func() { osExit = oldOsExit }()
	exitCalled := false
	osExit = func(code int) {
		exitCalled = true
		if code != 1 {
			t.Errorf("Fatal() called os.Exit(%d), want os.Exit(1)", code)
		}
	}

	log.Fatal("fatal message")

	if !exitCalled {
		t.Error("Fatal() didn't call os.Exit")
	}

	output := buf.String()
	if !strings.Contains(output, "fatal message") {
		t.Errorf("Fatal() output doesn't contain message: %q", output)
	}
	if output[0] != 'E' {
		t.Errorf("Fatal() didn't use 'E' level character: %q", output)
	}
}

// TestFatalf tests formatted Fatal logging (without actually exiting)
func TestFatalf(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, true)

	// Replace os.Exit with a no-op for testing
	oldOsExit := osExit
	defer func() { osExit = oldOsExit }()
	exitCalled := false
	osExit = func(code int) {
		exitCalled = true
		if code != 1 {
			t.Errorf("Fatalf() called os.Exit(%d), want os.Exit(1)", code)
		}
	}

	log.Fatalf("formatted %s %d", "fatal", 1)

	if !exitCalled {
		t.Error("Fatalf() didn't call os.Exit")
	}

	output := buf.String()
	if !strings.Contains(output, "formatted fatal 1") {
		t.Errorf("Fatalf() output doesn't contain formatted message: %q", output)
	}
	if output[0] != 'E' {
		t.Errorf("Fatalf() didn't use 'E' level character: %q", output)
	}
}

// TestConcurrentLogging tests thread safety
func TestConcurrentLogging(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, true)

	// Run concurrent logging operations
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			log.Infof("concurrent message %d", id)
			log.Warningf("concurrent warning %d", id)
			log.Errorf("concurrent error %d", id)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	output := buf.String()
	// Should have 30 lines (10 goroutines * 3 messages each)
	lines := strings.Count(output, "\n")
	if lines != 30 {
		t.Errorf("Concurrent logging produced %d lines, want 30", lines)
	}
}

// TestLogLevelFiltering tests that messages are filtered by level
func TestLogLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, false) // verbose disabled

	log.Info("should not appear")
	log.Warning("should appear")
	log.Error("should also appear")

	output := buf.String()

	if strings.Contains(output, "should not appear") {
		t.Error("Info message logged when verbose=false")
	}

	if !strings.Contains(output, "should appear") {
		t.Error("Warning message not logged")
	}

	if !strings.Contains(output, "should also appear") {
		t.Error("Error message not logged")
	}
}

// TestPIDInOutput tests that the process ID is included in the output
func TestPIDInOutput(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, true)

	log.Info("test")

	output := buf.String()

	// The PID should appear in the output
	// Format: I0314 15:24:36.123456 12345 file.go:123]
	pattern := regexp.MustCompile(`\d{6} (\d+) \w+\.go:\d+\]`)
	if !pattern.MatchString(output) {
		t.Errorf("Could not find PID in expected format in output: %q", output)
	}
}

// TestTimestampFormat tests that the timestamp is in the correct format
func TestTimestampFormat(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, true)

	log.Info("test")

	output := buf.String()

	// Format: I0314 15:24:36.123456 ...
	// Month/Day: 4 digits, Time: HH:MM:SS.microseconds
	timestampPattern := `^I\d{4} \d{2}:\d{2}:\d{2}\.\d{6}`
	matched, err := regexp.MatchString(timestampPattern, output)
	if err != nil {
		t.Fatalf("regex error: %v", err)
	}
	if !matched {
		t.Errorf("Timestamp format doesn't match expected pattern.\nGot: %q", output)
	}
}
