// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package registry

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"www.velocidex.com/golang/regparser"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
    "www.velocidex.com/golang/go-ntfs/parser"
)

func getFileContents(filePath string, log *logger.Logger) ([]byte, error) {
	content, err := os.ReadFile(filePath)
	if err == nil {
		return content, nil
	}
	log.Infof("failed to read %s, falling back to low level read", filePath)
	return readFileViaNTFS(filePath)
}

// This function was written with help from Claude Code, and is based on the code
// found in the fslib library for doing low level NTFS reads.  fslib kept us pinned
// to an older version of go-ntfs, but this functionality was all we needed from that library,
// which already used go-ntfs under the hood.  By implementing it ourselves we were able 
// to update to the latest version of go-ntfs
func readFileViaNTFS(filePath string) ([]byte, error) {
	if len(filePath) < 3 || filePath[1] != ':' {
		return nil, fmt.Errorf("unsupported path format: %s", filePath)
	}

	driveLetter := filePath[0]
	ntfsPath := "/" + filepath.ToSlash(filePath[3:]) // C:\Windows\foo.txt → /Windows/foo.txt

	volume, err := os.Open(fmt.Sprintf(`\\.\%c:`, driveLetter))
	if err != nil {
		return nil, fmt.Errorf("failed to open volume: %w", err)
	}
	defer volume.Close()

	reader, err := parser.NewPagedReader(volume, 1024*1024, 100*1024*1024)
	if err != nil {
		return nil, fmt.Errorf("failed to create paged reader: %w", err)
	}

	ntfsCtx, err := parser.GetNTFSContext(reader, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to parse NTFS: %w", err)
	}

	root, err := ntfsCtx.GetMFT(5)
	if err != nil {
		return nil, fmt.Errorf("failed to get MFT root: %w", err)
	}

	entry, err := root.Open(ntfsCtx, ntfsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s via NTFS: %w", ntfsPath, err)
	}

	attr, err := entry.GetAttribute(ntfsCtx, 128, -1, "") // 128 = $DATA
	if err != nil {
		return nil, fmt.Errorf("failed to get data attribute: %w", err)
	}

	infos, err := parser.ModelMFTEntry(ntfsCtx, entry)
	if err != nil {
		return nil, fmt.Errorf("failed to get file size: %w", err)
	}

	data := make([]byte, infos.Size)
	_, err = attr.Data(ntfsCtx).ReadAt(data, 0)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read file data: %w", err)
	}
	return data, nil
}

// loadExistingRegistry loads the registry from the given file path.  Without any transaction logs.
func loadExistingRegistry(filePath string, log *logger.Logger) (*regparser.Registry, error) {
	hiveContent, err := getFileContents(filePath, log)
	if err != nil {
		return nil, err
	}
	log.Infof("loaded registry from %s", filePath)
	return regparser.NewRegistry(bytes.NewReader(hiveContent))
}

// createTempFile creates a temporary file with the given contents and returns the file handle.
// caller is responsible for closing the file and removing the file after use. function returns
// a handle that has been seeked to the start of the file.
func createTempFile(contents []byte, log *logger.Logger) (*os.File, error) {
	// create a temporary file
	tempFile, err := os.CreateTemp("", "registry-*.hive")
	if err != nil {
		return nil, err
	}

	// write the contents to the temporary file
	if _, err := tempFile.Write(contents); err != nil {
		log.Errorf("failed to write to temp file: %s", err.Error())
		return nil, err
	}

	// sync the temporary file
	if err := tempFile.Sync(); err != nil {
		log.Errorf("failed to sync temp file: %s", err.Error())
		tempFile.Close()
		os.Remove(tempFile.Name())
		return nil, err
	}

	// seek to the start of the temporary file
	_, err = tempFile.Seek(0, io.SeekStart)
	if err != nil {
		log.Errorf("failed to seek to start of temp file: %s", err.Error())
		tempFile.Close()
		os.Remove(tempFile.Name())
		return nil, err
	}
	return tempFile, nil
}

// recoverRegistry recovers the registry from the given file path and transaction log paths.
func recoverRegistry(filePath string, transactionLogPaths []string, log *logger.Logger) (registry *regparser.Registry, err error) {
	// get the hive content
	hiveContent, err := getFileContents(filePath, log)
	if err != nil {
		return nil, err
	}

	// create a temporary file for the hive content
	tempHiveFile, err := createTempFile(hiveContent, log)
	if err != nil {
		return nil, err
	}
	defer os.Remove(tempHiveFile.Name()) // Clean up temp file
	defer tempHiveFile.Close()

	// get the transaction log handles
	transactionLogHandles := make([]*os.File, 0, len(transactionLogPaths))
	for _, transactionLogPath := range transactionLogPaths {
		transactionLogContent, err := getFileContents(transactionLogPath, log)
		if err != nil {
			log.Errorf("failed to get file contents for transaction log %s: %s", transactionLogPath, err.Error())
			return nil, err
		}
		transactionLogHandle, err := createTempFile(transactionLogContent, log)
		if err != nil {
			log.Errorf("failed to create temporary file for transaction log %s: %s", transactionLogPath, err.Error())
			return nil, err
		}

		// defer the removal and closing of the temporary file we just created.
		// this is done here because we have easy access to the file handle and name of the file,
		// whereas outside of this loop we don't have access to the file name.
		defer os.Remove(transactionLogHandle.Name())
		defer transactionLogHandle.Close()

		transactionLogHandles = append(transactionLogHandles, transactionLogHandle)
	}

	// attempt to recover the hive
	recoveredHive, err := regparser.RecoverHive(tempHiveFile, transactionLogHandles...)
	if err != nil {
		log.Errorf("failed to recover hive: %s", err.Error())
		return nil, err
	}

	// close an remove the recovered hive at the end of the function,
	// we are reading it back into memory later so we don't need to keep it open.
	defer recoveredHive.Close()
	defer os.Remove(recoveredHive.Name())

	// seek to the start of the recovered hive
	_, err = recoveredHive.Seek(0, io.SeekStart)
	if err != nil {
		log.Errorf("failed to seek to start of recovered hive: %s", err.Error())
		return nil, err
	}

	// read the recovered hive contents into memory
	recoveredHiveContents, err := io.ReadAll(recoveredHive)
	if err != nil {
		log.Errorf("failed to read recovered hive: %s", err.Error())
		return nil, err
	}

	// return the recovered hive as a registry
	return regparser.NewRegistry(bytes.NewReader(recoveredHiveContents))
}

// findTransactionLogs finds all transaction logs for the given hive file.
// Registry transaction logs are files that record changes to the Windows
// Registry to prevent corruption and to allow for recovery
func findTransactionLogs(filePath string, log *logger.Logger) []string {
	logFiles := make([]string, 0)

	// Get the directory and base name of the file
	dir := filepath.Dir(filePath)
	baseName := filepath.Base(filePath)

	// Read the directory
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Errorf("failed to read directory %s: %v", dir, err)
		return logFiles
	}

	// Look for files that start with the base name and end with .LOG*
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fileName := entry.Name()
		pattern := fmt.Sprintf("%s.LOG*", baseName)
		if match, err := filepath.Match(pattern, fileName); err == nil && match {
			logFiles = append(logFiles, filepath.Join(dir, fileName))
		}
	}

	return logFiles
}

// LoadRegistry loads the registry from the given file path.
// If transaction logs are found, it will attempt to recover the registry.
// If no transaction logs are found, it will load the existing registry.
// The function returns a registry object and an error.
// The registry object is the recovered registry if recovery was successful,
// otherwise it is the existing registry.
func LoadRegistry(filePath string, log *logger.Logger) (registry *regparser.Registry, recovered bool, err error) {
	// ensure a path was provided
	if filePath == "" {
		log.Errorf("hive file path is empty")
		return nil, false, fmt.Errorf("hive file path is empty")
	}

	transactionLogs := findTransactionLogs(filePath, log)
	if len(transactionLogs) == 0 {
		log.Infof("no transaction logs found, loading existing registry")
		registry, err = loadExistingRegistry(filePath, log)
		if err != nil {
			log.Errorf("failed to load existing registry: %v", err)
			return nil, false, err
		}
		return registry, false, nil
	}
	log.Infof("transaction logs found, recovering registry")
	registry, err = recoverRegistry(filePath, transactionLogs, log)
	if err != nil {
		log.Errorf("failed to recover registry: %v", err)
		return nil, false, err
	}
	return registry, true, nil
}
