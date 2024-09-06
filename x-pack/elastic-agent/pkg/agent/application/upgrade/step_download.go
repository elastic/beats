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
	downloader "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/download/localremote"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/download/snapshot"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"
)

const (
	defaultUpgradeFallbackPGP = "https://artifacts.elastic.co/GPG-KEY-elastic-agent"
)

func (u *Upgrader) downloadArtifact(ctx context.Context, version, sourceURI string, skipVerifyOverride bool, pgpBytes ...string) (string, error) {
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

	pgpBytes = appendFallbackPGP(pgpBytes)

	fetcher, err := newDownloader(version, u.log, &settings)
	if err != nil {
		return "", errors.New(err, "initiating fetcher")
	}

	path, err := fetcher.Download(ctx, agentSpec, version)
	if err != nil {
		return "", errors.New(err, "failed upgrade of agent binary")
	}

	if skipVerifyOverride {
		return path, nil
	}

	verifier, err := newVerifier(version, u.log, &settings)
	if err != nil {
		return "", errors.New(err, "initiating verifier")
	}

	matches, err := verifier.Verify(agentSpec, version, true, pgpBytes...)
	if err != nil {
		return "", errors.New(err, "failed verification of agent binary")
	}
	if !matches {
		return "", errors.New("failed verification of agent binary, hash does not match", errors.TypeSecurity)
	}

	return path, nil
}

func appendFallbackPGP(pgpBytes []string) []string {
	if pgpBytes == nil {
		pgpBytes = make([]string, 0, 1)
	}

	fallbackPGP := download.PgpSourceURIPrefix + defaultUpgradeFallbackPGP
	pgpBytes = append(pgpBytes, fallbackPGP)
	return pgpBytes
}

func newDownloader(version string, log *logger.Logger, settings *artifact.Config) (download.Downloader, error) {
	if !strings.HasSuffix(version, "-SNAPSHOT") {
		return downloader.NewDownloader(log, settings)
	}

	// We can use up to 3 downloaders
	downloaders := make([]download.Downloader, 0, 3)

	downloaders = append(downloaders, fs.NewDownloader(settings))

	// try snapshot repo before official
	snapDownloader, err := snapshot.NewDownloader(log, settings, version)
	if err != nil {
		log.Error("Error initializing snapshot downloader: %s", err)
		//return nil, err
	} else {
		downloaders = append(downloaders, snapDownloader)
	}

	httpDownloader, err := http.NewDownloader(log, settings)
	if err != nil {
		log.Error("Error initializing http downloader: %s", err)
	} else {
		downloaders = append(downloaders, httpDownloader)
	}

	return composed.NewDownloader(log, downloaders...), nil
}

func newVerifier(version string, log *logger.Logger, settings *artifact.Config) (download.Verifier, error) {
	allowEmptyPgp, pgp := release.PGP()
	if !strings.HasSuffix(version, "-SNAPSHOT") {
		return downloader.NewVerifier(log, settings, allowEmptyPgp, pgp)
	}

	fsVerifier, err := fs.NewVerifier(settings, allowEmptyPgp, pgp)
	if err != nil {
		return nil, err
	}

	snapshotVerifier, err := snapshot.NewVerifier(settings, allowEmptyPgp, pgp, version)
	if err != nil {
		return nil, err
	}

	remoteVerifier, err := http.NewVerifier(settings, allowEmptyPgp, pgp)
	if err != nil {
		return nil, err
	}

	return composed.NewVerifier(fsVerifier, snapshotVerifier, remoteVerifier), nil
}
