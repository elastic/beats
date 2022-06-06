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
	"strconv"
	"strings"
	"time"

	"github.com/docker/go-units"

	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/beats/v7/libbeat/common/transport/httpcommon"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"
)

const (
	packagePermissions = 0660

	// downloadProgressIntervalPercentage defines how often to report the current download progress when percentage
	// of time has passed in the overall interval for the complete download to complete. 5% is a good default, as
	// the default timeout is 10 minutes and this will have it log every 30 seconds.
	downloadProgressIntervalPercentage = 0.05

	// warningProgressIntervalPercentage defines how often to log messages as a warning once the amount of time
	// passed is this percentage or more of the total allotted time to download.
	warningProgressIntervalPercentage = 0.75
)

var headers = map[string]string{
	"User-Agent": fmt.Sprintf("Beat elastic-agent v%s", release.Version()),
}

// Downloader is a downloader able to fetch artifacts from elastic.co web page.
type Downloader struct {
	log    progressLogger
	config *artifact.Config
	client http.Client
}

// NewDownloader creates and configures Elastic Downloader
func NewDownloader(log progressLogger, config *artifact.Config) (*Downloader, error) {
	client, err := config.HTTPTransportSettings.Client(
		httpcommon.WithAPMHTTPInstrumentation(),
	)
	if err != nil {
		return nil, err
	}

	client.Transport = withHeaders(client.Transport, headers)
	return NewDownloaderWithClient(log, config, *client), nil
}

// NewDownloaderWithClient creates Elastic Downloader with specific client used
func NewDownloaderWithClient(log progressLogger, config *artifact.Config, client http.Client) *Downloader {
	return &Downloader{
		log:    log,
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

	fileSize := -1
	if contentLength := resp.Header.Get("Content-Length"); contentLength != "" {
		if length, err := strconv.Atoi(contentLength); err == nil {
			fileSize = length
		}
	}

	reportCtx, reportCancel := context.WithCancel(ctx)
	dp := newDownloadProgressReporter(e.log, sourceURI, e.config.HTTPTransportSettings.Timeout, fileSize)
	dp.Report(reportCtx)
	_, err = io.Copy(destinationFile, io.TeeReader(resp.Body, dp))
	if err != nil {
		reportCancel()
		dp.ReportFailed(err)
		return "", errors.New(err, "fetching package failed", errors.TypeNetwork, errors.M(errors.MetaKeyURI, sourceURI))
	}
	reportCancel()
	dp.ReportComplete()

	return fullPath, nil
}

type downloadProgressReporter struct {
	log         progressLogger
	sourceURI   string
	timeout     time.Duration
	interval    time.Duration
	warnTimeout time.Duration
	length      float64

	downloaded atomic.Int
	started    time.Time
}

func newDownloadProgressReporter(log progressLogger, sourceURI string, timeout time.Duration, length int) *downloadProgressReporter {
	return &downloadProgressReporter{
		log:         log,
		sourceURI:   sourceURI,
		timeout:     timeout,
		interval:    time.Duration(float64(timeout) * downloadProgressIntervalPercentage),
		warnTimeout: time.Duration(float64(timeout) * warningProgressIntervalPercentage),
		length:      float64(length),
	}
}

func (dp *downloadProgressReporter) Write(b []byte) (int, error) {
	n := len(b)
	dp.downloaded.Add(n)
	return n, nil
}

func (dp *downloadProgressReporter) Report(ctx context.Context) {
	started := time.Now()
	dp.started = started
	sourceURI := dp.sourceURI
	log := dp.log
	length := dp.length
	warnTimeout := dp.warnTimeout
	interval := dp.interval

	go func() {
		t := time.NewTimer(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				now := time.Now()
				timePast := now.Sub(started)
				downloaded := float64(dp.downloaded.Load())
				bytesPerSecond := downloaded / float64(timePast/time.Second)

				var msg string
				var args []interface{}
				if length > 0 {
					// length of the download is known, so more detail can be provided
					percentComplete := downloaded / length * 100.0
					msg = "download progress from %s is %s/%s (%.2f%% complete) @ %sps"
					args = []interface{}{
						sourceURI, units.HumanSize(downloaded), units.HumanSize(length), percentComplete, units.HumanSize(bytesPerSecond),
					}
				} else {
					// length unknown so provide the amount downloaded and the speed
					msg = "download progress from %s has fetched %s @ %sps"
					args = []interface{}{
						sourceURI, units.HumanSize(downloaded), units.HumanSize(bytesPerSecond),
					}
				}

				log.Infof(msg, args...)
				if timePast >= warnTimeout {
					// duplicate to warn when over the warnTimeout; this still has it logging to info that way if
					// they are filtering the logs to info they still see the messages when over the warnTimeout, but
					// when filtering only by warn they see these messages only
					log.Warnf(msg, args...)
				}
			}
		}
	}()
}

func (dp *downloadProgressReporter) ReportComplete() {
	now := time.Now()
	timePast := now.Sub(dp.started)
	downloaded := float64(dp.downloaded.Load())
	bytesPerSecond := downloaded / float64(timePast/time.Second)
	msg := "download from %s completed in %s @ %sps"
	args := []interface{}{
		dp.sourceURI, units.HumanDuration(timePast), units.HumanSize(bytesPerSecond),
	}
	dp.log.Infof(msg, args...)
	if timePast >= dp.warnTimeout {
		// see reason in `Report`
		dp.log.Warnf(msg, args...)
	}
}

func (dp *downloadProgressReporter) ReportFailed(err error) {
	now := time.Now()
	timePast := now.Sub(dp.started)
	downloaded := float64(dp.downloaded.Load())
	bytesPerSecond := downloaded / float64(timePast/time.Second)
	var msg string
	var args []interface{}
	if dp.length > 0 {
		// length of the download is known, so more detail can be provided
		percentComplete := downloaded / dp.length * 100.0
		msg = "download from %s failed at %s/%s (%.2f%% complete) @ %sps: %s"
		args = []interface{}{
			dp.sourceURI, units.HumanSize(downloaded), units.HumanSize(dp.length), percentComplete, units.HumanSize(bytesPerSecond), err,
		}
	} else {
		// length unknown so provide the amount downloaded and the speed
		msg = "download from %s failed at %s @ %sps: %s"
		args = []interface{}{
			dp.sourceURI, units.HumanSize(downloaded), units.HumanSize(bytesPerSecond), err,
		}
	}
	dp.log.Infof(msg, args...)
	if timePast >= dp.warnTimeout {
		// see reason in `Report`
		dp.log.Warnf(msg, args...)
	}
}

// progressLogger is a logger that only needs to implement Infof and Warnf, as those are the only functions
// that the downloadProgressReporter uses.
type progressLogger interface {
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
}
