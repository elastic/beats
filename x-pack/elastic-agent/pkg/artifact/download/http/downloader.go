// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/elastic/beats/v7/libbeat/common/transport/httpcommon"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"
)

const (
	packagePermissions = 0660
)

var headers = map[string]string{
	"User-Agent": fmt.Sprintf("Beat elastic-agent v%s", release.Version()),
}

// Downloader is a downloader able to fetch artifacts from elastic.co web page.
type Downloader struct {
	config *artifact.Config
	client http.Client
}

// NewDownloader creates and configures Elastic Downloader
func NewDownloader(config *artifact.Config) (*Downloader, error) {
	client, err := config.HTTPTransportSettings.Client(
		httpcommon.WithAPMHTTPInstrumentation(),
	)
	if err != nil {
		return nil, err
	}

	client.Transport = withHeaders(client.Transport, headers)
	return NewDownloaderWithClient(config, *client), nil
}

// NewDownloaderWithClient creates Elastic Downloader with specific client used
func NewDownloaderWithClient(config *artifact.Config, client http.Client) *Downloader {
	return &Downloader{
		config: config,
		client: client,
	}
}

// Download fetches the package from configured source.
// Returns absolute path to downloaded package and an error.
func (e *Downloader) Download(ctx context.Context, spec program.Spec, version string) (_ string, err error) {
	downloadedFiles := make([]string, 0, 2)
	defer func() {
		if err != nil {
			for _, path := range downloadedFiles {
				os.Remove(path)
			}
		}
	}()

	// download from source to dest
	path, err := e.download(ctx, e.config.OS(), spec, version)
	downloadedFiles = append(downloadedFiles, path)
	if err != nil {
		return "", err
	}

	hashPath, err := e.downloadHash(ctx, e.config.OS(), spec, version)
	downloadedFiles = append(downloadedFiles, hashPath)
	return path, err
}

func (e *Downloader) composeURI(artifactName, packageName string) (string, error) {
	upstream := e.config.SourceURI
	if !strings.HasPrefix(upstream, "http") && !strings.HasPrefix(upstream, "file") && !strings.HasPrefix(upstream, "/") {
		// always default to https
		upstream = fmt.Sprintf("https://%s", upstream)
	}

	// example: https://artifacts.elastic.co/downloads/beats/filebeat/filebeat-7.1.1-x86_64.rpm
	uri, err := url.Parse(upstream)
	if err != nil {
		return "", errors.New(err, "invalid upstream URI", errors.TypeConfig)
	}

	uri.Path = path.Join(uri.Path, artifactName, packageName)
	return uri.String(), nil
}

func (e *Downloader) download(ctx context.Context, operatingSystem string, spec program.Spec, version string) (string, error) {
	filename, err := artifact.GetArtifactName(spec, version, operatingSystem, e.config.Arch())
	if err != nil {
		return "", errors.New(err, "generating package name failed")
	}

	fullPath, err := artifact.GetArtifactPath(spec, version, operatingSystem, e.config.Arch(), e.config.TargetDirectory)
	if err != nil {
		return "", errors.New(err, "generating package path failed")
	}

	return e.downloadFile(ctx, spec.Artifact, filename, fullPath)
}

func (e *Downloader) downloadHash(ctx context.Context, operatingSystem string, spec program.Spec, version string) (string, error) {
	filename, err := artifact.GetArtifactName(spec, version, operatingSystem, e.config.Arch())
	if err != nil {
		return "", errors.New(err, "generating package name failed")
	}

	fullPath, err := artifact.GetArtifactPath(spec, version, operatingSystem, e.config.Arch(), e.config.TargetDirectory)
	if err != nil {
		return "", errors.New(err, "generating package path failed")
	}

	filename = filename + ".sha512"
	fullPath = fullPath + ".sha512"

	return e.downloadFile(ctx, spec.Artifact, filename, fullPath)
}

func (e *Downloader) downloadFile(ctx context.Context, artifactName, filename, fullPath string) (string, error) {
	sourceURI, err := e.composeURI(artifactName, filename)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("GET", sourceURI, nil)
	if err != nil {
		return "", errors.New(err, "fetching package failed", errors.TypeNetwork, errors.M(errors.MetaKeyURI, sourceURI))
	}

	destinationFile, err := os.OpenFile(fullPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, packagePermissions)
	if err != nil {
		return "", errors.New(err, "creating package file failed", errors.TypeFilesystem, errors.M(errors.MetaKeyPath, fullPath))
	}
	defer destinationFile.Close()

	resp, err := e.client.Do(req.WithContext(ctx))
	if err != nil {
		return "", errors.New(err, "fetching package failed", errors.TypeNetwork, errors.M(errors.MetaKeyURI, sourceURI))
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", errors.New(fmt.Sprintf("call to '%s' returned unsuccessful status code: %d", sourceURI, resp.StatusCode), errors.TypeNetwork, errors.M(errors.MetaKeyURI, sourceURI))
	}

	_, err = io.Copy(destinationFile, resp.Body)
	if err != nil {
		return "", errors.New(err, "fetching package failed", errors.TypeNetwork, errors.M(errors.MetaKeyURI, sourceURI))
	}

	return fullPath, nil
}
