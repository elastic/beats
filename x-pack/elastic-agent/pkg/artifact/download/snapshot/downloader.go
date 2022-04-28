// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package snapshot

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"

	"github.com/elastic/beats/v7/libbeat/common/transport/httpcommon"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/download"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/download/http"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"
)

// NewDownloader creates a downloader which first checks local directory
// and then fallbacks to remote if configured.
func NewDownloader(log *logger.Logger, config *artifact.Config, versionOverride string) (download.Downloader, error) {
	cfg, err := snapshotConfig(config, versionOverride)
	if err != nil {
		return nil, err
	}
	return http.NewDownloader(log, cfg)
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
		if index == -1 {
			return "", fmt.Errorf("not an agent uri: '%s'", uri)
		}

		return uri[:index], nil
	}

	return "", fmt.Errorf("uri not detected")
}
