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

//go:build linux

package kprobes

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func (d *dEntryCache) Dump(path string) error {
	fileDump, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}

	defer fileDump.Close()

	for _, entry := range d.index {
		if _, err = fileDump.WriteString(entry.Path() + "\n"); err != nil {
			return err
		}
	}

	return nil
}

func Test_DirEntryCache_Add(t *testing.T) {
	cases := []struct {
		name     string
		parent   *dEntry
		children map[string]*dEntry
	}{
		{
			"dentry_no_children",
			&dEntry{
				Depth:    0,
				Name:     "test",
				Ino:      1,
				DevMajor: 1,
				DevMinor: 1,
			},
			nil,
		},
		{
			"dentry_with_children",
			&dEntry{
				Depth:    1,
				Name:     "test",
				Ino:      1,
				DevMajor: 1,
				DevMinor: 1,
			},
			map[string]*dEntry{
				"child1": {
					Depth:    2,
					Name:     "child1",
					Ino:      2,
					DevMajor: 1,
					DevMinor: 1,
				},
				"child2": {
					Depth:    2,
					Name:     "child2",
					Ino:      3,
					DevMajor: 1,
					DevMinor: 1,
				},
			},
		},
		{
			// we shouldn't add nil dentries
			"check_nil_dentry_add",
			nil,
			nil,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cache := newDirEntryCache()

			expectedLen := 0
			if c.parent != nil {
				expectedLen++
				if c.children != nil {
					for _, child := range c.children {
						c.parent.AddChild(child)
						expectedLen++
					}
				}
			}

			cache.Add(c.parent, nil)

			require.Len(t, cache.index, expectedLen)
			if c.parent != nil {
				require.Equal(t, c.parent, cache.index[dKey{
					Ino:      c.parent.Ino,
					DevMajor: c.parent.DevMajor,
					DevMinor: c.parent.DevMinor,
				}])
			}

			if c.children != nil {
				for _, child := range c.children {
					require.Equal(t, child, cache.index[dKey{
						Ino:      child.Ino,
						DevMajor: child.DevMajor,
						DevMinor: child.DevMinor,
					}])
				}
			}
		})
	}
}

func Test_DirEntryCache_Get(t *testing.T) {
	cases := []struct {
		name  string
		key   dKey
		entry *dEntry
	}{
		{
			"dentry_exists",
			dKey{
				Ino:      1,
				DevMajor: 1,
				DevMinor: 1,
			},
			&dEntry{
				Depth:    1,
				Parent:   nil,
				Children: nil,
				Name:     "test",
				Ino:      1,
				DevMajor: 1,
				DevMinor: 1,
			},
		},
		{
			"dentry_non_existent",
			dKey{
				Ino:      10000,
				DevMajor: 2,
				DevMinor: 3,
			},
			nil,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cache := newDirEntryCache()
			cache.Add(c.entry, nil)

			cacheEntry := cache.Get(c.key)
			require.Equal(t, c.entry, cacheEntry)
		})
	}
}

func Test_DirEntryCache_Remove(t *testing.T) {
	cases := []struct {
		name             string
		parent           *dEntry
		children         dEntryChildren
		childrenChildren dEntryChildren
	}{
		{
			"dentry_no_children",
			&dEntry{
				Parent:   nil,
				Name:     "test",
				Ino:      1,
				DevMajor: 1,
				DevMinor: 1,
			},
			nil,
			nil,
		},
		{
			"dentry_with_children",
			&dEntry{
				Parent:   nil,
				Name:     "test",
				Ino:      1,
				DevMajor: 1,
				DevMinor: 1,
			},
			dEntryChildren{
				"child1": {
					Parent:   nil,
					Name:     "child1",
					Ino:      4,
					DevMajor: 1,
					DevMinor: 1,
				},
				"child2": {
					Parent:   nil,
					Name:     "child2",
					Ino:      7,
					DevMajor: 1,
					DevMinor: 1,
				},
			},
			nil,
		},
		{
			"dentry_with_children_children",
			&dEntry{
				Parent:   nil,
				Name:     "test",
				Ino:      1,
				DevMajor: 1,
				DevMinor: 1,
			},
			dEntryChildren{
				"child1": {
					Parent:   nil,
					Name:     "child1",
					Ino:      4,
					DevMajor: 1,
					DevMinor: 1,
				},
				"child2": {
					Parent:   nil,
					Name:     "child2",
					Ino:      7,
					DevMajor: 1,
					DevMinor: 1,
				},
			},
			dEntryChildren{
				"child_child1": {
					Parent:   nil,
					Name:     "child_child1",
					Ino:      10,
					DevMajor: 1,
					DevMinor: 1,
				},
			},
		},
		{
			"dentry_nil",
			nil,
			nil,
			nil,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cache := newDirEntryCache()
			cache.Add(c.parent, nil)

			if c.parent != nil {
				if c.children != nil {
					for _, child := range c.children {
						cache.Add(child, c.parent)
					}
				}

				if len(c.children) > 0 && c.childrenChildren != nil {
					for _, childrenChildrenParent := range c.children {
						for _, child := range c.childrenChildren {
							cache.Add(child, childrenChildrenParent)
						}
						break
					}
				}
			}

			removedEntry := cache.Remove(c.parent)
			require.Len(t, cache.index, 0)
			require.Equal(t, c.parent, removedEntry)

			removedEntry.Release()
			if removedEntry != nil {
				require.Nil(t, removedEntry.Children)
			}
		})
	}
}

func Test_DirEntryCache_MoveFrom(t *testing.T) {
	cases := []struct {
		name     string
		tid      uint64
		parent   *dEntry
		children dEntryChildren
	}{
		{
			"dentry_move",
			1,
			&dEntry{
				Name:     "test",
				Depth:    0,
				Ino:      1,
				DevMajor: 1,
				DevMinor: 1,
			},
			dEntryChildren{
				"child1": {
					Name:     "child1",
					Ino:      4,
					DevMajor: 1,
					DevMinor: 1,
				},
				"child2": {
					Name:     "child2",
					Ino:      7,
					DevMajor: 1,
					DevMinor: 1,
				},
			},
		},
		{
			"dentry_nil",
			1,
			nil,
			nil,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cache := newDirEntryCache()
			cache.Add(c.parent, nil)

			if c.parent != nil {
				if c.children != nil {
					for _, child := range c.children {
						cache.Add(child, c.parent)
					}
				}
			}

			cache.MoveFrom(c.tid, c.parent)

			require.Empty(t, cache.index)

			if c.parent == nil {
				require.Len(t, cache.moveCache, 0)
				return
			}

			require.Len(t, cache.moveCache, 1)

			moveEntry, exists := cache.moveCache[c.tid]
			require.True(t, exists)
			require.Equal(t, c.parent, moveEntry)
			if c.children != nil {
				require.NotNil(t, c.parent.Children)
				for _, child := range moveEntry.Children {
					require.Equal(t, c.parent.Depth+1, child.Depth)
				}
			} else {
				require.Nil(t, c.parent.Children)
			}
		})
	}
}

func Test_DirEntryCache_MoveTo(t *testing.T) {
	cases := []struct {
		name         string
		srcTid       uint64
		dstTid       uint64
		entry        *dEntry
		children     dEntryChildren
		targetParent *dEntry
		newFileName  string
		pathsToSee   []string
		err          error
	}{
		{
			"dentry_move",
			1,
			1,
			&dEntry{
				Name:     "test",
				Depth:    0,
				Ino:      1,
				DevMajor: 1,
				DevMinor: 1,
			},
			dEntryChildren{
				"child1": {
					Name:     "child1",
					Ino:      4,
					DevMajor: 1,
					DevMinor: 1,
				},
				"child2": {
					Name:     "child2",
					Ino:      7,
					DevMajor: 1,
					DevMinor: 1,
				},
			},
			&dEntry{
				Name:     "test2",
				Depth:    0,
				Ino:      10,
				DevMajor: 1,
				DevMinor: 1,
			},
			"test3",
			[]string{
				"test2/test3",
				"test2/test3/child1",
				"test2/test3/child2",
			},
			nil,
		},
		{
			"dentry_not_found",
			1,
			2,
			&dEntry{
				Name:     "test",
				Depth:    0,
				Ino:      1,
				DevMajor: 1,
				DevMinor: 1,
			},
			nil,
			nil,
			"",
			nil,
			nil,
		},
		{
			"callback_err",
			1,
			1,
			&dEntry{
				Name:     "test",
				Depth:    0,
				Ino:      1,
				DevMajor: 1,
				DevMinor: 1,
			},
			nil,
			nil,
			"",
			nil,
			errors.New("error"),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var movedPaths []string

			cache := newDirEntryCache()
			if c.entry != nil {
				if c.children != nil {
					for _, child := range c.children {
						c.entry.AddChild(child)
					}
				}
				cache.moveCache[c.srcTid] = c.entry
			}

			movedEntry, err := cache.MoveTo(c.dstTid, c.targetParent, c.newFileName, func(path string) error {
				if c.err != nil {
					return c.err
				}

				movedPaths = append(movedPaths, path)
				return nil
			})
			if c.err == nil {
				require.Nil(t, err)
			} else {
				require.ErrorIs(t, err, c.err)
			}

			if c.srcTid == c.dstTid {
				require.True(t, movedEntry)
				require.Empty(t, cache.moveCache)
			} else {
				require.False(t, movedEntry)
				require.NotEmpty(t, cache.moveCache)
			}
			require.ElementsMatch(t, c.pathsToSee, movedPaths)
		})
	}
}

func Test_DirEntryCache_MoveClear(t *testing.T) {
	cases := []struct {
		name   string
		srcTid uint64
		dstTid uint64
		entry  *dEntry
	}{
		{
			"dentry_move",
			1,
			1,
			&dEntry{
				Name:     "test",
				Depth:    0,
				Ino:      1,
				DevMajor: 1,
				DevMinor: 1,
			},
		},
		{
			"dentry_not_found",
			1,
			2,
			&dEntry{
				Name:     "test",
				Depth:    0,
				Ino:      1,
				DevMajor: 1,
				DevMinor: 1,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cache := newDirEntryCache()
			if c.entry != nil {
				cache.moveCache[c.srcTid] = c.entry
			}

			cache.MoveClear(c.dstTid)

			if c.srcTid == c.dstTid {
				require.Empty(t, cache.moveCache)
			} else {
				require.NotEmpty(t, cache.moveCache)
			}
		})
	}
}

func Test_DirEntryCache_GetChild(t *testing.T) {
	cases := []struct {
		name      string
		entry     *dEntry
		children  dEntryChildren
		childName string
	}{
		{
			"dentry_with_children",
			&dEntry{
				Name:     "test",
				Depth:    0,
				Ino:      1,
				DevMajor: 1,
				DevMinor: 1,
			},
			dEntryChildren{
				"child1": {
					Name:     "child1",
					Ino:      4,
					DevMajor: 1,
					DevMinor: 1,
				},
				"child2": {
					Name:     "child2",
					Ino:      7,
					DevMajor: 1,
					DevMinor: 1,
				},
			},
			"child1",
		},
		{
			"dentry_no_children",
			&dEntry{
				Name:     "test",
				Depth:    0,
				Ino:      1,
				DevMajor: 1,
				DevMinor: 1,
			},
			nil,
			"child1",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for _, child := range c.children {
				c.entry.AddChild(child)
			}

			childEntry := c.entry.GetChild(c.childName)

			if c.children == nil {
				require.Nil(t, childEntry)
			} else {
				require.NotNil(t, childEntry)
			}
		})
	}
}
