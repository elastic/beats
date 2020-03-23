// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package localremote

import (
	"github.com/elastic/beats/v7/x-pack/agent/pkg/artifact"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/artifact/download"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/artifact/download/composed"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/artifact/download/fs"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/artifact/download/http"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/artifact/download/snapshot"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/release"
)

// NewDownloader creates a downloader which first checks local directory
// and then fallbacks to remote if configured.
func NewDownloader(config *artifact.Config, downloaders ...download.Downloader) download.Downloader {
	d := composed.NewDownloader(fs.NewDownloader(config), http.NewDownloader(config))
	if release.Snapshot() {
		return snapshot.NewDownloader(d)
	}

	return d
}
