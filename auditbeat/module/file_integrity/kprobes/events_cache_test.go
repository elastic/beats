package kprobes

import (
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func (d *dEntryCache) Dump(path string) error {
	fileDump, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
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
		entry    *dEntry
		children map[string]*dEntry
	}{
		{
			"dentry_no_children",
			&dEntry{
				Parent:   nil,
				Children: nil,
				Name:     "test",
				Ino:      1,
				DevMajor: 2,
				DevMinor: 3,
			},
			nil,
		},
		{
			"dentry_with_children",
			&dEntry{
				Parent:   nil,
				Name:     "test",
				Ino:      1,
				DevMajor: 2,
				DevMinor: 3,
			},
			map[string]*dEntry{
				"child1": {
					Parent:   nil,
					Children: nil,
					Name:     "child1",
					Ino:      4,
					DevMajor: 5,
					DevMinor: 6,
				},
				"child2": {
					Parent:   nil,
					Children: nil,
					Name:     "child2",
					Ino:      7,
					DevMajor: 8,
					DevMinor: 9,
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
			if c.entry != nil {
				expectedLen++
				if c.children != nil {
					c.entry.Children = make(dEntryChildren)
					for _, child := range c.children {
						c.entry.Children[child.Name] = child
						child.Parent = c.entry
						expectedLen++
					}
				}
			}

			cache.Add(c.entry)

			require.Len(t, cache.index, expectedLen)
			if c.entry != nil {
				require.Equal(t, c.entry, cache.index[dKey{
					Ino:      c.entry.Ino,
					DevMajor: c.entry.DevMajor,
					DevMinor: c.entry.DevMinor,
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
				DevMajor: 2,
				DevMinor: 3,
			},
			&dEntry{
				Parent:   nil,
				Children: nil,
				Name:     "test",
				Ino:      1,
				DevMajor: 2,
				DevMinor: 3,
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
			if c.entry != nil {
				cache.index = make(dEntriesIndex)
				cache.index[c.key] = c.entry
			}

			cacheEntry := cache.Get(c.key)
			require.Equal(t, c.entry, cacheEntry)
		})
	}
}

func Test_DirEntryCache_Remove(t *testing.T) {
	cases := []struct {
		name             string
		entry            *dEntry
		children         dEntryChildren
		childrenChildren dEntryChildren
	}{
		{
			"dentry_no_children",
			&dEntry{
				Parent:   nil,
				Name:     "test",
				Ino:      1,
				DevMajor: 2,
				DevMinor: 3,
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
				DevMajor: 2,
				DevMinor: 3,
			},
			dEntryChildren{
				"child1": {
					Parent:   nil,
					Name:     "child1",
					Ino:      4,
					DevMajor: 5,
					DevMinor: 6,
				},
				"child2": {
					Parent:   nil,
					Name:     "child2",
					Ino:      7,
					DevMajor: 8,
					DevMinor: 9,
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
				DevMajor: 2,
				DevMinor: 3,
			},
			dEntryChildren{
				"child1": {
					Parent:   nil,
					Name:     "child1",
					Ino:      4,
					DevMajor: 5,
					DevMinor: 6,
				},
				"child2": {
					Parent:   nil,
					Name:     "child2",
					Ino:      7,
					DevMajor: 8,
					DevMinor: 9,
				},
			},
			dEntryChildren{
				"child_child1": {
					Parent:   nil,
					Name:     "child_child1",
					Ino:      10,
					DevMajor: 11,
					DevMinor: 12,
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

			if c.entry != nil {
				if c.children != nil {
					c.entry.Children = c.children
					for _, child := range c.children {
						c.entry.Children[child.Name] = child
						child.Parent = c.entry
					}
				}

				if len(c.entry.Children) > 0 && c.childrenChildren != nil {
					for key := range c.entry.Children {
						c.entry.Children[key].Children = c.childrenChildren
						for _, child := range c.childrenChildren {
							c.entry.Children[key].Children[child.Name] = child
							child.Parent = c.entry.Children[key]
						}
						break
					}
				}
			}

			cache.Add(c.entry)
			removedEntry := cache.Remove(c.entry)
			require.Len(t, cache.index, 0)
			require.Equal(t, c.entry, removedEntry)
		})
	}
}
