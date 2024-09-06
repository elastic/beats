// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package composed

import (
	"context"
	goerrors "errors"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/download"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

// Downloader is a downloader with a predefined set of downloaders.
// During each download call it tries to call the first one and on failure fallbacks to
// the next one.
// Error is returned if all of them fail.
type Downloader struct {
	log *logger.Logger
	dd  []download.Downloader
}

// NewDownloader creates a downloader out of predefined set of downloaders.
// During each download call it tries to call the first one and on failure fallbacks to
// the next one.
// Error is returned if all of them fail.
func NewDownloader(log *logger.Logger, downloaders ...download.Downloader) *Downloader {
	return &Downloader{
		log: log,
		dd:  downloaders,
	}
}

// Download fetches the package from configured source.
// Returns absolute path to downloaded package and an error.
func (e *Downloader) Download(ctx context.Context, spec program.Spec, version string) (string, error) {
	var err error
	for _, d := range e.dd {
		e.log.Debugf("attempting download using downloader %T", d)
		s, downloadErr := d.Download(ctx, spec, version)
		if downloadErr == nil {
			return s, nil
		}
		e.log.Debugf("error using downloader %T: %s", d, downloadErr)
		err = goerrors.Join(err, downloadErr)
	}

	return "", err
}

// Reload reloads config
func (e *Downloader) Reload(c *artifact.Config) error {
	for _, d := range e.dd {
		reloadable, ok := d.(download.Reloader)
		if !ok {
			continue
		}

		if err := reloadable.Reload(c); err != nil {
			return errors.New(err, "failed reloading artifact config for composed downloader")
		}
	}
	return nil
}
