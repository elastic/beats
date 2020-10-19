// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package upgrade

import (
	"context"
	"strings"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/download"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/download/composed"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/download/fs"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/download/http"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/download/snapshot"
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

	verifier, err := u.verifier(&settings)
	if err != nil {
		return "", errors.New(err, "initiating verifier")
	}

	fetcher := u.downloader(&settings)
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

// gets a downloader for local, official, snapshot in that order
func (u *Upgrader) downloader(settings *artifact.Config) download.Downloader {
	downloaders := make([]download.Downloader, 0, 3)
	downloaders = append(downloaders, fs.NewDownloader(settings), http.NewDownloader(settings))

	snapDownloader, err := snapshot.NewDownloader(settings)
	if err != nil {
		u.log.Error(err)
	} else {
		downloaders = append(downloaders, snapDownloader)
	}

	return composed.NewDownloader(downloaders...)
}

// gets a verifier for local, official, snapshot in that order
func (u *Upgrader) verifier(settings *artifact.Config) (download.Verifier, error) {
	allowEmptyPgp, pgp := release.PGP()
	verifiers := make([]download.Verifier, 0, 3)

	fsVer, err := fs.NewVerifier(settings, allowEmptyPgp, pgp)
	if err != nil {
		return nil, err
	}
	verifiers = append(verifiers, fsVer)

	remoteVer, err := http.NewVerifier(settings, allowEmptyPgp, pgp)
	if err != nil {
		return nil, err
	}
	verifiers = append(verifiers, remoteVer)

	snapshotVerifier, err := snapshot.NewVerifier(settings, allowEmptyPgp, pgp)
	if err != nil {
		u.log.Error(err)
	} else {
		verifiers = append(verifiers, snapshotVerifier)
	}

	return composed.NewVerifier(verifiers...), nil
}
