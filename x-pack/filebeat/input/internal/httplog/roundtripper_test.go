// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httplog

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/paths"
)

var pathTests = []struct {
	name    string
	root    string
	path    string
	want    bool
	wantErr error
}{
	// Happy cases.
	{
		name:    "root_test_root",
		root:    "path/to/root",
		path:    "path/to/root",
		want:    true,
		wantErr: nil,
	},
	{
		name:    "abs_root_test_root",
		root:    "/abs/path/to/root",
		path:    "/abs/path/to/root",
		want:    true,
		wantErr: nil,
	},
	{
		name:    "root_test_subdir",
		root:    "path/to/root",
		path:    "path/to/root/subdir",
		want:    true,
		wantErr: nil,
	},
	{
		name:    "abs_root_test_subdir",
		root:    "/abs/path/to/root",
		path:    "/abs/path/to/root/subdir",
		want:    true,
		wantErr: nil,
	},
	{
		name:    "root_test_missing_subdir",
		root:    "path/to/root",
		path:    "path/to/root/no_subdir",
		want:    true,
		wantErr: nil,
	},
	{
		name:    "abs_root_test_missing_subdir",
		root:    "/abs/path/to/root",
		path:    "/abs/path/to/root/no_subdir",
		want:    true,
		wantErr: nil,
	},
	{
		name:    "root_test_missing_file",
		root:    "path/to/root",
		path:    "path/to/root/subdir/no_file",
		want:    true,
		wantErr: nil,
	},
	{
		name:    "abs_root_test_missing_file",
		root:    "/abs/path/to/root",
		path:    "/abs/path/to/root/subdir/no_file",
		want:    true,
		wantErr: nil,
	},
	{
		name:    "root_test_file",
		root:    "path/to/root",
		path:    "path/to/root/subdir/file",
		want:    true,
		wantErr: nil,
	},
	{
		name:    "abs_root_test_file",
		root:    "/abs/path/to/root",
		path:    "/abs/path/to/root/subdir/file",
		want:    true,
		wantErr: nil,
	},
	{
		name:    "symlink_traversal_with_file_back_in_to_root",
		root:    "testdata/root",
		path:    "testdata/root/outside_root/../../root/target-*.log",
		want:    true,
		wantErr: nil,
	},

	// Malory's tests.
	{
		name:    "root_test_escape_subdir",
		root:    "path/to/root",
		path:    "path/to/root/../../escape_dir",
		want:    false,
		wantErr: nil,
	},
	{
		name:    "abs_root_test_escape_subdir",
		root:    "/abs/path/to/root",
		path:    "/abs/path/to/root/../../escape_dir",
		want:    false,
		wantErr: nil,
	},
	{
		name:    "root_test_pwd",
		root:    "path/to/root",
		path:    ".",
		want:    false,
		wantErr: nil,
	},
	{
		name:    "abs_root_test_pwd",
		root:    "/abs/path/to/root",
		path:    ".",
		want:    false,
		wantErr: nil,
	},
	{
		name:    "root_test_pwd_parent",
		root:    "path/to/root",
		path:    "..",
		want:    false,
		wantErr: nil,
	},
	{
		name:    "abs_root_test_pwd_parent",
		root:    "/abs/path/to/root",
		path:    "..",
		want:    false,
		wantErr: nil,
	},
	{
		name:    "root_test_fs_root",
		root:    "path/to/root",
		path:    "/",
		want:    false,
		wantErr: nil,
	},
	{
		name:    "abs_root_test_fs_root",
		root:    "/abs/path/to/root",
		path:    "/",
		want:    false,
		wantErr: nil,
	},
	{
		name:    "root_test_var",
		root:    "path/to/root",
		path:    "/var/log",
		want:    false,
		wantErr: nil,
	},
	{
		name:    "symlink_traversal_no_file",
		root:    "testdata/root",
		path:    "testdata/root/outside_root",
		want:    false,
		wantErr: nil,
	},
	{
		name:    "symlink_traversal_with_file",
		root:    "testdata/root",
		path:    "testdata/root/outside_root/target-*.log",
		want:    false,
		wantErr: nil,
	},
	{
		name:    "symlink_traversal_prefix_deep_file",
		root:    "testdata/root",
		path:    "testdata/root/path/outside_root/target-*.log",
		want:    false,
		wantErr: nil,
	},
	{
		name:    "symlink_traversal_postfix_deep_file",
		root:    "testdata/root",
		path:    "testdata/root/outside_root/path/target-*.log",
		want:    false,
		wantErr: nil,
	},
	{
		name:    "abs_root_is_parent_of_root",
		root:    "/abs/path/to/root",
		path:    "/abs/path/to",
		want:    false,
		wantErr: nil,
	},
}

func TestIsPathIn(t *testing.T) {
	for _, test := range pathTests {
		t.Run(test.name, func(t *testing.T) {
			got, err := IsPathIn(filepath.FromSlash(test.root), filepath.FromSlash(test.path))
			if !sameError(err, test.wantErr) {
				t.Errorf("unexpected error from IsPathIn: got:%q want:%q", err, test.wantErr)
			}
			if got != test.want {
				t.Errorf("unexpected result from IsPathIn: got:%t want:%t", got, test.want)
			}
		})
	}
}

var symlinkTests = []struct {
	path, want string
}{
	{path: "path/to/root", want: "path/to/root"},
	{path: "/abs/path/to/root", want: "/abs/path/to/root"},
	{path: "testdata/root/outside_root", want: "testdata/outside_root"},
	{path: "testdata/root/outside_root/target-*.log", want: "testdata/outside_root/target-*.log"},
	{path: "testdata/root/path/outside_root/target-*.log", want: "testdata/outside_root/target-*.log"},
	{path: "testdata/root/outside_root/path/target-*.log", want: "testdata/outside_root/path/target-*.log"},
	{path: "testdata/root/outside_root/../../root/target-*.log", want: "testdata/root/target-*.log"},
}

func TestResolveSymlinks(t *testing.T) {
	for _, test := range symlinkTests {
		t.Run(test.path, func(t *testing.T) {
			got, err := resolveSymlinks(filepath.FromSlash(test.path))
			if err != nil {
				t.Fatalf("unexpected error calling resolveSymlinks: %v", err)
			}
			if got != filepath.FromSlash(test.want) {
				t.Errorf("unexpected result: got %q, want %q", got, filepath.FromSlash(test.want))
			}
		})
	}

}

func TestResolvePathInLogsFor(t *testing.T) {
	logsDir := filepath.Join(t.TempDir(), "logs")
	p := &paths.Path{Logs: logsDir}

	const input = "cel"
	root := filepath.Join(logsDir, input)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name         string
		path         string
		wantResolved string
		wantOK       bool

		goos string
	}{
		{
			name:         "bare_filename",
			path:         "trace.ndjson",
			wantResolved: filepath.Join(root, "trace.ndjson"),
			wantOK:       true,
		},
		{
			name:         "relative_subdir",
			path:         "subdir/trace.ndjson",
			wantResolved: filepath.Join(root, "subdir", "trace.ndjson"),
			wantOK:       true,
		},
		{
			name:         "relative_dotdot_stays_within",
			path:         "subdir/../trace.ndjson",
			wantResolved: filepath.Join(root, "trace.ndjson"),
			wantOK:       true,
		},
		{
			name:         "relative_dotdot_escapes",
			path:         "../../etc/passwd",
			wantResolved: filepath.Clean(filepath.Join(root, "../../etc/passwd")),
			wantOK:       false,
		},
		{
			name:         "absolute_within",
			path:         filepath.Join(root, "trace.ndjson"),
			wantResolved: filepath.Join(root, "trace.ndjson"),
			wantOK:       true,
		},
		{
			name:         "absolute_outside",
			path:         "/var/log/other.log",
			wantResolved: "/var/log/other.log",
			wantOK:       false,
		},
		{
			// This is the pattern used by Fleet integrations: the
			// relative path climbs out and back through ../../logs/<input>/
			// which collapses to the root when joined.
			name:         "integration_relative_pattern",
			path:         "../../logs/cel/http-request-trace-*.ndjson",
			wantResolved: filepath.Join(root, "http-request-trace-*.ndjson"),
			wantOK:       true,
		},
		{
			name:         "dot_resolves_to_root",
			path:         ".",
			wantResolved: root,
			wantOK:       true,
		},

		// Windows-specific path forms that exercise isRooted and
		// filepath.IsAbs independently. On Unix these forms have
		// different semantics (backslash is a literal character, drive
		// letters don't exist) so they are only meaningful on Windows.
		//
		// UNC (\\server\share\foo) and device (\\.\C:\foo) paths are
		// not tested here because resolving a non-existent UNC or
		// device path produces network/device errors that
		// resolveSymlinks does not handle, causing the test to fail
		// with an unexpected error rather than testing path
		// classification.
		{
			// Backslash-rooted: the counterpart of the forward-slash
			// absolute_outside case. filepath.IsAbs returns false
			// (no drive letter), but isRooted must catch the leading \.
			name:         "backslash_rooted_outside",
			path:         `\var\log\other.log`,
			wantResolved: `\var\log\other.log`,
			wantOK:       false,
			goos:         "windows",
		},
		{
			// Fully qualified DOS path outside root.
			// filepath.IsAbs returns true so isRooted is never reached.
			name:         "fully_qualified_dos_outside",
			path:         filepath.VolumeName(root) + `\other\path\file.log`,
			wantResolved: filepath.VolumeName(root) + `\other\path\file.log`,
			wantOK:       false,
			goos:         "windows",
		},
	}

	for _, tt := range tests {
		if tt.goos != "" && runtime.GOOS != tt.goos {
			continue
		}
		t.Run(tt.name, func(t *testing.T) {
			resolved, ok, err := ResolvePathInLogsFor(p, input, tt.path)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ok != tt.wantOK {
				t.Errorf("unexpected ok: got:%t want:%t", ok, tt.wantOK)
			}
			if resolved != tt.wantResolved {
				t.Errorf("unexpected resolved path: got:%q want:%q", resolved, tt.wantResolved)
			}
		})
	}
}

func TestResolveTraceFilename(t *testing.T) {
	logsDir := filepath.Join(t.TempDir(), "logs")
	p := &paths.Path{Logs: logsDir}

	const input = "cel"
	root := filepath.Join(logsDir, input)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		id       string
		filename string
		wantOK   bool
	}{
		{
			name:     "empty_filename",
			id:       "test",
			filename: "",
			wantOK:   true,
		},
		{
			name:     "relative_with_star",
			id:       "my:input:id",
			filename: "http-request-trace-*.ndjson",
			wantOK:   true,
		},
		{
			name:     "integration_pattern",
			id:       "cel-test-1",
			filename: "../../logs/cel/http-request-trace-*.ndjson",
			wantOK:   true,
		},
		{
			name:     "absolute_outside",
			id:       "test",
			filename: "/var/log/evil.ndjson",
			wantOK:   false,
		},
		{
			name:     "relative_escape",
			id:       "test",
			filename: "../../../etc/passwd",
			wantOK:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved, err := ResolveTraceFilename(p, input, tt.id, tt.filename)
			if tt.wantOK {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tt.filename == "" {
					if resolved != "" {
						t.Errorf("empty filename should resolve to empty, got %q", resolved)
					}
					return
				}
				ok, err := IsPathIn(root, resolved)
				if err != nil {
					t.Fatalf("IsPathIn error: %v", err)
				}
				if !ok {
					t.Errorf("resolved path %q is not within root %q", resolved, root)
				}
				if filepath.Ext(resolved) == "" {
					t.Errorf("resolved path %q lost its extension", resolved)
				}
			} else {
				if err == nil {
					t.Fatalf("expected error for path %q escaping logs dir, got resolved=%q", tt.filename, resolved)
				}
			}
		})
	}
}

func TestResolveTraceFilenameSanitisesID(t *testing.T) {
	logsDir := filepath.Join(t.TempDir(), "logs")
	p := &paths.Path{Logs: logsDir}

	const input = "cel"
	root := filepath.Join(logsDir, input)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}

	resolved, err := ResolveTraceFilename(p, input, "has:colon:chars", "trace-*.ndjson")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(root, "trace-has_colon_chars.ndjson")
	if resolved != want {
		t.Errorf("unexpected resolved path:\n got: %q\nwant: %q", resolved, want)
	}
}

func TestCleanTraceFiles(t *testing.T) {
	dir := t.TempDir()

	primary := filepath.Join(dir, "trace.ndjson")
	rotated := filepath.Join(dir, "trace-2026-01-02T15-04-05.000.ndjson")
	unrelated := filepath.Join(dir, "other.log")

	for _, f := range []string{primary, rotated, unrelated} {
		if err := os.WriteFile(f, []byte("data"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	log := logptest.NewTestingLogger(t, "test")
	CleanTraceFiles(primary, log)

	if _, err := os.Stat(primary); err == nil {
		t.Error("primary trace file should have been removed")
	}
	if _, err := os.Stat(rotated); err == nil {
		t.Error("rotated trace file should have been removed")
	}
	if _, err := os.Stat(unrelated); err != nil {
		t.Error("unrelated file should not have been removed")
	}
}

func TestCleanTraceFilesNonexistent(t *testing.T) {
	dir := t.TempDir()
	primary := filepath.Join(dir, "does-not-exist.ndjson")

	log := logptest.NewTestingLogger(t, "test")
	CleanTraceFiles(primary, log)
}

func TestLogRequestRedactsSensitiveHeaders(t *testing.T) {
	tests := []struct {
		name        string
		headers     http.Header
		sensitive   []string
		wantAbsent  []string
		wantPresent []string
	}{
		{
			name: "authorization_redacted",
			headers: http.Header{
				"Authorization": []string{"Bearer secret-token"},
				"Content-Type":  []string{"application/json"},
			},
			sensitive:   []string{"Authorization"},
			wantAbsent:  []string{"Authorization", "secret-token"},
			wantPresent: []string{"Content-Type", "application/json"},
		},
		{
			name: "multiple_sensitive_headers",
			headers: http.Header{
				"Authorization": []string{"Basic dXNlcjpwYXNz"},
				"X-Api-Key":     []string{"key-12345"},
				"Accept":        []string{"*/*"},
			},
			sensitive:   []string{"Authorization", "X-Api-Key"},
			wantAbsent:  []string{"dXNlcjpwYXNz", "key-12345"},
			wantPresent: []string{"Accept"},
		},
		{
			name: "no_sensitive_headers",
			headers: http.Header{
				"Authorization": []string{"Bearer visible"},
				"Content-Type":  []string{"text/plain"},
			},
			sensitive:   nil,
			wantAbsent:  nil,
			wantPresent: []string{"Authorization", "visible"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			core := zapcore.NewCore(
				zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
				zapcore.AddSync(&buf),
				zap.DebugLevel,
			)
			logger := zap.New(core)

			req, err := http.NewRequest(http.MethodGet, "http://example.com/test", nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header = tt.headers

			LogRequest(logger, req, tt.sensitive, 1024)
			logger.Sync()

			output := buf.String()
			for _, s := range tt.wantAbsent {
				if strings.Contains(output, s) {
					t.Errorf("log output must not contain %q", s)
				}
			}
			for _, s := range tt.wantPresent {
				if !strings.Contains(output, s) {
					t.Errorf("log output must contain %q", s)
				}
			}
		})
	}
}

func sameError(a, b error) bool {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil, b == nil:
		return false
	default:
		return a.Error() == b.Error()
	}
}
