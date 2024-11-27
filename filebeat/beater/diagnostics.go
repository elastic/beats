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
	"strings"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/paths"
)

func gzipRegistry() []byte {
	buf := bytes.Buffer{}
	dataPath := paths.Resolve(paths.Data, "")
	registryPath := filepath.Join(dataPath, "registry")
	f, err := os.CreateTemp("", "filebeat-registry-*.tar")
	if err != nil {
		panic(err)
	}
	f.Close()
	defer func() {
		if err := os.Remove(f.Name()); err != nil {
			logp.L().Named("diagnostics").Warnf("cannot remove temporary registry archive '%s': '%w'", f.Name(), err)
		}
	}()

	defer func() {
		if err := os.Remove(f.Name()); err != nil {
			panic(err)
		}
	}()

	tarFolder(registryPath, f.Name())
	gzipFile(f.Name(), &buf)

	return buf.Bytes()
}

// gzipFile gzips src writing the compressed data to dst
func gzipFile(src string, dst io.Writer) error {
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
func tarFolder(src, dst string) error {
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

	return filepath.Walk(fullPath, func(path string, info fs.FileInfo, err error) error {
		header, err := tar.FileInfoHeader(info, info.Name())
		header.Name = filepath.Join(baseDir, strings.TrimPrefix(path, src))

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

		if _, err := io.Copy(tarWriter, file); err != nil {
			return fmt.Errorf("cannot read '%s': '%w'", path, err)
		}

		return nil
	})
}
