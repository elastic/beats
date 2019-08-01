// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package localremote

import (
	"github.com/elastic/fleet/x-pack/pkg/artifact"
	"github.com/elastic/fleet/x-pack/pkg/artifact/download"
	"github.com/elastic/fleet/x-pack/pkg/artifact/download/composed"
	"github.com/elastic/fleet/x-pack/pkg/artifact/download/fs"
	"github.com/elastic/fleet/x-pack/pkg/artifact/download/http"
)

// NewVerifier creates a downloader which first checks local directory
// and then fallbacks to remote if configured.
func NewVerifier(config *artifact.Config, downloaders ...download.Downloader) (download.Verifier, error) {
	fsVer, err := fs.NewVerifier(config)
	if err != nil {
		return nil, err
	}
	remoteVer, err := http.NewVerifier(config)
	if err != nil {
		return nil, err
	}

	return composed.NewVerifier(fsVer, remoteVer), nil
}
