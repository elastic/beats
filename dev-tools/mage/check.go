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

package mage

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/dev-tools/mage/gotool"
	"github.com/elastic/beats/v8/libbeat/dashboards"
	"github.com/elastic/beats/v8/libbeat/processors/dissect"
)

// Check looks for created/modified/deleted/renamed files and returns an error
// if it finds any modifications. If executed in in verbose mode it will write
// the results of 'git diff' to stdout to indicate what changes have been made.
//
// It checks the file permissions of python test cases and YAML files.
// It checks .go source files using 'go vet'.
func Check() error {
	fmt.Println(">> check: Checking source code for common problems")

	mg.Deps(GoVet, CheckPythonTestNotExecutable, CheckYAMLNotExecutable, CheckDashboardsFormat)

	changes, err := GitDiffIndex()
	if err != nil {
		return errors.Wrap(err, "failed to diff the git index")
	}

	if len(changes) > 0 {
		if mg.Verbose() {
			GitDiff()
		}

		return errors.Errorf("some files are not up-to-date. "+
			"Run 'make update' then review and commit the changes. "+
			"Modified: %v", changes)
	}
	return nil
}

// GitDiffIndex returns a list of files that differ from what is committed.
// These could file that were created, deleted, modified, or moved.
func GitDiffIndex() ([]string, error) {
	// Ensure the index is updated so that diff-index gives accurate results.
	if err := sh.Run("git", "update-index", "-q", "--refresh"); err != nil {
		return nil, err
	}

	// git diff-index provides a list of modified files.
	// https://www.git-scm.com/docs/git-diff-index
	out, err := sh.Output("git", "diff-index", "HEAD", "--", ".")
	if err != nil {
		return nil, err
	}

	// Example formats.
	// :100644 100644 bcd1234... 0123456... M file0
	// :100644 100644 abcd123... 1234567... R86 file1 file3
	d, err := dissect.New(":%{src_mode} %{dst_mode} %{src_sha1} %{dst_sha1} %{status}\t%{paths}")
	if err != nil {
		return nil, err
	}

	// Parse lines.
	var modified []string
	s := bufio.NewScanner(bytes.NewBufferString(out))
	for s.Scan() {
		m, err := d.Dissect(s.Text())
		if err != nil {
			return nil, errors.Wrap(err, "failed to dissect git diff-index output")
		}

		paths := strings.Split(m["paths"], "\t")
		if len(paths) > 1 {
			modified = append(modified, paths[1])
		} else {
			modified = append(modified, paths[0])
		}
	}
	if err = s.Err(); err != nil {
		return nil, err
	}

	return modified, nil
}

// GitDiff runs 'git diff' and writes the output to stdout.
func GitDiff() error {
	c := exec.Command("git", "--no-pager", "diff", "--minimal")
	c.Stdin = nil
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	log.Println("exec:", strings.Join(c.Args, " "))
	err := c.Run()
	return err
}

// CheckPythonTestNotExecutable checks that none of the python test files are
// executable. They are silently skipped and we don't want this to happen.
func CheckPythonTestNotExecutable() error {
	if runtime.GOOS == "windows" {
		// Skip windows because it doesn't have POSIX permissions.
		return nil
	}

	tests, err := FindFiles(pythonTestFiles...)
	if err != nil {
		return err
	}

	var executableTestFiles []string
	for _, file := range tests {
		info, err := os.Stat(file)
		if err != nil {
			return err
		}

		if info.Mode().Perm()&0111 > 0 {
			executableTestFiles = append(executableTestFiles, file)
		}
	}

	if len(executableTestFiles) > 0 {
		return errors.Errorf("python test files cannot be executable because "+
			"they will be skipped. Fix permissions of %v", executableTestFiles)
	}
	return nil
}

// CheckYAMLNotExecutable checks that no .yml or .yaml files are executable.
func CheckYAMLNotExecutable() error {
	if runtime.GOOS == "windows" {
		// Skip windows because it doesn't have POSIX permissions.
		return nil
	}

	executableYAMLFiles, err := FindFilesRecursive(func(path string, info os.FileInfo) bool {
		switch filepath.Ext(path) {
		default:
			return false
		case ".yml", ".yaml":
			return info.Mode().Perm()&0111 > 0
		}
	})
	if err != nil {
		return errors.Wrap(err, "failed search for YAML files")
	}

	if len(executableYAMLFiles) > 0 {
		return errors.Errorf("YAML files cannot be executable. Fix "+
			"permissions of %v", executableYAMLFiles)

	}
	return nil
}

// GoVet vets the .go source code using 'go vet'.
func GoVet() error {
	err := sh.RunV("go", "vet", "./...")
	return errors.Wrap(err, "failed running go vet, please fix the issues reported")
}

// CheckLicenseHeaders checks license headers in .go files.
func CheckLicenseHeaders() error {
	fmt.Println(">> fmt - go-licenser: Checking for missing headers")

	mg.Deps(InstallGoLicenser)

	var license string
	switch BeatLicense {
	case "ASL2", "ASL 2.0":
		license = "ASL2"
	case "Elastic", "Elastic License":
		license = "Elastic"
	default:
		return errors.Errorf("unknown license type %v", BeatLicense)
	}

	licenser := gotool.Licenser
	return licenser(licenser.Check(), licenser.License(license))
}

// CheckDashboardsFormat checks the format of dashboards
func CheckDashboardsFormat() error {
	dashboardSubDir := "/_meta/kibana/"
	dashboardFiles, err := FindFilesRecursive(func(path string, _ os.FileInfo) bool {
		if strings.HasPrefix(path, "vendor") {
			return false
		}
		return strings.Contains(filepath.ToSlash(path), dashboardSubDir) && strings.HasSuffix(path, ".json")
	})
	if err != nil {
		return errors.Wrap(err, "failed to find dashboards")
	}

	hasErrors := false
	for _, file := range dashboardFiles {
		d, err := ioutil.ReadFile(file)
		if err != nil {
			return errors.Wrapf(err, "failed to read dashboard file %s", file)
		}

		if checkDashboardForErrors(file, d) {
			hasErrors = true
		}
	}

	if hasErrors {
		return errors.New("there are format errors in dashboards")
	}
	return nil
}

func checkDashboardForErrors(file string, d []byte) bool {
	if len(bytes.TrimRight(d, "\n")) == 0 {
		return false
	}
	var hasErrors bool
	var dashboard DashboardObject
	err := json.Unmarshal(d, &dashboard)
	if err != nil {
		fmt.Println(errors.Wrapf(err, "failed to parse dashboard from %s", file).Error())
		return true
	}

	module := moduleNameFromDashboard(file)
	err = dashboard.CheckFormat(module)
	if err != nil {
		hasErrors = true
		fmt.Printf(">> Dashboard format - %s:\n", file)
		fmt.Println("  ", err)
	}

	replaced := dashboards.ReplaceIndexInDashboardObject("my-test-index-*", d)
	if bytes.Contains(replaced, []byte(BeatName+"-*")) {
		hasErrors = true
		fmt.Printf(">> Cannot modify all index pattern references in dashboard - %s\n", file)
		fmt.Println("Please edit the dashboard override function named ReplaceIndexInDashboardObject in libbeat.")
		fmt.Println(string(replaced))
	}

	return hasErrors
}

func moduleNameFromDashboard(path string) string {
	moduleDir := filepath.Clean(filepath.Join(filepath.Dir(path), "../../../.."))
	return filepath.Base(moduleDir)
}

// DashboardObject is a dashboard
type DashboardObject struct {
	Version    string `json:"version"`
	Type       string `json:"type"`
	Attributes struct {
		Description           string `json:"description"`
		Title                 string `json:"title"`
		KibanaSavedObjectMeta *struct {
			SearchSourceJSON struct {
				Index *string `json:"index"`
			} `json:"searchSourceJSON,omitempty"`
		} `json:"kibanaSavedObjectMeta"`
		VisState *struct {
			Params *struct {
				IndexPattern *string `json:"index_pattern"`
				Controls     []struct {
					IndexPattern *string
				} `json:"controls"`
			} `json:"params"`
		} `json:"visState,omitempty"`
	} `json:"attributes"`
	References []struct {
		Type string `json:"type"`
		ID   string `json:"id"`
	} `json:"references"`
}

var (
	visualizationTitleRegexp = regexp.MustCompile(`^.+\[([^\s]+) (.+)\]( ECS)?$`)
	dashboardTitleRegexp     = regexp.MustCompile(`^\[([^\s]+) (.+)\].+$`)
)

// CheckFormat checks the format of a dashboard
func (d *DashboardObject) CheckFormat(module string) error {
	switch d.Type {
	case "dashboard":
		if d.Attributes.Description == "" {
			return errors.Errorf("empty description on dashboard '%s'", d.Attributes.Title)
		}
		if err := checkTitle(dashboardTitleRegexp, d.Attributes.Title, module); err != nil {
			return errors.Wrapf(err, "expected title with format '[%s Module] Some title', found '%s'", strings.Title(BeatName), d.Attributes.Title)
		}
	case "visualization":
		if err := checkTitle(visualizationTitleRegexp, d.Attributes.Title, module); err != nil {
			return errors.Wrapf(err, "expected title with format 'Some title [%s Module]', found '%s'", strings.Title(BeatName), d.Attributes.Title)
		}
	}

	expectedIndexPattern := strings.ToLower(BeatName) + "-*"
	if err := checkDashboardIndexPattern(expectedIndexPattern, d); err != nil {
		return errors.Wrapf(err, "expected index pattern reference '%s'", expectedIndexPattern)
	}
	return nil
}

func checkTitle(re *regexp.Regexp, title string, module string) error {
	match := re.FindStringSubmatch(title)
	if len(match) < 3 {
		return errors.New("title doesn't match pattern")
	}
	beatTitle := strings.Title(BeatName)
	if match[1] != beatTitle {
		return errors.Errorf("expected: '%s', found: '%s'", beatTitle, match[1])
	}

	// Compare case insensitive, and ignore spaces and underscores in module names
	replacer := strings.NewReplacer("_", "", " ", "")
	expectedModule := replacer.Replace(strings.ToLower(module))
	foundModule := replacer.Replace(strings.ToLower(match[2]))
	if expectedModule != foundModule {
		return errors.Errorf("expected module name (%s), found '%s'", module, match[2])
	}
	return nil
}

func checkDashboardIndexPattern(expectedIndex string, o *DashboardObject) error {
	if objectMeta := o.Attributes.KibanaSavedObjectMeta; objectMeta != nil {
		if index := objectMeta.SearchSourceJSON.Index; index != nil && *index != expectedIndex {
			return errors.Errorf("unexpected index pattern reference found in object meta: `%s` in visualization `%s`", *index, o.Attributes.Title)
		}
	}
	if visState := o.Attributes.VisState; visState != nil {
		for _, control := range visState.Params.Controls {
			if index := control.IndexPattern; index != nil && *index != expectedIndex {
				return errors.Errorf("unexpected index pattern reference found in visualization state: `%s` in visualization `%s`", *index, o.Attributes.Title)
			}
		}
		if index := visState.Params.IndexPattern; index != nil && *index != expectedIndex {
			return errors.Errorf("unexpected index pattern reference found in visualization state params: `%s` in visualization `%s`", *index, o.Attributes.Title)
		}
	}
	for _, reference := range o.References {
		if reference.Type == "index-pattern" && reference.ID != expectedIndex {
			return errors.Errorf("unexpected reference to index pattern `%s`", reference.ID)
		}
	}
	return nil
}
