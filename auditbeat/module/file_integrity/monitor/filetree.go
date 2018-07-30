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

package monitor

import (
	"fmt"
	"os"
	path_pkg "path"
	"strings"
)

// VisitOrder is a two-valued flag used to control how directories are visited.
type VisitOrder int8

const (
	// PreOrder has directories visited before their contents.
	PreOrder VisitOrder = iota
	// PostOrder has directories visited after their contents.
	PostOrder
)

var (
	// PathSeparator can be used to override the operating system separator.
	PathSeparator = string(os.PathSeparator)
)

// FileTree represents a directory in a filesystem-tree structure.
type FileTree map[string]FileTree

// VisitFunc is the type for a callback to visit the entries on a directory
// and its subdirectories.
type VisitFunc func(path string, isDir bool) error

// AddFile adds a file to a FileTree. If the path includes subdirectories
// they are created as necessary.
func (tree FileTree) AddFile(path string) error {
	return tree.add(path_pkg.Clean(path), nil)
}

// AddDir adds a directory to a FileTree. If the path includes subdirectories
// they are created as necessary.
func (tree FileTree) AddDir(path string) error {
	return tree.add(path_pkg.Clean(path), FileTree{})
}

// Remove an entry from a FileTree.
func (tree FileTree) Remove(path string) error {
	components := strings.Split(path, PathSeparator)
	last := -1
	for pos := len(components) - 1; pos >= 0; pos-- {
		if len(components[pos]) != 0 {
			last = pos
			break
		}
	}
	if last > 0 {
		subtree, err := tree.getByComponents(path, components[:last])
		if err != nil {
			return err
		}
		delete(subtree, components[last])
	}
	return nil
}

// Visit calls the callback function for the given path and recursively all its
// contents, if a directory path is passed.
func (tree FileTree) Visit(path string, order VisitOrder, fn VisitFunc) error {
	entry, err := tree.At(path)
	if err != nil {
		return err
	}
	return entry.visitDirRecursive(path_pkg.Clean(path), order, fn)
}

// At returns a new FileTree rooted at the given path.
func (tree FileTree) At(path string) (FileTree, error) {
	return tree.getByComponents(path, strings.Split(path, PathSeparator))
}

func (tree FileTree) add(path string, value FileTree) error {
	components := strings.Split(path, PathSeparator)
	dir, last := tree, len(components)-1
	for i := 0; i < last; i++ {
		if len(components[i]) == 0 {
			continue
		}
		if next, exists := dir[components[i]]; exists {
			if next == nil {
				return fmt.Errorf("directory expected: '%s' in %s", components[i], path)
			}
			dir = next
		} else {
			newDir := FileTree{}
			dir[components[i]] = newDir
			dir = newDir
		}
	}
	dir[components[last]] = value
	return nil
}

func (tree FileTree) getByComponents(path string, components []string) (FileTree, error) {
	dir, exists := tree, false
	for _, item := range components {
		if len(item) != 0 {
			if dir == nil {
				// previous component is a file, not a directory
				return nil, fmt.Errorf("path component %s is a file: %s", item, path)
			}
			if dir, exists = dir[item]; !exists {
				return nil, fmt.Errorf("path component %s not found in %s", item, path)
			}
		}
	}
	return dir, nil
}

func (tree FileTree) visitDirRecursive(path string, order VisitOrder, fn VisitFunc) error {
	if tree == nil {
		return fn(path, false)
	}
	if order == PreOrder {
		if err := fn(path, true); err != nil {
			return err
		}
	}
	for name, content := range tree {
		fullpath := path_pkg.Join(path, name)
		if content == nil {
			if err := fn(fullpath, false); err != nil {
				return err
			}
		} else {
			if err := content.visitDirRecursive(fullpath, order, fn); err != nil {
				return err
			}
		}
	}
	if order == PostOrder {
		return fn(path, true)
	}
	return nil
}
