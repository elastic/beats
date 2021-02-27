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
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"github.com/pkg/errors"
)

// Copy copies a file or a directory (recursively) and preserves the permissions.
func Copy(src, dest string) error {
	copy := &CopyTask{Source: src, Dest: dest}
	return copy.Execute()
}

// CopyTask copies a file or directory (recursively) and preserves the permissions.
type CopyTask struct {
	Source   string           // Source directory or file.
	Dest     string           // Destination directory or file.
	Mode     os.FileMode      // Mode to use for copied files. Defaults to preserve permissions.
	DirMode  os.FileMode      // Mode to use for copied dirs. Defaults to preserve permissions.
	Exclude  []string         // Exclude paths that match these regular expressions.
	excludes []*regexp.Regexp // Compiled exclude regexes.
}

// Execute executes the copy and returns an error of there is a failure.
func (t *CopyTask) Execute() error {
	if err := t.init(); err != nil {
		return errors.Wrap(err, "copy failed")
	}

	info, err := os.Stat(t.Source)
	if err != nil {
		return errors.Wrapf(err, "copy failed: cannot stat source file %v", t.Source)
	}

	return errors.Wrap(t.recursiveCopy(t.Source, t.Dest, info), "copy failed")
}

func (t *CopyTask) init() error {
	for _, excl := range t.Exclude {
		re, err := regexp.Compile(excl)
		if err != nil {
			return errors.Wrapf(err, "bad exclude pattern %v", excl)
		}
		t.excludes = append(t.excludes, re)
	}
	return nil
}

func (t *CopyTask) isExcluded(src string) bool {
	for _, excl := range t.excludes {
		if match := excl.MatchString(filepath.ToSlash(src)); match {
			return true
		}
	}
	return false
}

func (t *CopyTask) recursiveCopy(src, dest string, info os.FileInfo) error {
	if info.IsDir() {
		return t.dirCopy(src, dest, info)
	}
	return t.fileCopy(src, dest, info)
}

func (t *CopyTask) fileCopy(src, dest string, info os.FileInfo) error {
	if t.isExcluded(src) {
		return nil
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	if !info.Mode().IsRegular() {
		return errors.Errorf("failed to copy source file because it is not a " +
			"regular file")
	}

	mode := t.Mode
	if mode == 0 {
		mode = info.Mode()
	}
	destFile, err := os.OpenFile(createDir(dest),
		os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode&os.ModePerm)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err = io.Copy(destFile, srcFile); err != nil {
		return err
	}
	return destFile.Close()
}

func (t *CopyTask) dirCopy(src, dest string, info os.FileInfo) error {
	if t.isExcluded(src) {
		return nil
	}

	mode := t.DirMode
	if mode == 0 {
		mode = info.Mode()
	}
	if err := os.MkdirAll(dest, mode&os.ModePerm); err != nil {
		return errors.Wrap(err, "failed creating dirs")
	}

	contents, err := ioutil.ReadDir(src)
	if err != nil {
		return errors.Wrapf(err, "failed to read dir %v", src)
	}

	for _, info := range contents {
		srcFile := filepath.Join(src, info.Name())
		destFile := filepath.Join(dest, info.Name())
		if err = t.recursiveCopy(srcFile, destFile, info); err != nil {
			return errors.Wrapf(err, "failed to copy %v to %v", srcFile, destFile)
		}
	}

	return nil
}
