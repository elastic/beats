// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package ntfs

import (
	"context"
	"fmt"
	"path"
	"strconv"
	"strings"

	"www.velocidex.com/golang/go-ntfs/parser"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/client"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/filters"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
	elasticntfsfile "github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/tables/generated/ntfs/elastic_ntfs_file"
	"github.com/osquery/osquery-go/plugin/table"
)

func (v *Volume) explodePath(p string, remove_ads bool) ([]string, error) {
	if p == "" {
		return nil, fmt.Errorf("path is empty")
	}

	p = strings.ReplaceAll(p, "\\", "/")
	if strings.HasPrefix(p[1:], ":/") {
		if !strings.HasPrefix(p, v.DriveLetter) {
			return nil, fmt.Errorf("path %s does not start with drive letter %s", p, v.DriveLetter)
		}
		p = strings.TrimPrefix(p, v.DriveLetter+":/")
	}
	if remove_ads {
		p = strings.Split(p, ":")[0] // remove ADS if any as not needed
	}
	if p == "" {
		return nil, fmt.Errorf("path is empty after removing drive letter and ADS")
	}
	p = path.Clean(p)
	return strings.Split(p, "/"), nil
}

// childrenMatching lists all direct children of parent whose names satisfy predicate.
// It owns the Dir → GetMFT → NewFileInfo pipeline and applies the correct MftReference mask.
func (v *Volume) childrenMatching(parent *fileNode, predicate func(string) bool) ([]*fileNode, error) {
	ntfsCtx, err := v.ntfsContext()
	if err != nil {
		return nil, err
	}
	var result []*fileNode
	for _, idx := range parent.mftEntry.Dir(ntfsCtx) {
		name := idx.File().Name()
		if name == "." || name == ".." {
			continue
		}
		// Dir returns both DOS and Win32 entries for each child; We will skip the DOS entries
		// since they will create duplicate inodes in the results
		if idx.File().NameType().Name == "DOS" {
			continue
		}
		if !predicate(name) {
			continue
		}
		mftEntry, err := ntfsCtx.GetMFT(int64(idx.MftReference() & 0xFFFFFFFFFFFF))
		if err != nil {
			return nil, fmt.Errorf("GetMFT for %q: %w", name, err)
		}
		node, err := NewFileNode(v, mftEntry, name, parent)
		if err != nil {
			return nil, fmt.Errorf("NewFileInfo for %q: %w", name, err)
		}
		result = append(result, node)
	}
	return result, nil
}

// lookupChild finds the single direct child of parent with the given name (case-insensitive).
// It short-circuits on the first match rather than exhausting the full directory index.
func (v *Volume) lookupChild(parent *fileNode, name string) (*fileNode, error) {
	ntfsCtx, err := v.ntfsContext()
	if err != nil {
		return nil, err
	}
	mftEntry, err := parent.mftEntry.Open(ntfsCtx, name)
	if err != nil {
		return nil, fmt.Errorf("Open %q: %w", name, err)
	}
	node, err := NewFileNode(v, mftEntry, name, parent)
	if err != nil {
		return nil, fmt.Errorf("NewFileInfo for %q: %w", name, err)
	}
	return node, nil
}

func (v *Volume) Root() (*parser.MFT_ENTRY, error) {
	ntfsCtx, err := v.ntfsContext()
	if err != nil {
		return nil, err
	}
	return ntfsCtx.GetMFT(5)
}

func (v *Volume) findAndResolveInode(inode int64, depth int) (*fileNode, error) {
	if depth > 64 {
		return nil, fmt.Errorf("exceeded max depth resolving parent chain at inode %d", inode)
	}

	ntfsCtx, err := v.ntfsContext()
	if err != nil {
		return nil, err
	}
	entry, err := ntfsCtx.GetMFT(inode)
	if err != nil {
		return nil, fmt.Errorf("GetMFT for inode %d: %w", inode, err)
	}
	parentInode, err := parentInode(v, entry)
	if err != nil {
		return nil, fmt.Errorf("failed to get parent inode for inode %d: %w", inode, err)
	}

	fn := preferredFileName(entry.FileName(ntfsCtx))
	name := ""
	if fn != nil {
		name = fn.Name()
	}

	if parentInode == 5 {
		return NewFileNode(v, entry, name, nil)
	}

	parent, err := v.findAndResolveInode(parentInode, depth+1)
	if err != nil {
		return nil, err
	}
	return NewFileNode(v, entry, name, parent)
}

func (v *Volume) FindByInode(inode int64) (*fileNode, error) {
	return v.findAndResolveInode(inode, 0)
}

func (v *Volume) FindByPath(fullPath string, parent *fileNode) (*fileNode, error) {
	explodedPath, err := v.explodePath(fullPath, true)
	if err != nil {
		return nil, fmt.Errorf("failed to explode path: %w", err)
	}
	if len(explodedPath) == 0 {
		return nil, fmt.Errorf("path %s is invalid after splitting components", fullPath)
	}

	if parent == nil {
		root, err := v.Root()
		if err != nil {
			return nil, fmt.Errorf("failed to get root MFT entry: %w", err)
		}
		parent, err = NewFileNode(v, root, "", nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create root FileInfo: %w", err)
		}
	}

	cur := parent
	for _, name := range explodedPath {
		if name == "" {
			continue
		}
		cur, err = v.lookupChild(cur, name)
		if err != nil {
			return nil, fmt.Errorf("resolving %q in %s: %w", name, fullPath, err)
		}
	}
	return cur, nil
}

func (v *Volume) FindByDirectory(directory string, pattern string) ([]*fileNode, error) {
	dirNode, err := v.FindByPath(directory, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to find directory %s: %w", directory, err)
	}

	children, err := v.childrenMatching(dirNode, func(name string) bool {
		matched, _ := path.Match(pattern, name)
		return matched
	})
	return children, nil
}

func (v *Volume) IndexEntries(directory string, filename string) ([]*fileNode, error) {
	ntfsCtx, err := v.ntfsContext()
	if err != nil {
		return nil, err
	}

	node, err := v.FindByPath(directory, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to find node for path %s: %w", directory, err)
	}
	log := getLogger()

	dirEntries := node.mftEntry.DirNodes(ntfsCtx)

	var slacks []*parser.INDEX_RECORD_ENTRY
	var active []*parser.INDEX_RECORD_ENTRY
	for _, dirEntry := range dirEntries {
		slacks = append(slacks, dirEntry.ScanSlack(ntfsCtx)...)
		active = append(active, dirEntry.GetRecords(ntfsCtx)...)
	}

	log.Infof("Found %d index record entries in slack for directory %s", len(slacks), directory)
	log.Infof("Found %d active index record entries for directory %s", len(active), directory)

	notepad_slacks := 0
	for _, slack := range slacks {
		if !strings.EqualFold(slack.File().Name(), filename) {
			continue
		}
		// if !slack.IsValid() {
		// 	log.Errorf("Skipping invalid index record entry for %s: %s", filename, slack.DebugString())
		// 	continue
		// }
		// log.Infof("Found matching slack index record entry for %s: %s", filename, slack.DebugString())
		log.Infof("SlackEntryOffset: %d, SlackEntrySize: %s", slack.Offset, slack.DebugString())
		notepad_slacks++
	}

	notepad_actives := 0
	for _, active := range active {
		if !strings.EqualFold(active.File().Name(), filename) {
			continue
		}
		// if !active.IsValid() {
		// 	log.Errorf("Skipping invalid index record entry for %s: %s", filename, active.DebugString())
		// 	continue
		// }
		// log.Infof("Found matching active index record entry for %s: %s", filename, active.DebugString())
		notepad_actives++
	}
	log.Infof("Total matching slack index record entries for %s: %d", filename, notepad_slacks)
	log.Infof("Total matching active index record entries for %s: %d", filename, notepad_actives)

	// fileInfos := parser.ExtractI30List(v.ntfsContext(), node.mftEntry)
	// log.Infof("Extracted %d index entries for directory %s", len(fileInfos), directory)

	// for _, fileInfo := range fileInfos {
	// 	if !strings.EqualFold(fileInfo.Name, filename) {
	// 		continue
	// 	}
	// 	marshaled, err := json.MarshalIndent(fileInfo, "", "  ")
	// 	if err != nil {
	// 		log.Errorf("Failed to marshal fileInfo for %s: %v", filename, err)
	// 		continue
	// 	}
	// 	log.Infof("Found matching index entry for %s: %s", filename, string(marshaled))
	// }

	// for _, idx := range node.mftEntry.Dir(v.ntfsContext()) {
	// 	if !strings.EqualFold(idx.File().Name(), filename) {
	// 		continue
	// 	}
	// 	mftEntry, err := v.ntfsContext().GetMFT(int64(idx.MftReference() & 0xFFFFFFFFFFFF))
	// 	if err != nil {
	// 		return nil, fmt.Errorf("GetMFT for %q: %w", filename, err)
	// 	}
	// 	fileInfos := parser.ExtractI30List(v.ntfsContext(), mftEntry)
	// 	log.Infof("Extracted %d index entries for %s in directory %s", len(fileInfos), filename, directory)
	// }
	return nil, nil
}

func determineDriveLetter(queryContext table.QueryContext) (string, error) {
	driveFilters := filters.GetColumnConstraints(queryContext, "drive", table.OperatorEquals)
	if len(driveFilters) > 0 {
		if len(driveFilters) > 1 {
			return "", fmt.Errorf("Multiple drive constraints found, only one is supported: %s", driveFilters[0].Expression)
		}
		return driveFilters[0].Expression, nil
	}

	getDriveLetterFromPath := func(path string) (string, error) {
		if len(path) < 2 || path[1] != ':' {
			return "", fmt.Errorf("invalid path %s, expected format like C:\\path\\to\\file", path)
		}
		driveLetter := strings.ToUpper(string(path[0]))
		if driveLetter < "A" || driveLetter > "Z" {
			return "", fmt.Errorf("invalid drive letter %s in path %s", driveLetter, path)
		}
		return driveLetter, nil
	}

	pathFilters := filters.GetColumnConstraints(queryContext, "path", table.OperatorEquals)
	if len(pathFilters) > 0 {
		if len(pathFilters) > 1 {
			return "", fmt.Errorf("Multiple path constraints found, only one is supported: %s", pathFilters[0].Expression)
		}
		return getDriveLetterFromPath(pathFilters[0].Expression)
	}

	directoryFilters := filters.GetColumnConstraints(queryContext, "directory", table.OperatorEquals)
	if len(directoryFilters) > 0 {
		if len(directoryFilters) > 1 {
			return "", fmt.Errorf("Multiple directory constraints found, only one is supported: %s", directoryFilters[0].Expression)
		}
		return getDriveLetterFromPath(directoryFilters[0].Expression)
	}
	return "", fmt.Errorf("no drive, path, or directory constraints found")
}

func fileGenerateFunc(_ context.Context, queryContext table.QueryContext, log *logger.Logger, _ *client.ResilientClient) ([]elasticntfsfile.Result, error) {
	
	directoryConstraints := filters.GetColumnConstraints(queryContext, "directory", table.OperatorEquals)
	pathConstraints := filters.GetColumnConstraints(queryContext, "path", table.OperatorEquals)
	inodeConstraints := filters.GetColumnConstraints(queryContext, "inode", table.OperatorEquals)
	filenameConstraints := filters.GetColumnConstraints(queryContext, "filename", table.OperatorGlob)
	
	// Check for conflicting constraints
	if len(directoryConstraints) > 0 && (len(pathConstraints) > 0 || len(inodeConstraints) > 0) {
		return nil, fmt.Errorf("directory constraint cannot be combined with path or inode constraints")
	}
	if len(directoryConstraints) > 1 {
		return nil, fmt.Errorf("Multiple directory constraints found, only one is supported: %s", directoryConstraints[0].Expression)
	}
	if len(directoryConstraints) > 0 && len(filenameConstraints) == 0 {
		return nil, fmt.Errorf("directory constraint requires a filename glob constraint")
	}
	if len(directoryConstraints) > 0 && len(filenameConstraints) > 1 {
		return nil, fmt.Errorf("Multiple filename constraints found, only one is supported: %s", filenameConstraints[0].Expression)
	}

	if len(pathConstraints) > 0 && len(inodeConstraints) > 0 {
		return nil, fmt.Errorf("path and inode constraints cannot be combined")
	}

	// Check for multiple constraints
	if len(pathConstraints) > 1 {
		return nil, fmt.Errorf("Multiple path constraints found, only one is supported: %s", pathConstraints[0].Expression)
	}

	// Check for multiple constraints
	if len(inodeConstraints) > 1 {
		return nil, fmt.Errorf("Multiple inode constraints found, only one is supported: %s", inodeConstraints[0].Expression)
	}

	// Determine the drive letter from the query constraints
	driveLetter, err := determineDriveLetter(queryContext)
	if err != nil {
		return nil, fmt.Errorf("failed to determine drive letter from query constraints: %w", err)
	}

	// Open the volume
	vol, err := newVolume(driveLetter)
	if err != nil {
		return nil, fmt.Errorf("failed to open volume for drive %s: %w", driveLetter, err)
	}
	defer vol.Close()

	// Handle inode constraint
	if len(inodeConstraints) > 0 {
		inodeStr := inodeConstraints[0].Expression
		log.Infof("Query has inode constraint: %s", inodeStr)
		inode, err := strconv.Atoi(inodeStr)
		if err != nil {
			return nil, fmt.Errorf("failed to convert inode %s to integer: %w", inodeStr, err)
		}
		result, err := vol.FindByInode(int64(inode))
		if err != nil {
			return nil, fmt.Errorf("failed to find by inode %d: %w", inode, err)
		}
		materialized, err := result.Materialize()
		if err != nil {
			return nil, fmt.Errorf("failed to materialize result for inode %d: %w", inode, err)
		}
		return []elasticntfsfile.Result{*materialized}, nil
	}

	// Handle path constraint
	if len(pathConstraints) > 0 {
		path := pathConstraints[0].Expression
		log.Infof("Query has path constraint: %s", path)
		result, err := vol.FindByPath(path, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to find by path %s: %w", path, err)
		}
		materialized, err := result.Materialize()
		if err != nil {
			return nil, fmt.Errorf("failed to materialize result for path %s: %w", path, err)
		}
		return []elasticntfsfile.Result{*materialized}, nil
	}

	// Handle directory constraint	
	if len(directoryConstraints) > 0 {
		directoryFilters := filters.GetColumnConstraints(queryContext, "directory", table.OperatorEquals)
		if len(directoryFilters) != 1 {
			return nil, fmt.Errorf("Multiple directory constraints found, only one is supported: %s", directoryFilters[0].Expression)
		}
		filenameFilters := filters.GetColumnConstraints(queryContext, "filename", table.OperatorGlob)
		if len(filenameFilters) != 1 {
			return nil, fmt.Errorf("Directory constraint requires a filename glob constraint")
		}
		directory := directoryFilters[0].Expression
		pattern := filenameFilters[0].Expression
		log.Infof("Performing directory search with filename pattern: %s", pattern)
		nodes, err := vol.FindByDirectory(directory, pattern)
		if err != nil {
			return nil, fmt.Errorf("failed to perform scoped search for directory %s: %w", directory, err)
		}

		var results []elasticntfsfile.Result
		for _, node := range nodes {
			record, err := node.Materialize()
			if err != nil {
				log.Errorf("Failed to materialize file record for node %v: %v", node, err)
				continue
			}
			results = append(results, *record)
		}
		return results, nil
	}
	return nil, fmt.Errorf("unsupported query, must have either directory constraint with filename glob, or path constraint, or inode constraint")
}

func init() {
	elasticntfsfile.RegisterGenerateFunc(fileGenerateFunc)
}
