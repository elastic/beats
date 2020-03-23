// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package snapshot

import (
	"context"
	"fmt"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/artifact/download"
)

// Downloader has an embedded downloader and tweaks behavior in a way to use
// handle SNAPSHOTs only.
type Downloader struct {
	embeddedDownloader download.Downloader
}

// NewDownloader creates a snapshot downloader out of predefined downloader.
func NewDownloader(downloader download.Downloader) *Downloader {
	return &Downloader{
		embeddedDownloader: downloader,
	}
}

// Download fetches the package from configured source.
// Returns absolute path to downloaded package and an error.
func (e *Downloader) Download(ctx context.Context, programName, version string) (string, error) {
	version = fmt.Sprintf("%s-SNAPSHOT", version)
	return e.embeddedDownloader.Download(ctx, programName, version)
}
