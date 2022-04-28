// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package localremote

import (
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/download"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/download/composed"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/download/fs"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/download/http"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/download/snapshot"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"
)

// NewDownloader creates a downloader which first checks local directory
// and then fallbacks to remote if configured.
func NewDownloader(log *logger.Logger, config *artifact.Config) (download.Downloader, error) {
	downloaders := make([]download.Downloader, 0, 3)
	downloaders = append(downloaders, fs.NewDownloader(config))

	// try snapshot repo before official
	if release.Snapshot() {
		snapDownloader, err := snapshot.NewDownloader(log, config, "")
		if err != nil {
			log.Error(err)
		} else {
			downloaders = append(downloaders, snapDownloader)
		}
	}

	httpDownloader, err := http.NewDownloader(log, config)
	if err != nil {
		return nil, err
	}

	downloaders = append(downloaders, httpDownloader)
	return composed.NewDownloader(downloaders...), nil
}
