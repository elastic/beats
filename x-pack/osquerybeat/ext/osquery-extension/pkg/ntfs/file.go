// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package ntfs

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	elasticntfsfile "github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/tables/generated/ntfs/elastic_ntfs_file"

	"www.velocidex.com/golang/go-ntfs/parser"
)

type entryType string

const (
	entryTypeFile      entryType = "file"
	entryTypeDirectory entryType = "directory"
)

type fileNode struct {
	mftEntry *parser.MFT_ENTRY
	volume   *Volume
	parent   *fileNode
	name     string // this component's filename, used to build the full path
}

func (f *fileNode) BuildFullPath() string {
	// Walk the parent chain collecting names, then reverse to build path.
	var parts []string
	cur := f
	for cur != nil {
		if cur.name != "" {
			parts = append(parts, cur.name)
		}
		cur = cur.parent
	}
	slices.Reverse(parts)
	return f.volume.DriveLetter + ":\\" + strings.Join(parts, "\\")
}

func (f *fileNode) Materialize() (*elasticntfsfile.Result, error) {
	if f.mftEntry == nil {
		return nil, fmt.Errorf("FileInfo has no metadata to load from")
	}

	ntfsCtx, err := f.volume.ntfsContext()
	if err != nil {
		return nil, err
	}

	result := &elasticntfsfile.Result{
		Drive:  f.volume.DriveLetter,
		Device: f.volume.Device,
		Inode:  int64(f.mftEntry.Record_number()),
	}

	result.Path = f.BuildFullPath()
	result.Directory = filepath.Dir(result.Path)
	if f.parent != nil {
		result.ParentInode = int64(f.parent.mftEntry.Record_number())
	}

	if f.mftEntry.IsDir(ntfsCtx) {
		result.Type = "directory"
	} else {
		result.Type = "file"
	}
	if f.mftEntry.Flags().IsSet("ALLOCATED") {
		result.Active = 1
	} else {
		result.Active = 0
	}

	result.HardLinkCount = int32(f.mftEntry.Link_count())

	// Filename from $FILE_NAME attributes in the MFT entry.
	// NTFS files can have multiple $FILE_NAME attributes with different namespaces:
	//   0=POSIX, 1=Win32, 2=DOS, 3=DOS+Win32
	// The DOS namespace entry has stale timestamps (frozen at creation and rarely updated),
	// while Win32 and Win32+DOS namespaces reflect the most recent file operations.
	// Prefer Win32 (1) or Win32+DOS (3); fall back to the first attribute if neither is present.
	fileNames := f.mftEntry.FileName(ntfsCtx)
	fn := preferredFileName(fileNames)
	if fn != nil {
		result.Filename = fn.Name()
		result.AllocatedSize = int64(fn.Allocated_size())
		result.FnBtime = fn.Created().Time.Unix()
		result.FnMtime = fn.File_modified().Time.Unix()
		result.FnCtime = fn.Mft_modified().Time.Unix()
		result.FnAtime = fn.File_accessed().Time.Unix()
	}

	// Timestamps, flags, security ID, and owner ID from $STANDARD_INFORMATION.
	if si, err := f.mftEntry.StandardInformation(ntfsCtx); err == nil {
		result.Btime = si.Create_time().Time.Unix()
		result.Mtime = si.File_altered_time().Time.Unix()
		result.Ctime = si.Mft_altered_time().Time.Unix()
		result.Atime = si.File_accessed_time().Time.Unix()
		result.Flags = int32(si.Flags().Value)
		result.SecurityId = int32(si.Sid())
		result.OwnerId = int32(si.Owner_id())
	}

	// Size, ADS presence, and ObjectID from attribute enumeration.
	for _, attr := range f.mftEntry.EnumerateAttributes(ntfsCtx) {
		switch attr.Type().Value {
		case 128: // $DATA
			if attr.Name() == "" {
				result.Size = int64(attr.DataSize())
			} else {
				result.Ads = 1
			}
		case 64: // $OBJECT_ID — first 16 bytes are a GUID
			var buf [16]byte
			if n, err := attr.Data(ntfsCtx).ReadAt(buf[:], 0); n == 16 && err == nil {
				result.ObjectId = guidStringFromBytes(buf)
			}
		}
	}
	return result, nil
}

// preferredFileName returns the Win32 or Win32+DOS namespace $FILE_NAME attribute from
// the list, falling back to the first entry if neither is present.
// preferredFileName selects the best $FILE_NAME attribute for timestamps,
// filename, and allocated size. Priority:
//  1. Win32 (namespace 1) with non-zero allocated size
//  2. Win32+DOS (namespace 3) with non-zero allocated size
//  3. Last POSIX entry (namespace 0) — no Win32 family entry has valid size data;
//     take the last one in MFT order, which tends to be the most recently written
//
// TODO: Should we consider adding a FILE_NAME table that returns all FILE_NAME attributes for a given MFT entry, would have an inode or path constraint
func preferredFileName(fileNames []*parser.FILE_NAME) *parser.FILE_NAME {
	if len(fileNames) == 0 {
		return nil
	}
	for _, fn := range fileNames {
		if fn.NameType().Value == 1 && fn.Allocated_size() > 0 { // Win32
			return fn
		}
	}
	for _, fn := range fileNames {
		if fn.NameType().Value == 3 && fn.Allocated_size() > 0 { // Win32+DOS
			return fn
		}
	}
	// Fall back to last entry — matches TSK/ToB iteration order
	return fileNames[len(fileNames)-1]
}

func parentInode(volumeInfo *Volume, mftEntry *parser.MFT_ENTRY) (int64, error) {
	ntfsCtx, err := volumeInfo.ntfsContext()
	if err != nil {
		return 0, fmt.Errorf("failed to get NTFS context: %w", err)
	}
	fn := preferredFileName(mftEntry.FileName(ntfsCtx))
	if fn == nil {
		return 0, fmt.Errorf("MFT entry has no FILE_NAME attributes")
	}
	parentInode := int64(fn.MftReference() & 0xFFFFFFFFFFFF) // mask off sequence number
	return parentInode, nil
}

func NewFileNode(volumeInfo *Volume, mftEntry *parser.MFT_ENTRY, name string, parent *fileNode) (*fileNode, error) {
	if mftEntry == nil {
		return nil, fmt.Errorf("invalid MFT entry provided")
	}

	fi := &fileNode{
		mftEntry: mftEntry,
		volume:   volumeInfo,
		name:     name,
		parent:   parent,
	}

	return fi, nil
}
