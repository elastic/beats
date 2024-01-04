package kprobes

import (
	"path/filepath"
	"slices"
)

type dKey struct {
	Ino      uint64
	DevMajor uint32
	DevMinor uint32
}

type dEntryChildren map[string]*dEntry

type dEntry struct {
	Parent   *dEntry
	Depth    uint32
	Children dEntryChildren
	Name     string
	Ino      uint64
	DevMajor uint32
	DevMinor uint32
}

func (d *dEntry) GetParent() *dEntry {
	if d == nil {
		return nil
	}

	return d.Parent
}

func (d *dEntry) Path() string {
	if d == nil {
		return ""
	}

	var pathTokens []string
	startEntry := d
	for startEntry != nil {
		pathTokens = append(pathTokens, startEntry.Name)
		startEntry = startEntry.Parent
	}
	slices.Reverse(pathTokens)
	finalPath := filepath.Join(pathTokens...)
	return finalPath
}

// releaseRecursive recursive func to satisfy the needs of Release.
func releaseRecursive(val *dEntry) {
	for _, child := range val.Children {
		releaseRecursive(child)
		delete(val.Children, child.Name)
	}

	val.Children = nil
	val.Parent = nil
	val = nil
	return
}

// Release releases the resources associated with the given dEntry and all its children.
func (d *dEntry) Release() {
	if d == nil {
		return
	}

	releaseRecursive(d)
}

func (d *dEntry) RemoveChild(name string) {
	if d == nil || d.Children == nil {
		return
	}

	delete(d.Children, name)
}

// AddChild adds a child entry to the dEntry.
func (d *dEntry) AddChild(child *dEntry) {
	if d == nil || child == nil {
		return
	}

	if d.Children == nil {
		d.Children = make(map[string]*dEntry)
	}

	child.Parent = d
	child.Depth = d.Depth + 1

	d.Children[child.Name] = child
}

// GetChild returns the child entry with the given name, if it exists. Otherwise, nil is returned.
func (d *dEntry) GetChild(name string) *dEntry {
	if d == nil || d.Children == nil {
		return nil
	}

	child, exists := d.Children[name]
	if !exists {
		return nil
	}

	return child
}
