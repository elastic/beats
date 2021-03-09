package source

import (
	"archive/zip"
	"fmt"
	"github.com/elastic/beats/v7/libbeat/logp"
	"io"
	"io/ioutil"
	"net/http"
	"os"
)

type ZipURLSource struct {
	URL    string `config:"url"`
	Subdirectory string `config:"subdirectory"`
	BaseSource
	// Etag from last successful fetch
	etag string
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

	newWorkDir, err := ioutil.TempDir("/tmp", "elastic-synthetics-unzip-")
	if err != nil {
		return fmt.Errorf("could not make temp dir for zip download: %w", err)
	}

	err = unzip(tf, newWorkDir)
	if err != nil {
		os.RemoveAll(newWorkDir)
	}

	return nil
}

func unzip(tf *os.File, dir string) error {
	stat, err := tf.Stat()
	if err != nil {
		return err
	}

	rdr, err := zip.NewReader(tf, stat.Size())
	if err != nil {
		return fmt.Errorf("could not read file %s as zip: %w", tf.Name(), err)
	}

	for _, f := range rdr.File {
		logp.Info("FNAME", f.Name)
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
	if  etag == "" {
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
	panic("implement me")
}

func (z *ZipURLSource) Close() error {
	panic("implement me")
}
