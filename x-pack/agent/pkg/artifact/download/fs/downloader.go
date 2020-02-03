// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fs

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/elastic/beats/x-pack/agent/pkg/agent/errors"
	"github.com/elastic/beats/x-pack/agent/pkg/artifact"
)

const (
	packagePermissions = 0660
	beatsSubfolder     = "beats"
)

// Downloader is a downloader able to fetch artifacts from elastic.co web page.
type Downloader struct {
	dropPath string
	config   *artifact.Config
}

// NewDownloader creates and configures Elastic Downloader
func NewDownloader(config *artifact.Config) *Downloader {
	return &Downloader{
		config:   config,
		dropPath: getDropPath(config),
	}
}

// Download fetches the package from configured source.
// Returns absolute path to downloaded package and an error.
func (e *Downloader) Download(_ context.Context, programName, version string) (string, error) {
	// create a destination directory root/program
	destinationDir := filepath.Join(e.config.TargetDirectory, programName)
	if err := os.MkdirAll(destinationDir, os.ModeDir); err != nil {
		return "", errors.New(err, "creating directory for downloaded artifact failed", errors.TypeFilesystem, errors.M(errors.MetaKeyPath, destinationDir))
	}

	// download from source to dest
	path, err := e.download(e.config.OS(), programName, version)
	if err != nil {
		os.Remove(path)
	}

	return path, err
}

func (e *Downloader) download(operatingSystem, programName, version string) (string, error) {
	filename, err := artifact.GetArtifactName(programName, version, operatingSystem, e.config.Arch())
	if err != nil {
		return "", errors.New(err, "generating package name failed")
	}

	fullPath, err := artifact.GetArtifactPath(programName, version, operatingSystem, e.config.Arch(), e.config.TargetDirectory)
	if err != nil {
		return "", errors.New(err, "generating package path failed")
	}

	sourcePath := filepath.Join(e.dropPath, filename)
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return "", errors.New(err, fmt.Sprintf("package '%s' not found", sourcePath), errors.TypeFilesystem, errors.M(errors.MetaKeyPath, fullPath))
	}
	defer sourceFile.Close()

	destinationFile, err := os.OpenFile(fullPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, packagePermissions)
	if err != nil {
		return "", errors.New(err, "creating package file failed", errors.TypeFilesystem, errors.M(errors.MetaKeyPath, fullPath))
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	return fullPath, nil
}

func getDropPath(cfg *artifact.Config) string {
	// if drop path is not provided fallback to beats subfolder
	if cfg == nil || cfg.DropPath == "" {
		return beatsSubfolder
	}

	// if droppath does not exist fallback to beats subfolder
	stat, err := os.Stat(cfg.DropPath)
	if err != nil || !stat.IsDir() {
		return beatsSubfolder
	}

	return cfg.DropPath
}
