// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package bundle

import (
	"archive/zip"
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

// ReadCloserWith takes a reader and a closer for the specific reader and return an io.ReaderCloser.
func ReadCloserWith(reader io.Reader, closer io.Closer) io.ReadCloser {
	return &ReadCloser{reader: reader, closer: closer}
}

// ReadCloser wraps a io.Reader and a file handle into a FileReadCloser interface,
// this leave the responsability on the consumer to close the handle when its done consuming the
// io.Reader.
type ReadCloser struct {
	reader io.Reader
	closer io.Closer
}

// Read proxies the Read to the original io.Reader.
func (f *ReadCloser) Read(p []byte) (int, error) {
	return f.reader.Read(p)
}

// Close closes the file handle this must be called after consuming the io.Reader to make sure we
// don't leak any file handle.
func (f *ReadCloser) Close() error {
	return f.closer.Close()
}

// Resource is the interface used to bundle the resource, a resource can be a local or a remote file.
// Reader must be a io.ReadCloser, this make it easier to deal with streaming of remote data.
type Resource interface {
	// Open return an io.ReadCloser of the original resource, this will be used to stream content to
	// The compressed file.
	Open() (io.ReadCloser, error)

	// Name return the string that will be used as the file name inside the Zip file.
	Name() string

	// Mode returns the permission of the file.
	Mode() os.FileMode
}

// Folder returns a list of files in a folder.
func Folder(folder, root string, filemode os.FileMode) []Resource {
	resources := make([]Resource, 0)
	err := filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		resources = append(resources, &RelativeFile{LocalFile{Path: path, FileMode: filemode}, root})

		return nil
	})

	if err != nil {
		return nil
	}

	return resources
}

// LocalFile represents a local file on disk.
type LocalFile struct {
	Path     string
	FileMode os.FileMode
}

// Open return a reader for the opened file.
func (l *LocalFile) Open() (io.ReadCloser, error) {
	fd, err := os.Open(l.Path)
	if err != nil {
		return nil, err
	}

	reader := bufio.NewReader(fd)
	return ReadCloserWith(reader, fd), nil
}

// Name return the basename of the file to be used as the name of the file in the archive.
func (l *LocalFile) Name() string {
	return filepath.Base(l.Path)
}

// Mode return the permissions of the file in the zip.
func (l *LocalFile) Mode() os.FileMode {
	return l.FileMode
}

// RelativeFile is a Localfile which needs to be placed relatively in the
// root of the bundle.
type RelativeFile struct {
	LocalFile
	root string
}

// Name returns the name of the file.
func (r *RelativeFile) Name() string {
	if r.root == "" {
		return r.LocalFile.Path
	}
	return r.LocalFile.Path[len(r.root)+1:]
}

// MemoryFile an in-memory representation of a physical file.
type MemoryFile struct {
	Path     string
	FileMode os.FileMode
	Raw      []byte
}

// Open the reader for the raw byte slice.
func (m *MemoryFile) Open() (io.ReadCloser, error) {
	reader := bytes.NewReader(m.Raw)
	return ioutil.NopCloser(reader), nil
}

// Name returns the path to use in the zip.
func (m *MemoryFile) Name() string {
	return m.Path
}

// Mode returns the permission of the file.
func (m *MemoryFile) Mode() os.FileMode {
	return m.FileMode
}

// ZipBundle accepts a set of local files to bundle them into a zip file, it also accept size limits
// for the uncompressed and the compressed data.
type ZipBundle struct {
	resources           []Resource
	maxSizeUncompressed int64
	maxSizeCompressed   int64
}

// NewZipWithoutLimits creates a bundle that doesn't impose any limit on the uncompressed data and the
// compressed data.
func NewZipWithoutLimits(resources ...Resource) *ZipBundle {
	return NewZipWithLimits(-1, -1, resources...)
}

// NewZipWithLimits creates a Bundle that impose limit for the uncompressed data and the compressed data,
// using a limit of -1 with desactivate the check.
func NewZipWithLimits(maxSizeUncompressed, maxSizeCompressed int64, resources ...Resource) *ZipBundle {
	return &ZipBundle{
		resources:           resources,
		maxSizeUncompressed: maxSizeUncompressed,
		maxSizeCompressed:   maxSizeCompressed,
	}
}

// Bytes takes the resources and bundle them into a zip and validates if needed that the
// created resources doesn't go over any predefined size limits.
func (p *ZipBundle) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	var uncompressed int64
	for _, file := range p.resources {
		l, err := zipAddFile(zipWriter, file)
		if err != nil {
			return nil, err
		}

		uncompressed = uncompressed + l
		if p.maxSizeUncompressed != -1 && uncompressed > p.maxSizeUncompressed {
			// Close the current zip, the zip has incomplete data.
			zipWriter.Close()
			return nil, fmt.Errorf(
				"max uncompressed size reached, size %d, limit is %d",
				uncompressed,
				p.maxSizeUncompressed,
			)
		}

		if l == 0 {
			continue
		}

		// Force a flush to accurately check for the size of the bytes.Buffer and see if
		// we are over the limit.
		if err := zipWriter.Flush(); err != nil {
			return nil, err
		}

		if p.maxSizeCompressed != -1 && int64(buf.Len()) > p.maxSizeCompressed {
			// Close the current zip, the zip has incomplete data.
			zipWriter.Close()
			return nil, fmt.Errorf(
				"max compressed size reached, size %d, limit is %d",
				buf.Len(),
				p.maxSizeCompressed,
			)
		}
	}

	// Flush bytes/writes headers, the zip is valid at this point.
	zipWriter.Close()
	return buf.Bytes(), nil
}

func zipAddFile(zipWriter *zip.Writer, r Resource) (int64, error) {
	f, err := r.Open()
	if err != nil {
		return 0, err
	}
	defer f.Close()

	header := &zip.FileHeader{
		Name:   r.Name(),
		Method: zip.Deflate,
	}

	header.SetMode(r.Mode())
	w, err := zipWriter.CreateHeader(header)
	if err != nil {
		return 0, err
	}

	l, err := io.Copy(w, f)
	if err != nil {
		return 0, err
	}

	return l, nil
}
