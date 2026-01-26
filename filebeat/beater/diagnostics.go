// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package beater

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/paths"
)

func getRegexpsForRegistryFiles() ([]*regexp.Regexp, error) {
	// We use regexps here because globs do not support specifying a character
	// range like we do in the checkpoint file.

	registryFileRegExps := []*regexp.Regexp{}
	preFilesList := [][]string{
		{"^registry$"},
		{"^registry", "filebeat$"},
		{"^registry", "filebeat", "meta\\.json$"},
		{"^registry", "filebeat", "log\\.json$"},
		{"^registry", "filebeat", "active\\.dat$"},
		{"^registry", "filebeat", "[[:digit:]]*\\.json$"},
	}

	for _, lst := range preFilesList {
		var path string
		if filepath.Separator == '\\' {
			path = strings.Join(lst, `\\`)
		} else {
			path = filepath.Join(lst...)
		}

		// Compile the reg exp, if there is an error, stop and return.
		// There should be no error here as this code is tested in all
		// supported OSes, however to avoid a code path that leads to a
		// panic, we cannot use `regexp.MustCompile` and handle the error
		re, err := regexp.Compile(path)
		if err != nil {
			return nil, fmt.Errorf("cannot compile reg exp: %w", err)
		}

		registryFileRegExps = append(registryFileRegExps, re)
	}

	return registryFileRegExps, nil
}

func gzipRegistry(logger *logp.Logger, beatPaths *paths.Path) func() []byte {
	logger = logger.Named("diagnostics")

	return func() []byte {
		buf := bytes.Buffer{}
		dataPath := beatPaths.Resolve(paths.Data, "")
		registryPath := filepath.Join(dataPath, "registry")
		f, err := os.CreateTemp("", "filebeat-registry-*.tar")
		if err != nil {
			logger.Errorw("cannot create temporary registry archive", "error.message", err)
		}
		// Close the file, we just need the empty file created to use it later
		f.Close()
		defer logger.Debug("finished gziping Filebeat's registry")

		defer func() {
			if err := os.Remove(f.Name()); err != nil {
				logger.Warnf("cannot remove temporary registry archive '%s': '%s'", f.Name(), err)
			}
		}()

		logger.Debugf("temporary file '%s' created", f.Name())
		if err := tarFolder(logger, registryPath, f.Name()); err != nil {
			logger.Errorw(fmt.Sprintf("cannot archive Filebeat's registry at '%s'", f.Name()), "error.message", err)
		}

		if err := gzipFile(logger, f.Name(), &buf); err != nil {
			logger.Errorw("cannot gzip Filebeat's registry", "error.message", err)
		}

		// if the final file is too large, skip it
		if buf.Len() >= 20_000_000 { // 20 Mb
			logger.Warnf("registry is too large for diagnostics, %dmb bytes > 20mb", buf.Len()/1_000_000)
			return nil
		}

		return buf.Bytes()
	}
}

// gzipFile gzips src writing the compressed data to dst
func gzipFile(logger *logp.Logger, src string, dst io.Writer) error {
	reader, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("cannot open '%s': '%w'", src, err)
	}
	defer reader.Close()

	writer := gzip.NewWriter(dst)
	defer writer.Close()
	writer.Name = filepath.Base(src)

	if _, err := io.Copy(writer, reader); err != nil {
		if err != nil {
			return fmt.Errorf("cannot gzip file '%s': '%w'", src, err)
		}
	}

	return nil
}

// tarFolder creates a tar archive from the folder src and stores it at dst.
//
// dst must be the full path with extension, e.g: /tmp/foo.tar
// If src is not a folder an error is retruned
func tarFolder(logger *logp.Logger, src, dst string) error {
	fullPath, err := filepath.Abs(src)
	if err != nil {
		return fmt.Errorf("cannot get full path from '%s': '%w'", src, err)
	}

	tarFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("cannot create tar file '%s': '%w'", dst, err)
	}
	defer tarFile.Close()

	tarWriter := tar.NewWriter(tarFile)
	defer tarWriter.Close()

	info, err := os.Stat(fullPath)
	if err != nil {
		return fmt.Errorf("cannot stat '%s': '%w'", fullPath, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("'%s' is not a directory", fullPath)
	}
	baseDir := filepath.Base(src)

	logger.Debugf("starting to walk '%s'", fullPath)

	// This error should never happen at runtime, if something
	// breaks it should break the tests and be fixed before a
	// release. We handle the error here to avoid a code path
	// that can end into a panic.
	registryFileRegExps, err := getRegexpsForRegistryFiles()
	if err != nil {
		return err
	}

	return filepath.Walk(fullPath, func(path string, info fs.FileInfo, prevErr error) error {
		// Stop if there is any errors
		if prevErr != nil {
			return prevErr
		}

		pathInTar := filepath.Join(baseDir, strings.TrimPrefix(path, src))
		if !matchRegistyFiles(registryFileRegExps, pathInTar) {
			return nil
		}
		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return fmt.Errorf("cannot create tar info header: '%w'", err)
		}
		header.Name = pathInTar

		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("cannot write tar header for '%s': '%w'", path, err)
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("cannot open '%s' for reading: '%w", path, err)
		}
		defer file.Close()

		logger.Debugf("adding '%s' to the tar archive", file.Name())
		if _, err := io.Copy(tarWriter, file); err != nil {
			return fmt.Errorf("cannot read '%s': '%w'", path, err)
		}

		return nil
	})
}

func matchRegistyFiles(registryFileRegExps []*regexp.Regexp, path string) bool {
	for _, regExp := range registryFileRegExps {
		if regExp.MatchString(path) {
			return true
		}
	}
	return false
}
