// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package registry

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/forensicanalysis/fslib"
	"github.com/forensicanalysis/fslib/systemfs"
	"www.velocidex.com/golang/regparser"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

// ensureLogger ensures that a logger is provided. If no logger is provided,
// a new logger is created and returned. A simple check to ensure
// that the logger is not nil.
func ensureLogger(log *logger.Logger) *logger.Logger {
	if log == nil {
		log = logger.New(os.Stderr, false)
	}
	return log
}

// getFileContents reads the contents of a file and returns it as a byte slice.
// If the file is not readable, it will attempt to read it using a low level read
// using fslib.
func getFileContents(filePath string, log *logger.Logger) ([]byte, error) {
	log = ensureLogger(log)
	content, err := os.ReadFile(filePath)
	if err == nil {
		return content, nil
	}
	log.Infof("failed to read %s, falling back to low level read", filePath)

	// fallback to a low level read using fslib
	sourceFS, err := systemfs.New()
	if err != nil {
		log.Errorf("failed to open file %s: %s", filePath, err.Error())
		return nil, err
	}
	fsPath, err := fslib.ToFSPath(filePath)
	if err != nil {
		return nil, err
	}
	return fs.ReadFile(sourceFS, fsPath)
}

// loadExistingRegistry loads the registry from the given file path.  Without any transaction logs.
func loadExistingRegistry(filePath string, log *logger.Logger) (*regparser.Registry, error) {
	log = ensureLogger(log)
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
	log = ensureLogger(log)

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
	log = ensureLogger(log)

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

		// close and remove the temporary file
		defer transactionLogHandle.Close()
		defer os.Remove(transactionLogHandle.Name()) // Clean up temp file
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
	log = ensureLogger(log)
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
	log = ensureLogger(log)

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
	} else {
		log.Infof("transaction logs found, recovering registry")
		registry, err := recoverRegistry(filePath, transactionLogs, log)
		if err != nil {
			log.Errorf("failed to recover registry: %v", err)
			// if recovery fails, try to fall back to the existing registry
			registry, err = loadExistingRegistry(filePath, log)
			if err != nil {
				log.Errorf("failed to load existing registry: %v", err)
				return nil, false, err
			}
			return registry, false, nil
		} else {
			// if recovery succeeds, return the recovered registry
			return registry, true, nil
		}
	}
}
