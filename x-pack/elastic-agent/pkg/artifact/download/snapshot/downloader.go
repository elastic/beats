// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package snapshot

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"

	"github.com/elastic/beats/v7/libbeat/common/transport/httpcommon"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/download"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/download/http"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"
)

// Downloader is responsible for downloading artifacts
type Downloader struct {
	downloader      download.Downloader
	versionOverride string
}

// NewDownloader creates a downloader which first checks local directory
// and then fallbacks to remote if configured.
func NewDownloader(log *logger.Logger, config *artifact.Config, versionOverride string) (download.Downloader, error) {
	cfg, err := snapshotConfig(config, versionOverride)
	if err != nil {
		return nil, err
	}

	httpDownloader, err := http.NewDownloader(log, cfg)
	if err != nil {
		return nil, errors.New(err, "failed to create snapshot downloader")
	}

	return &Downloader{
		downloader:      httpDownloader,
		versionOverride: versionOverride,
	}, nil
}

// Download fetches the package from configured source.
// Returns absolute path to downloaded package and an error.
func (e *Downloader) Download(ctx context.Context, spec program.Spec, version string) (string, error) {
	return e.downloader.Download(ctx, spec, version)
}

// Reload reloads config
func (e *Downloader) Reload(c *artifact.Config) error {
	reloader, ok := e.downloader.(artifact.ConfigReloader)
	if !ok {
		return nil
	}

	cfg, err := snapshotConfig(c, e.versionOverride)
	if err != nil {
		return errors.New(err, "snapshot.downloader: failed to generate snapshot config")
	}

	return reloader.Reload(cfg)
}

func snapshotConfig(config *artifact.Config, versionOverride string) (*artifact.Config, error) {
	snapshotURI, err := snapshotURI(versionOverride, config)
	if err != nil {
		return nil, fmt.Errorf("failed to detect remote snapshot repo, proceeding with configured: %w", err)
	}

	return &artifact.Config{
		OperatingSystem:       config.OperatingSystem,
		Architecture:          config.Architecture,
		SourceURI:             snapshotURI,
		TargetDirectory:       config.TargetDirectory,
		InstallPath:           config.InstallPath,
		DropPath:              config.DropPath,
		HTTPTransportSettings: config.HTTPTransportSettings,
	}, nil
}

func snapshotURI(versionOverride string, config *artifact.Config) (string, error) {
	version := release.Version()
	if versionOverride != "" {
		versionOverride = strings.TrimSuffix(versionOverride, "-SNAPSHOT")
		version = versionOverride
	}

	client, err := config.HTTPTransportSettings.Client(httpcommon.WithAPMHTTPInstrumentation())
	if err != nil {
		return "", err
	}

	artifactsURI := fmt.Sprintf("https://artifacts-api.elastic.co/v1/search/%s-SNAPSHOT/elastic-agent", version)
	resp, err := client.Get(artifactsURI)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body := struct {
		Packages map[string]interface{} `json:"packages"`
	}{}

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&body); err != nil {
		return "", err
	}

	if len(body.Packages) == 0 {
		return "", fmt.Errorf("no packages found in snapshot repo")
	}

	for k, pkg := range body.Packages {
		pkgMap, ok := pkg.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("content of '%s' is not a map", k)
		}

		uriVal, found := pkgMap["url"]
		if !found {
			return "", fmt.Errorf("item '%s' does not contain url", k)
		}

		uri, ok := uriVal.(string)
		if !ok {
			return "", fmt.Errorf("uri is not a string")
		}

		index := strings.Index(uri, "/beats/elastic-agent/")

		// Because we're iterating over a map from the API response,
		// the order is random and some elements there do not contain the
		// `/beats/elastic-agent/` substring, so we need to go through the
		// whole map before returning an error.
		//
		// One of the elements that might be there and do not contain this
		// substring is the `elastic-agent-shipper`, whose URL is something like:
		// https://snapshots.elastic.co/8.7.0-d050210c/downloads/elastic-agent-shipper/elastic-agent-shipper-8.7.0-SNAPSHOT-linux-x86_64.tar.gz
		if index != -1 {
			return uri[:index], nil
		}
	}

	return "", fmt.Errorf("uri not detected")
}
