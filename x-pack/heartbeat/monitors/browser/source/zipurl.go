// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux || darwin
// +build linux darwin

package source

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
)

type ZipURLSource struct {
	URL      string `config:"url" json:"url"`
	Folder   string `config:"folder" json:"folder"`
	Username string `config:"username" json:"username"`
	Password string `config:"password" json:"password"`
	Retries  int    `config:"retries" default:"3" json:"retries"`
	BaseSource
	TargetDirectory string `config:"target_directory" json:"target_directory"`

	// Etag from last successful fetch
	etag string

	Transport httpcommon.HTTPTransportSettings `config:",inline" yaml:",inline"`

	httpClient *http.Client
}

var ErrNoEtag = fmt.Errorf("no ETag header in zip file response. Heartbeat requires an etag to efficiently cache downloaded code")

func (z *ZipURLSource) Validate() (err error) {
	logp.L().Warn("Zip URL browser monitors are now deprecated! Please use project monitors instead. See the Elastic synthetics docs at https://www.elastic.co/guide/en/observability/current/synthetic-run-tests.html")
	if z.httpClient == nil {
		z.httpClient, _ = z.Transport.Client()
	}
	return err
}

func (z *ZipURLSource) Fetch() error {
	changed, err := checkIfChanged(z)
	if err != nil {
		return fmt.Errorf("could not check if zip source changed for %s: %w", z.URL, err)
	}
	if !changed {
		return nil
	}

	// remove target directory if etag changed
	if z.TargetDirectory != "" {
		os.RemoveAll(z.TargetDirectory)
	}

	tf, err := ioutil.TempFile(os.TempDir(), "elastic-synthetics-zip-")
	if err != nil {
		return fmt.Errorf("could not create tmpfile for zip source: %w", err)
	}
	defer os.Remove(tf.Name())

	newEtag, err := download(z, tf)
	if err != nil {
		return fmt.Errorf("could not download %s: %w", z.URL, err)
	}
	// We are guaranteed an etag
	z.etag = newEtag

	if z.TargetDirectory != "" {
		err := os.MkdirAll(z.TargetDirectory, 0755)
		if err != nil {
			return fmt.Errorf("could not make directory %s: %w", z.TargetDirectory, err)
		}
	} else {
		z.TargetDirectory, err = ioutil.TempDir(os.TempDir(), "elastic-synthetics-unzip-")
		if err != nil {
			return fmt.Errorf("could not make temp dir for zip download: %w", err)
		}
	}

	err = unzip(tf, z.TargetDirectory, z.Folder)
	if err != nil {
		z.Close()
		return err
	}

	// run as the local job after extracting the files
	if !Offline() {
		err = setupOnlineDir(z.TargetDirectory)
		if err != nil {
			z.Close()
			return fmt.Errorf("failed to install dependencies at: '%s' %w", z.TargetDirectory, err)
		}
	}

	return nil
}

func unzip(tf *os.File, targetDir string, folder string) error {
	rdr, err := zip.OpenReader(tf.Name())
	if err != nil {
		return err
	}
	defer rdr.Close()

	for _, f := range rdr.File {
		err = unzipFile(targetDir, folder, f)
		if err != nil {
			rmErr := os.RemoveAll(targetDir)
			if rmErr != nil {
				return fmt.Errorf("could not remove directory after encountering error unzipping file: %w, (original unzip error: %s)", rmErr, err)
			}
			return err
		}
	}
	return nil
}

func sanitizeFilePath(filePath string, workdir string) (string, error) {
	destPath := filepath.Join(workdir, filePath)
	if !strings.HasPrefix(destPath, filepath.Clean(workdir)+string(os.PathSeparator)) {
		return filePath, fmt.Errorf("failed to extract illegal file path: %s", filePath)
	}
	return destPath, nil
}

// unzip file takes a given directory and a zipped file and extracts
// all the contents of the file based on the provided folder path,
// if the folder path is empty, it extracts the contents based on file
// tree structure
func unzipFile(workdir string, folder string, f *zip.File) error {
	var destPath string
	var err error
	if folder != "" {
		folderPaths := strings.Split(folder, string(filepath.Separator))
		var folderDepth = 1
		for _, path := range folderPaths {
			if path != "" {
				folderDepth++
			}
		}
		splitZipFileName := strings.Split(f.Name, string(filepath.Separator))
		root := splitZipFileName[0]

		prefix := filepath.Join(root, folder)
		if !strings.HasPrefix(f.Name, prefix) {
			return nil
		}

		sansFolder := splitZipFileName[folderDepth:]
		destPath = filepath.Join(workdir, filepath.Join(sansFolder...))
	} else {
		destPath, err = sanitizeFilePath(f.Name, workdir)
		if err != nil {
			return err
		}
	}

	// Never unpack node modules
	if strings.HasPrefix(destPath, "node_modules/") {
		return nil
	}

	if f.FileInfo().IsDir() {
		err := os.MkdirAll(destPath, 0755)
		if err != nil {
			return fmt.Errorf("could not make dest zip dir '%s': %w", destPath, err)
		}
		return nil
	}

	// In the case of project monitors, the destPath would be the direct
	// file path instead of directory, so we create the directory
	// if its not set up properly
	destDir := filepath.Dir(destPath)
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		err = os.MkdirAll(destDir, 0700) // Create your file
		if err != nil {
			return fmt.Errorf("could not make dest zip dir '%s': %w", destDir, err)
		}
	}

	dest, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("could not create dest file for zip '%s': %w", destPath, err)
	}
	defer dest.Close()

	rdr, err := f.Open()
	if err != nil {
		return fmt.Errorf("could not open source zip file '%s': %w", f.Name, err)
	}
	defer rdr.Close()

	// Cap decompression to a max of 2GiB to prevent decompression bombs
	//nolint:gosec // zip bomb possibility, but user controls the zip, so it would only impact them
	_, err = io.Copy(dest, rdr)
	if err != nil {
		return err
	}

	return nil
}

func retryingZipRequest(method string, z *ZipURLSource) (resp *http.Response, err error) {
	if z.Retries < 1 {
		z.Retries = 1
	}
	for i := z.Retries; i > 0; i-- {
		resp, err = zipRequest(method, z)
		// If the request is successful
		// Retry server errors, but not non-retryable 4xx errors
		if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 500 {
			break
		}
		if err == nil {
			resp.Body.Close()
		}
		logp.Warn("attempt to download zip at %s failed: %s, will retry in 1s", z.URL, err)
		time.Sleep(time.Second)
	}
	if resp != nil && resp.StatusCode > 300 {
		return nil, fmt.Errorf("failed to retrieve zip, received status of %d requesting zip URL", resp.StatusCode)
	}
	return resp, err
}

func zipRequest(method string, z *ZipURLSource) (*http.Response, error) {
	req, err := http.NewRequestWithContext(context.TODO(), method, z.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("could not issue request to: %s %w", z.URL, err)
	}
	if z.Username != "" && z.Password != "" {
		req.SetBasicAuth(z.Username, z.Password)
	}
	return z.httpClient.Do(req)
}

func download(z *ZipURLSource, tf *os.File) (etag string, err error) {
	resp, err := retryingZipRequest("GET", z)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	etag = resp.Header.Get("ETag")
	if etag == "" {
		return "", ErrNoEtag
	}

	_, err = io.Copy(tf, resp.Body)
	if err != nil {
		return "", fmt.Errorf("could not copy resp body: %w", err)
	}

	return etag, nil
}

func checkIfChanged(z *ZipURLSource) (bool, error) {
	resp, err := retryingZipRequest("HEAD", z)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// If the etag matches what we already have on file, skip this
	if resp.Header.Get("ETag") == "" {
		return false, ErrNoEtag
	}
	// Nothing has changed since the last fetch, so we can just abort
	if resp.Header.Get("ETag") == z.etag {
		return false, nil
	}

	return true, nil
}

func (z *ZipURLSource) Workdir() string {
	return z.TargetDirectory
}

func (z *ZipURLSource) Close() error {
	return os.RemoveAll(z.TargetDirectory)
}
