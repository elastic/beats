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

type downloaderFactory func(string, *logger.Logger, *artifact.Config) (download.Downloader, error)

func (u *Upgrader) downloadArtifact(ctx context.Context, version, sourceURI string, skipVerifyOverride bool, pgpBytes ...string) (string, error) {
	// do not update source config
	settings := *u.settings

	var factory downloaderFactory
	var verifier download.Verifier
	var err error
	if sourceURI != "" {
		if strings.HasPrefix(sourceURI, "file://") {
			// update the DropPath so the fs.Downloader can download from this
			// path instead of looking into the installed downloads directory
			settings.DropPath = strings.TrimPrefix(sourceURI, "file://")

			// set specific downloader, local file just uses the fs.NewDownloader
			// no fallback is allowed because it was requested that this specific source be used
			factory = func(version string, l *logger.Logger, config *artifact.Config) (download.Downloader, error) {
				return fs.NewDownloader(config), nil
			}

			// set specific verifier, local file verifies locally only
			allowEmptyPgp, pgp := release.PGP()
			verifier, err = fs.NewVerifier(&settings, allowEmptyPgp, pgp)
			if err != nil {
				return "", errors.New(err, "initiating verifier")
			}
			// log that a local upgrade artifact is being used
			u.log.Infow("Using local upgrade artifact", "version", version,
				"drop_path", settings.DropPath,
				"target_path", settings.TargetDirectory, "install_path", settings.InstallPath)
		} else {
			settings.SourceURI = sourceURI
		}
	}

	if factory == nil {
		// set the factory to the newDownloader factory
		factory = newDownloader
		u.log.Infow("Downloading upgrade artifact", "version", version,
			"source_uri", settings.SourceURI, "drop_path", settings.DropPath,
			"target_path", settings.TargetDirectory, "install_path", settings.InstallPath)
	}

	pgpBytes = appendFallbackPGP(pgpBytes)

	fetcher, err := factory(version, u.log, &settings)
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

	if verifier == nil {
		verifier, err = newVerifier(version, u.log, &settings)
		if err != nil {
			return "", errors.New(err, "initiating verifier")
		}
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

	// try snapshot repo before official
	snapDownloader, err := snapshot.NewDownloader(log, settings, version)
	if err != nil {
		return nil, err
	}

	httpDownloader, err := http.NewDownloader(log, settings)
	if err != nil {
		return nil, err
	}

	return composed.NewDownloader(fs.NewDownloader(settings), snapDownloader, httpDownloader), nil
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
