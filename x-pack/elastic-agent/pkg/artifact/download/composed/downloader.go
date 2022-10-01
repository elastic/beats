// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package composed

import (
	"context"

	"github.com/hashicorp/go-multierror"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/download"
)

// Downloader is a downloader with a predefined set of downloaders.
// During each download call it tries to call the first one and on failure fallbacks to
// the next one.
// Error is returned if all of them fail.
type Downloader struct {
	dd []download.Downloader
}

// NewDownloader creates a downloader out of predefined set of downloaders.
// During each download call it tries to call the first one and on failure fallbacks to
// the next one.
// Error is returned if all of them fail.
func NewDownloader(downloaders ...download.Downloader) *Downloader {
	return &Downloader{
		dd: downloaders,
	}
}

// Download fetches the package from configured source.
// Returns absolute path to downloaded package and an error.
func (e *Downloader) Download(ctx context.Context, spec program.Spec, version string) (string, error) {
	var err error

	for _, d := range e.dd {
		s, e := d.Download(ctx, spec, version)
		if e == nil {
			return s, nil
		}

		err = multierror.Append(err, e)
	}

	return "", err
}
