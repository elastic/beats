// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httplog

import (
	"testing"
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
}

func TestIsPathIn(t *testing.T) {
	for _, test := range pathTests {
		t.Run(test.name, func(t *testing.T) {
			got, err := IsPathIn(test.root, test.path)
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
			got, err := resolveSymlinks(test.path)
			if err != nil {
				t.Fatalf("unexpected error calling resolveSymlinks: %v", err)
			}
			if got != test.want {
				t.Errorf("unexpected result: got %q, want %q", got, test.want)
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
