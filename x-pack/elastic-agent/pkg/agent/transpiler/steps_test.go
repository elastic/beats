// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package transpiler

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSubpath(t *testing.T) {
	testCases := map[string][]struct {
		root       string
		path       string
		resultPath string
		isSubpath  bool
	}{
		"linux": {
			{"/", "a", "/a", true},
			{"/a", "b", "/a/b", true},
			{"/a", "b/c", "/a/b/c", true},

			{"/a/b", "/a/c", "/a/c", false},

			{"/a/b", "/a/b/../c", "/a/c", false},
			{"/a/b", "../c", "/a/c", false},
			{"/a", "/a/b/c", "/a/b/c", true},
			{"/a", "/A/b/c", "/A/b/c", false},
		},
		"darwin": {
			{"/", "a", "/a", true},
			{"/a", "b", "/a/b", true},
			{"/a", "b/c", "/a/b/c", true},
			{"/a/b", "/a/c", "/a/c", false},
			{"/a/b", "/a/b/../c", "/a/c", false},
			{"/a/b", "../c", "/a/c", false},
			{"/a", "/a/b/c", "/a/b/c", true},
			{"/a", "/A/b/c", "/a/b/c", true},
		},
		"windows": {
			{"c:/", "/a", "c:\\a", true},
			{"c:/a", "b", "c:\\a\\b", true},
			{"c:/a", "b/c", "c:\\a\\b\\c", true},
			{"c:/a/b", "/a/c", "c:\\a\\c", false},
			{"c:/a/b", "/a/b/../c", "c:\\a\\c", false},
			{"c:/a/b", "../c", "c:\\a\\c", false},
			{"c:/a", "/a/b/c", "c:\\a\\b\\c", true},
			{"c:/a", "/A/b/c", "c:\\a\\b\\c", true},
			{"c:/a", "c:/A/b/c", "c:\\a\\b\\c", true},
			{"c:/a", "c:/b/c", "c:\\b\\c", false},
		},
	}

	osSpecificTests, found := testCases[runtime.GOOS]
	if !found {
		return
	}

	for _, test := range osSpecificTests {
		t.Run(fmt.Sprintf("[%s] root:'%s path: %s'", runtime.GOOS, test.root, test.path), func(t *testing.T) {
			newPath, result, err := joinPaths(test.root, test.path)
			assert.NoError(t, err)
			assert.Equal(t, test.resultPath, newPath)
			assert.Equal(t, test.isSubpath, result)
		})
	}
}

func TestExecFile_Success(t *testing.T) {
	t.Skip("skipping failing tests")
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	binaryPath := "tests/exec-1.0-darwin-x86_64/exec"
	step := ExecFile(10, binaryPath, "-output=stdout", "-exitcode=0")
	err = step.Execute(context.Background(), pwd)
	if err != nil {
		t.Fatal("command should not have errored")
	}
}

func TestExecFile_StdErr(t *testing.T) {
	t.Skip("skipping failing tests")

	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	binaryPath := "tests/exec-1.0-darwin-x86_64/exec"
	step := ExecFile(10, binaryPath, "-output=stderr", "-exitcode=15")
	err = step.Execute(context.Background(), pwd)
	if err == nil {
		t.Fatal("command should have errored")
	}
	errMsg := "operation 'Exec' failed (return code: 15): message written to stderr"
	if err.Error() != errMsg {
		t.Fatalf("got unexpected error: %s", err)
	}
}

func TestExecFile_StdOut(t *testing.T) {
	t.Skip("skipping failing tests")

	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	binaryPath := "tests/exec-1.0-darwin-x86_64/exec"
	step := ExecFile(10, binaryPath, "-output=stdout", "-exitcode=16")
	err = step.Execute(context.Background(), pwd)
	if err == nil {
		t.Fatal("command should have errored")
	}
	errMsg := "operation 'Exec' failed (return code: 16): message written to stdout"
	if err.Error() != errMsg {
		t.Fatalf("got unexpected error: %s", err)
	}
}

func TestExecFile_NoOutput(t *testing.T) {
	t.Skip("skipping failing tests")

	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	binaryPath := "tests/exec-1.0-darwin-x86_64/exec"
	step := ExecFile(10, binaryPath, "-no-output", "-exitcode=17")
	err = step.Execute(context.Background(), pwd)
	if err == nil {
		t.Fatal("command should have errored")
	}
	errMsg := "operation 'Exec' failed (return code: 17): (command had no output)"
	if err.Error() != errMsg {
		t.Fatalf("got unexpected error: %s", err)
	}
}
