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

package logp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	golog "log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestLogger(t *testing.T) {
	exerciseLogger := func() {
		log := NewLogger("example")
		log.Info("some message")
		log.Infof("some message with parameter x=%v, y=%v", 1, 2)
		log.Infow("some message", "x", 1, "y", 2)
		log.Infow("some message", Int("x", 1))
		log.Infow("some message with namespaced args", Namespace("metrics"), "x", 1, "y", 1)
		log.Infow("", "empty_message", true)

		// Add context.
		log.With("x", 1, "y", 2).Warn("logger with context")

		someStruct := struct {
			X int `json:"x"`
			Y int `json:"y"`
		}{1, 2}
		log.Infow("some message with struct value", "metrics", someStruct)
	}

	err := TestingSetup()
	require.NoError(t, err)
	exerciseLogger()
	err = TestingSetup()
	require.NoError(t, err)
	exerciseLogger()
}

func TestLoggerLevel(t *testing.T) {
	if err := DevelopmentSetup(ToObserverOutput()); err != nil {
		t.Fatalf("cannot initialise logger on development mode: %+v", err)
	}

	const loggerName = "tester"
	logger := NewLogger(loggerName)

	logger.Debug("debug")
	logs := ObserverLogs().TakeAll()
	if assert.Len(t, logs, 1) {
		assert.Equal(t, zap.DebugLevel, logs[0].Level)
		assert.Equal(t, loggerName, logs[0].LoggerName)
		assert.Equal(t, "debug", logs[0].Message)
	}

	logger.Info("info")
	logs = ObserverLogs().TakeAll()
	if assert.Len(t, logs, 1) {
		assert.Equal(t, zap.InfoLevel, logs[0].Level)
		assert.Equal(t, loggerName, logs[0].LoggerName)
		assert.Equal(t, "info", logs[0].Message)
	}

	logger.Warn("warn")
	logs = ObserverLogs().TakeAll()
	if assert.Len(t, logs, 1) {
		assert.Equal(t, zap.WarnLevel, logs[0].Level)
		assert.Equal(t, loggerName, logs[0].LoggerName)
		assert.Equal(t, "warn", logs[0].Message)
	}

	logger.Error("error")
	logs = ObserverLogs().TakeAll()
	if assert.Len(t, logs, 1) {
		assert.Equal(t, zap.ErrorLevel, logs[0].Level)
		assert.Equal(t, loggerName, logs[0].LoggerName)
		assert.Equal(t, "error", logs[0].Message)
	}
}

func TestLoggerSetLevel(t *testing.T) {
	if err := DevelopmentSetup(ToObserverOutput()); err != nil {
		t.Fatal(err)
	}

	const loggerName = "tester"
	logger := NewLogger(loggerName)

	logger.Debug("debug")
	logs := ObserverLogs().TakeAll()
	if assert.Len(t, logs, 1) {
		assert.Equal(t, zap.DebugLevel, logs[0].Level)
		assert.Equal(t, loggerName, logs[0].LoggerName)
		assert.Equal(t, "debug", logs[0].Message)
	}

	SetLevel(zap.InfoLevel)
	logger.Info("info")
	logs = ObserverLogs().TakeAll()
	if assert.Len(t, logs, 1) {
		assert.Equal(t, zap.InfoLevel, logs[0].Level)
		assert.Equal(t, loggerName, logs[0].LoggerName)
		assert.Equal(t, "info", logs[0].Message)
	}

	logger.Debug("debug")
	logs = ObserverLogs().TakeAll()
	assert.Empty(t, logs, 1)
}

func TestL(t *testing.T) {
	if err := DevelopmentSetup(ToObserverOutput()); err != nil {
		t.Fatal(err)
	}

	L().Infow("infow", "rate", 2)
	logs := ObserverLogs().TakeAll()
	if assert.Len(t, logs, 1) {
		log := logs[0]
		assert.Equal(t, zap.InfoLevel, log.Level)
		assert.Equal(t, "", log.LoggerName)
		assert.Equal(t, "infow", log.Message)
		assert.Contains(t, log.ContextMap(), "rate")
	}

	const loggerName = "tester"
	L().Named(loggerName).Warnf("warning %d", 1)
	logs = ObserverLogs().TakeAll()
	if assert.Len(t, logs, 1) {
		log := logs[0]
		assert.Equal(t, zap.WarnLevel, log.Level)
		assert.Equal(t, loggerName, log.LoggerName)
		assert.Equal(t, "warning 1", log.Message)
	}
}

func TestDebugAllStdoutEnablesDefaultGoLogger(t *testing.T) {
	err := DevelopmentSetup(WithSelectors("*"))
	require.NoError(t, err)
	assert.Equal(t, _defaultGoLog, golog.Writer())

	err = DevelopmentSetup(WithSelectors("stdlog"))
	require.NoError(t, err)
	assert.Equal(t, _defaultGoLog, golog.Writer())

	err = DevelopmentSetup(WithSelectors("*", "stdlog"))
	require.NoError(t, err)
	assert.Equal(t, _defaultGoLog, golog.Writer())

	err = DevelopmentSetup(WithSelectors("other"))
	require.NoError(t, err)
	assert.Equal(t, io.Discard, golog.Writer())
}

func TestNotDebugAllStdoutDisablesDefaultGoLogger(t *testing.T) {
	err := DevelopmentSetup(WithSelectors("*"), WithLevel(InfoLevel))
	require.NoError(t, err)
	assert.Equal(t, io.Discard, golog.Writer())

	err = DevelopmentSetup(WithSelectors("stdlog"), WithLevel(InfoLevel))
	require.NoError(t, err)
	assert.Equal(t, io.Discard, golog.Writer())

	err = DevelopmentSetup(WithSelectors("*", "stdlog"), WithLevel(InfoLevel))
	require.NoError(t, err)
	assert.Equal(t, io.Discard, golog.Writer())

	err = DevelopmentSetup(WithSelectors("other"), WithLevel(InfoLevel))
	require.NoError(t, err)
	assert.Equal(t, io.Discard, golog.Writer())
}

func TestLoggingECSFields(t *testing.T) {
	cfg := Config{
		Beat:        "beat1",
		Level:       DebugLevel,
		development: true,
		Files: FileConfig{
			Name: "beat1",
		},
	}
	ToObserverOutput()(&cfg)
	err := Configure(cfg)
	require.NoError(t, err)

	logger := NewLogger("tester")

	logger.Debug("debug")
	logs := ObserverLogs().TakeAll()
	if assert.Len(t, logs, 1) {
		if assert.Len(t, logs[0].Context, 1) {
			assert.Equal(t, "service.name", logs[0].Context[0].Key)
			assert.Equal(t, "beat1", logs[0].Context[0].String)
		}
	}
}

func TestCreatingNewLoggerWithDifferentOutput(t *testing.T) {
	// We have no problems on Linux and Darwin, so we can rely on t.TempDir
	// that will remove the files once the tests finishes.
	tempDir1 := t.TempDir()
	tempDir2 := t.TempDir()

	secondLoggerMessage := "this is a log message"
	secondLoggerName := t.Name() + "-second"

	// We follow the same approach as on a Beat, first the logger
	// (always global) is configured and used, then we instantiate
	// a new one, secondLogger, and perform the tests on it.
	loggerCfg := DefaultConfig(DefaultEnvironment)
	loggerCfg.Beat = t.Name() + "-first"
	loggerCfg.ToFiles = true
	loggerCfg.ToStderr = false
	loggerCfg.Files.Name = "test-log-file-first"
	// We want a separate directory for this logger
	// and we don't need to inspect it.
	loggerCfg.Files.Path = tempDir1

	// Configures the global logger with the "default" log configuration.
	if err := Configure(loggerCfg); err != nil {
		t.Errorf("could not initialise logger: %s", err)
	}

	// Create a log entry just to "test" the logger
	firstLoggerName := "default-beat-logger"
	firstLoggerMessage := "not the message we want"

	logger := L().Named(firstLoggerName)
	logger.Info(firstLoggerMessage)
	if err := logger.Sync(); err != nil {
		t.Fatalf("could not sync log file from fist logger: %s", err)
	}
	t.Cleanup(func() {
		if err := logger.Close(); err != nil {
			t.Fatalf("could not close first logger: %s", err)
		}
	})

	// Actually clones the logger and use the "WithFileOutput" function
	secondCfg := DefaultConfig(DefaultEnvironment)
	secondCfg.ToFiles = true
	secondCfg.ToStderr = false
	secondCfg.Files.Name = "test-log-file"
	secondCfg.Files.Path = tempDir2

	// Create a new output for the second logger using the same level
	// as the global logger
	out, err := createLogOutput(secondCfg, loggerCfg.Level.ZapLevel())
	if err != nil {
		t.Fatalf("could not create output for second config")
	}
	outCore := func(zapcore.Core) zapcore.Core { return out }

	// We do not call Configure here as we do not want to affect
	// the global logger configuration
	secondLogger := NewLogger(secondLoggerName)
	secondLogger = secondLogger.WithOptions(zap.WrapCore(outCore))
	secondLogger.Info(secondLoggerMessage)
	if err := secondLogger.Sync(); err != nil {
		t.Fatalf("could not sync log file from second logger: %s", err)
	}
	t.Cleanup(func() {
		if err := secondLogger.Close(); err != nil {
			t.Fatalf("could not close second logger: %s", err)
		}
	})

	// Write again with the first logger to ensure it has not been affected
	// by the new configuration on the second logger.
	logger.Info(firstLoggerMessage)
	if err := logger.Sync(); err != nil {
		t.Fatalf("could not sync log file from fist logger: %s", err)
	}

	// Ensure the second logger is working as expected
	assertKVinLogentry(t, tempDir2, "log.logger", secondLoggerName)
	assertKVinLogentry(t, tempDir2, "message", secondLoggerMessage)

	// Ensure the first logger is working as expected
	assertKVinLogentry(t, tempDir1, "log.logger", firstLoggerName)
	assertKVinLogentry(t, tempDir1, "message", firstLoggerMessage)
}

func assertKVinLogentry(t *testing.T, dir, key, value string) {
	t.Helper()

	// Find the log file. The file name gets the date added, so we list the
	// directory and ensure there is only one file there.
	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("could not read temporary directory '%s': %s", dir, err)
	}

	// If there is more than one file, list all files
	// and fail the test.
	if len(files) != 1 {
		t.Errorf("found %d files in '%s', there must be only one", len(files), dir)
		t.Errorf("Files in '%s':", dir)
		for _, f := range files {
			t.Error(f.Name())
		}
		t.FailNow()
	}

	fullPath := filepath.Join(dir, files[0].Name())
	f, err := os.Open(fullPath)
	if err != nil {
		t.Fatalf("could not open '%s' for reading: %s", fullPath, err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	lines := []string{}
	for sc.Scan() {
		logData := sc.Bytes()

		logEntry := map[string]any{}
		if err := json.Unmarshal(logData, &logEntry); err != nil {
			t.Fatalf("could not read log entry as JSON. Log entry: '%s'", string(logData))
		}

		if logEntry[key] == value {
			return
		}
		lines = append(lines, string(logData))
	}

	t.Errorf("could not find [%s]='%s' in any log line.", key, value)
	t.Log("Log lines:")
	for _, l := range lines {
		t.Log(l)
	}
}

type writeSyncer struct {
	strings.Builder
}

// Sync is a no-op
func (w writeSyncer) Sync() error {
	return nil
}

func TestTypedLoggerCore(t *testing.T) {
	testCases := []struct {
		name               string
		entry              zapcore.Entry
		field              zapcore.Field
		expectedDefaultLog string
		expectedTypedLog   string
	}{
		{
			name:               "info level default logger",
			entry:              zapcore.Entry{Level: zapcore.InfoLevel, Message: "msg"},
			field:              skipField(),
			expectedDefaultLog: `{"level":"info","msg":"msg"}`,
		},
		{
			name:             "info level typed logger",
			entry:            zapcore.Entry{Level: zapcore.InfoLevel, Message: "msg"},
			field:            strField("log.type", "sensitive"),
			expectedTypedLog: `{"level":"info","msg":"msg","log.type":"sensitive"}`,
		},

		{
			name:  "debug level typed logger",
			entry: zapcore.Entry{Level: zapcore.DebugLevel, Message: "msg"},
			field: skipField(),
		},
		{
			name:  "debug level typed logger",
			entry: zapcore.Entry{Level: zapcore.DebugLevel, Message: "msg"},
			field: strField("log.type", "sensitive"),
		},
	}

	defaultWriter := writeSyncer{}
	typedWriter := writeSyncer{}

	cfg := zap.NewProductionEncoderConfig()
	cfg.TimeKey = "" // remove the time to make the log entry consistent

	defaultCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(cfg),
		&defaultWriter,
		zapcore.InfoLevel,
	)

	typedCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(cfg),
		&typedWriter,
		zapcore.InfoLevel,
	)

	core := typedLoggerCore{
		defaultCore: defaultCore,
		typedCore:   typedCore,
		key:         "log.type",
		value:       "sensitive",
	}

	for _, tc := range testCases {
		t.Run(tc.name+" Check method", func(t *testing.T) {
			defaultWriter.Reset()
			typedWriter.Reset()

			if ce := core.Check(tc.entry, nil); ce != nil {
				ce.Write(tc.field)
			}
			defaultLog := strings.TrimSpace(defaultWriter.String())
			typedLog := strings.TrimSpace(typedWriter.String())

			if tc.expectedDefaultLog != defaultLog {
				t.Errorf("expecting default log to be %q, got %q", tc.expectedDefaultLog, defaultLog)
			}
			if tc.expectedTypedLog != typedLog {
				t.Errorf("expecting typed log to be %q, got %q", tc.expectedTypedLog, typedLog)
			}
		})

		// The write method does not check the level, so we skip
		// this test if the test case is a lower level
		if tc.entry.Level < zapcore.InfoLevel {
			continue
		}

		t.Run(tc.name+" Write method", func(t *testing.T) {
			defaultWriter.Reset()
			typedWriter.Reset()

			//nolint:errcheck // It's a test and the underlying writer never fails.
			core.Write(tc.entry, []zapcore.Field{tc.field})

			defaultLog := strings.TrimSpace(defaultWriter.String())
			typedLog := strings.TrimSpace(typedWriter.String())

			if tc.expectedDefaultLog != defaultLog {
				t.Errorf("expecting default log to be %q, got %q", tc.expectedDefaultLog, defaultLog)
			}
			if tc.expectedTypedLog != typedLog {
				t.Errorf("expecting typed log to be %q, got %q", tc.expectedTypedLog, typedLog)
			}

		})
	}

	t.Run("method Enabled", func(t *testing.T) {
		if !core.Enabled(zapcore.InfoLevel) {
			t.Error("core.Enable must return true for level info")
		}

		if core.Enabled(zapcore.DebugLevel) {
			t.Error("core.Enable must return true for level debug")
		}
	})
}

func TestTypedLoggerCoreSync(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		core := typedLoggerCore{
			defaultCore: &ZapCoreMock{
				SyncFunc: func() error { return nil },
			},
			typedCore: &ZapCoreMock{
				SyncFunc: func() error { return nil },
			},
		}

		if err := core.Sync(); err != nil {
			t.Fatalf("Sync must not return an error: %s", err)
		}
	})

	t.Run("both cores return error", func(t *testing.T) {
		errMsg1 := "some error from defaultCore"
		errMsg2 := "some error from typedCore"
		core := typedLoggerCore{
			defaultCore: &ZapCoreMock{
				SyncFunc: func() error { return errors.New(errMsg1) },
			},
			typedCore: &ZapCoreMock{
				SyncFunc: func() error { return errors.New(errMsg2) },
			},
		}

		err := core.Sync()
		if err == nil {
			t.Fatal("Sync must return an error")
		}

		gotMsg := err.Error()
		if !strings.Contains(gotMsg, errMsg1) {
			t.Errorf("expecting %q in the error string: %q", errMsg1, gotMsg)
		}
		if !strings.Contains(gotMsg, errMsg2) {
			t.Errorf("expecting %q in the error string: %q", errMsg2, gotMsg)
		}
	})
}

func TestTypedLoggerCoreWith(t *testing.T) {
	defaultWriter := writeSyncer{}
	typedWriter := writeSyncer{}

	cfg := zap.NewProductionEncoderConfig()
	cfg.TimeKey = "" // remove the time to make the log entry consistent

	defaultCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(cfg),
		&defaultWriter,
		zapcore.InfoLevel,
	)

	typedCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(cfg),
		&typedWriter,
		zapcore.InfoLevel,
	)

	core := typedLoggerCore{
		defaultCore: defaultCore,
		typedCore:   typedCore,
		key:         "log.type",
		value:       "sensitive",
	}

	expectedLines := []string{
		// First/Default logger
		`{"level":"info","msg":"Very first message"}`,

		// Two messages after calling With
		`{"level":"info","msg":"a message with extra fields","foo":"bar"}`,
		`{"level":"info","msg":"another message with extra fields","foo":"bar"}`,

		// A message with the default logger
		`{"level":"info","msg":"a message without extra fields"}`,

		// Two more messages with a different field
		`{"level":"info","msg":"a message with an answer","answer":"42"}`,
		`{"level":"info","msg":"another message with an answer","answer":"42"}`,

		// One last message with the default logger
		`{"level":"info","msg":"another message without any extra fields"}`,
	}

	// The default logger, it should not be modified by any call to With.
	logger := zap.New(&core)
	logger.Info("Very first message")

	// Add a field and write messages
	loggerWithFields := logger.With(strField("foo", "bar"))
	loggerWithFields.Info("a message with extra fields")
	loggerWithFields.Info("another message with extra fields")

	// Use the default logger again
	logger.Info("a message without extra fields")

	// New logger with other fields
	loggerWithFields = logger.With(strField("answer", "42"))
	loggerWithFields.Info("a message with an answer")
	loggerWithFields.Info("another message with an answer")

	// One last message with the default logger
	logger.Info("another message without any extra fields")

	scanner := bufio.NewScanner(strings.NewReader(defaultWriter.String()))
	count := 0
	for scanner.Scan() {
		l := scanner.Text()
		if l != expectedLines[count] {
			t.Error("Expecting:\n", l, "\nGot:\n", expectedLines[count])
		}
		count++
	}
}

func TestCreateLogOutputAllDisabled(t *testing.T) {
	cfg := DefaultConfig(DefaultEnvironment)
	cfg.toIODiscard = false
	cfg.toObserver = false
	cfg.ToEventLog = false
	cfg.ToFiles = false
	cfg.ToStderr = false
	cfg.ToSyslog = false

	out, err := createLogOutput(cfg, zap.DebugLevel)
	if err != nil {
		t.Fatalf("did not expect an error calling createLogOutput: %s", err)
	}

	if out.Enabled(zap.DebugLevel) {
		t.Fatal("the output must be disabled to all log levels")
	}
}

func TestCoresCanBeClosed(t *testing.T) {
	cfg := DefaultConfig(DefaultEnvironment)
	cfg.ToFiles = true

	fileOutput, err := makeFileOutput(cfg, zapcore.DebugLevel)
	if err != nil {
		t.Fatalf("cannot create file output: %s", err)
	}

	closer, ok := fileOutput.(io.Closer)
	if !ok {
		t.Fatal("the 'File Output' does not implement io.Closer")
	}
	if err := closer.Close(); err != nil {
		t.Fatalf("Close must not return any error, got: %s", err)
	}
}

func TestCloserLoggerCoreWith(t *testing.T) {
	defaultWriter := writeSyncer{}

	cfg := zap.NewProductionEncoderConfig()
	cfg.TimeKey = "" // remove the time to make the log entry consistent

	core := closerCore{
		Core: zapcore.NewCore(
			zapcore.NewJSONEncoder(cfg),
			&defaultWriter,
			zapcore.InfoLevel,
		),
	}

	expectedLines := []string{
		// First/Default logger
		`{"level":"info","msg":"Very first message"}`,

		// Two messages after calling With
		`{"level":"info","msg":"a message with extra fields","foo":"bar"}`,
		`{"level":"info","msg":"another message with extra fields","foo":"bar"}`,

		// A message with the default logger
		`{"level":"info","msg":"a message without extra fields"}`,

		// Two more messages with a different field
		`{"level":"info","msg":"a message with an answer","answer":"42"}`,
		`{"level":"info","msg":"another message with an answer","answer":"42"}`,

		// One last message with the default logger
		`{"level":"info","msg":"another message without any extra fields"}`,
	}

	// The default logger, it should not be modified by any call to With.
	logger := zap.New(&core)
	logger.Info("Very first message")

	// Add a field and write messages
	loggerWithFields := logger.With(strField("foo", "bar"))
	loggerWithFields.Info("a message with extra fields")
	loggerWithFields.Info("another message with extra fields")

	// Use the default logger again
	logger.Info("a message without extra fields")

	// New logger with other fields
	loggerWithFields = logger.With(strField("answer", "42"))
	loggerWithFields.Info("a message with an answer")
	loggerWithFields.Info("another message with an answer")

	// One last message with the default logger
	logger.Info("another message without any extra fields")

	scanner := bufio.NewScanner(strings.NewReader(defaultWriter.String()))
	count := 0
	for scanner.Scan() {
		l := scanner.Text()
		if l != expectedLines[count] {
			t.Error("Expecting:\n", l, "\nGot:\n", expectedLines[count])
		}
		count++
	}
}

func TestConfigureWithCore(t *testing.T) {
	testMsg := "The quick brown fox jumped over the lazy dog."
	var b bytes.Buffer
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&b),
		zapcore.InfoLevel)
	err := ConfigureWithCore(Config{}, core)
	if err != nil {
		t.Fatalf("Unexpected err: %s", err)
	}
	Info("The quick brown %s jumped over the lazy %s.", "fox", "dog")
	var r map[string]interface{}

	err = json.Unmarshal(b.Bytes(), &r)
	if err != nil {
		t.Fatalf("unable to json unmarshal '%s': %s", b.String(), err)
	}

	val, prs := r["msg"]
	if !prs {
		t.Fatalf("expected 'msg' field not present in '%s'", b.String())
	}
	if val != testMsg {
		t.Fatalf("expected msg of '%s', got '%s'", testMsg, val)
	}

	val, prs = r["level"]
	if !prs {
		t.Fatalf("expected 'level' field not present in '%s'", b.String())
	}
	if val != "info" {
		t.Fatalf("expected log.level of 'info', got '%s'", val)
	}
}

func strField(key, val string) zapcore.Field {
	return zapcore.Field{Type: zapcore.StringType, Key: key, String: val}
}

func skipField() zapcore.Field {
	return zapcore.Field{Type: zapcore.SkipType}
}
