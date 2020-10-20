// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package upgrade

import (
	"context"
	"strings"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	downloader "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/download/localremote"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"
)

func (u *Upgrader) downloadArtifact(ctx context.Context, version, sourceURI string) (string, error) {
	// do not update source config
	settings := *u.settings
	if sourceURI != "" {
		if strings.HasPrefix(sourceURI, "file://") {
			// update the DropPath so the fs.Downloader can download from this
			// path instead of looking into the installed downloads directory
			settings.DropPath = strings.TrimPrefix(sourceURI, "file://")
		} else {
			settings.SourceURI = sourceURI
		}
	}

	allowEmptyPgp, pgp := release.PGP()
	verifier, err := downloader.NewVerifier(u.log, &settings, allowEmptyPgp, pgp, true)
	if err != nil {
		return "", errors.New(err, "initiating verifier")
	}

	fetcher := downloader.NewDownloader(u.log, &settings, true)
	path, err := fetcher.Download(ctx, agentName, agentArtifactName, version)
	if err != nil {
		return "", errors.New(err, "failed upgrade of agent binary")
	}

	matches, err := verifier.Verify(agentName, version, agentArtifactName, true)
	if err != nil {
		return "", errors.New(err, "failed verification of agent binary")
	}
	if !matches {
		return "", errors.New("failed verification of agent binary, hash does not match", errors.TypeSecurity)
	}

	return path, nil
}
