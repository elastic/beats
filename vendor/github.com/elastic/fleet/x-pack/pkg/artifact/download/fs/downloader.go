// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/elastic/fleet/x-pack/pkg/artifact"
)

const (
	packagePermissions = 0660
	beatsSubfolder     = "beats"
)

// Downloader is a downloader able to fetch artifacts from elastic.co web page.
type Downloader struct {
	config *artifact.Config
}

// NewDownloader creates and configures Elastic Downloader
func NewDownloader(config *artifact.Config) *Downloader {
	return &Downloader{
		config: config,
	}
}

// Download fetches the package from configured source.
// Returns absolute path to downloaded package and an error.
func (e *Downloader) Download(programName, version string) (string, error) {
	// create a destination directory root/program
	destinationDir := filepath.Join(e.config.TargetDirectory, programName)
	if err := os.MkdirAll(destinationDir, os.ModeDir); err != nil {
		return "", errors.Wrap(err, "creating directory for downloaded artifact failed")
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
		return "", errors.Wrap(err, "generating package name failed")
	}

	fullPath, err := artifact.GetArtifactPath(programName, version, operatingSystem, e.config.Arch(), e.config.TargetDirectory)
	if err != nil {
		return "", errors.Wrap(err, "generating package path failed")
	}

	sourcePath := filepath.Join(e.getDropPath(), filename)
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return "", errors.Wrapf(err, "package '%s' not found", sourcePath)
	}
	defer sourceFile.Close()

	destinationFile, err := os.OpenFile(fullPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, packagePermissions)
	if err != nil {
		return "", errors.Wrap(err, "creating package file failed")
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	return fullPath, nil
}

func (e *Downloader) getDropPath() string {
	fmt.Println("drop path?")
	fmt.Println(e.config.DropPath)
	// if drop path is not provided fallback to beats subfolder
	if e.config.DropPath == "" {
		return beatsSubfolder
	}

	// if droppath does not exist fallback to beats subfolder
	stat, err := os.Stat(e.config.DropPath)
	if err != nil || !stat.IsDir() {
		return beatsSubfolder
	}

	return e.config.DropPath
}
