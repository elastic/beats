// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package source

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/elastic/beats/v8/libbeat/common/transport/httpcommon"
	"github.com/elastic/beats/v8/libbeat/logp"
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

var ErrNoEtag = fmt.Errorf("No ETag header in zip file response. Heartbeat requires an etag to efficiently cache downloaded code")

func (z *ZipURLSource) Validate() (err error) {
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

	tf, err := ioutil.TempFile("/tmp", "elastic-synthetics-zip-")
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
		os.MkdirAll(z.TargetDirectory, 0755)
	} else {
		z.TargetDirectory, err = ioutil.TempDir("/tmp", "elastic-synthetics-unzip-")
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
			os.RemoveAll(z.TargetDirectory)
			return fmt.Errorf("failed to install dependencies at: '%s' %w", z.TargetDirectory, err)
		}
	}

	return nil
}

func unzip(tf *os.File, targetDir string, folder string) error {
	stat, err := tf.Stat()
	if err != nil {
		return err
	}

	rdr, err := zip.NewReader(tf, stat.Size())
	if err != nil {
		return fmt.Errorf("could not read file %s as zip: %w", tf.Name(), err)
	}

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

func unzipFile(workdir string, folder string, f *zip.File) error {
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
	destPath := filepath.Join(workdir, filepath.Join(sansFolder...))

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
	req, err := http.NewRequest(method, z.URL, nil)
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

	io.Copy(tf, resp.Body)

	return
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
