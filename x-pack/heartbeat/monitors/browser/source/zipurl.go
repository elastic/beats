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
)

type ZipURLSource struct {
	URL      string `config:"url" json:"url"`
	Folder   string `config:"folder" json:"folder"`
	Username string `config:"username" json:"username"`
	Password string `config:"password" json:"password"`
	BaseSource
	// Etag from last successful fetch
	etag            string
	TargetDirectory string `config:"target_directory" json:"target_directory"`
}

var ErrNoEtag = fmt.Errorf("No ETag header in zip file response. Heartbeat requires an etag to efficiently cache downloaded code")

func (z *ZipURLSource) Fetch() error {
	changed, err := checkIfChanged(z.URL, z.etag)
	if err != nil {
		return fmt.Errorf("could not check if zip source changed for %s: %w", z.URL, err)
	}
	if !changed {
		return nil
	}
	tf, err := ioutil.TempFile("/tmp", "elastic-synthetics-zip-")
	if err != nil {
		return fmt.Errorf("could not create tmpfile for zip source: %w", err)
	}
	defer os.Remove(tf.Name())
	newEtag, err := download(z.URL, tf)
	if err != nil {
		return fmt.Errorf("could not download %s: %w", z.URL, err)
	}
	// We are guaranteed an etag
	z.etag = newEtag

	if z.TargetDirectory != "" {
		os.MkdirAll(z.TargetDirectory, 0755)
	} else {
		z.TargetDirectory, err = ioutil.TempDir("/tmp/oneshot", "elastic-synthetics-unzip-")
		if err != nil {
			return fmt.Errorf("could not make temp dir for zip download: %w", err)
		}
	}

	err = unzip(tf, z.TargetDirectory, z.Folder)
	if err != nil {
		os.RemoveAll(z.TargetDirectory)
		return err
	}

	return nil
}

func unzip(tf *os.File, dir string, folder string) error {
	stat, err := tf.Stat()
	if err != nil {
		return err
	}

	rdr, err := zip.NewReader(tf, stat.Size())
	if err != nil {
		return fmt.Errorf("could not read file %s as zip: %w", tf.Name(), err)
	}

	for _, f := range rdr.File {
		err = unzipFile(dir, folder, f)
		if err != nil {
			// TODO: err handler
			os.RemoveAll(dir)
			return err
		}
	}
	return nil
}

func unzipFile(workdir string, folder string, f *zip.File) error {
	folderDepth := len(strings.Split(folder, string(filepath.Separator))) + 1
	splitZipName := strings.Split(f.Name, string(filepath.Separator))
	root := splitZipName[0]

	prefix := filepath.Join(root, folder)
	if !strings.HasPrefix(f.Name, prefix) {
		return nil
	}

	sansFolder := strings.Split(f.Name, string(filepath.Separator))[folderDepth:]
	destPath := filepath.Join(workdir, filepath.Join(sansFolder...))
	outName, err := filepath.Rel(workdir, destPath)

	// Never unpack node modules
	if strings.HasPrefix(outName, "node_modules/") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("relpath err: %s", err)
	}
	// if !strings.HasPrefix(outName, workdir) {
	// 	return fmt.Errorf("security error unpacking zip: %s -> %s", f.Name, outName)
	// }

	if f.FileInfo().IsDir() {
		err := os.MkdirAll(outName, 0755)
		if err != nil {
			return fmt.Errorf("could not make dest zip dir '%s': %w", outName, err)
		}
		return nil
	}

	dest, err := os.Create(outName)
	if err != nil {
		return fmt.Errorf("could not open dest file for zip '%s': %w", outName, err)
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

func download(url string, tf *os.File) (etag string, err error) {
	resp, err := http.Get(url)
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

func checkIfChanged(zipUrl string, etag string) (bool, error) {
	resp, err := http.Head(zipUrl)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// If the etag matches what we already have on file, skip this
	if resp.Header.Get("ETag") == "" {
		return false, ErrNoEtag
	}
	// Nothing has changed since the last fetch, so we can just abort
	if resp.Header.Get("ETag") == etag {
		return false, nil
	}

	return true, nil
}

func (z *ZipURLSource) Workdir() string {
	return z.TargetDirectory
}

func (z *ZipURLSource) Close() error {
	return nil
}
