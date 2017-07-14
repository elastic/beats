package dev_tools

// This file contains tests that can be run on the generated packages.
// To run these tests use `go test package_test.go`.

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"flag"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/blakesmith/ar"
	"github.com/cavaliercoder/go-rpm"
)

const (
	expectedConfigMode     = os.FileMode(0600)
	expectedManifestMode   = os.FileMode(0644)
	expectedModuleFileMode = expectedManifestMode
	expectedModuleDirMode  = os.FileMode(0755)
	expectedConfigUID      = 0
	expectedConfigGID      = 0
)

var (
	configFilePattern   = regexp.MustCompile(`.*beat\.yml`)
	manifestFilePattern = regexp.MustCompile(`manifest.yml`)
	modulesDirPattern   = regexp.MustCompile(`modules.d/$`)
	modulesFilePattern  = regexp.MustCompile(`modules.d/.+`)
)

var (
	files = flag.String("files", "../build/upload/*/*", "filepath glob containing package files")
)

func TestRPM(t *testing.T) {
	rpms := getFiles(t, regexp.MustCompile(`\.rpm$`))
	for _, rpm := range rpms {
		checkRPM(t, rpm)
	}
}

func TestDeb(t *testing.T) {
	debs := getFiles(t, regexp.MustCompile(`\.deb$`))
	buf := new(bytes.Buffer)
	for _, deb := range debs {
		checkDeb(t, deb, buf)
	}
}

func TestTar(t *testing.T) {
	tars := getFiles(t, regexp.MustCompile(`\.tar\.gz$`))
	for _, tar := range tars {
		checkTar(t, tar)
	}
}

func TestZip(t *testing.T) {
	zips := getFiles(t, regexp.MustCompile(`^\w+beat-\S+.zip$`))
	for _, zip := range zips {
		checkZip(t, zip)
	}
}

// Sub-tests

func checkRPM(t *testing.T, file string) {
	p, err := readRPM(file)
	if err != nil {
		t.Error(err)
		return
	}

	checkConfigPermissions(t, p)
	checkConfigOwner(t, p)
	checkManifestPermissions(t, p)
	checkManifestOwner(t, p)
	checkModulesPermissions(t, p)
	checkModulesOwner(t, p)
}

func checkDeb(t *testing.T, file string, buf *bytes.Buffer) {
	p, err := readDeb(file, buf)
	if err != nil {
		t.Error(err)
		return
	}

	checkConfigPermissions(t, p)
	checkConfigOwner(t, p)
	checkManifestPermissions(t, p)
	checkManifestOwner(t, p)
	checkModulesPermissions(t, p)
	checkModulesOwner(t, p)
}

func checkTar(t *testing.T, file string) {
	p, err := readTar(file)
	if err != nil {
		t.Error(err)
		return
	}

	checkConfigPermissions(t, p)
	checkConfigOwner(t, p)
	checkManifestPermissions(t, p)
	checkModulesPermissions(t, p)
	checkModulesOwner(t, p)
}

func checkZip(t *testing.T, file string) {
	p, err := readZip(file)
	if err != nil {
		t.Error(err)
		return
	}

	checkConfigPermissions(t, p)
	checkManifestPermissions(t, p)
	checkModulesPermissions(t, p)
}

// Verify that the main configuration file is installed with a 0600 file mode.
func checkConfigPermissions(t *testing.T, p *packageFile) {
	t.Run(p.Name+" config file permissions", func(t *testing.T) {
		for _, entry := range p.Contents {
			if configFilePattern.MatchString(entry.File) {
				mode := entry.Mode.Perm()
				if expectedConfigMode != mode {
					t.Errorf("file %v has wrong permissions: expected=%v actual=%v",
						entry.File, expectedConfigMode, mode)
				}
				return
			}
		}
		t.Errorf("no config file found matching %v", configFilePattern)
	})
}

func checkConfigOwner(t *testing.T, p *packageFile) {
	t.Run(p.Name+" config file owner", func(t *testing.T) {
		for _, entry := range p.Contents {
			if configFilePattern.MatchString(entry.File) {
				if expectedConfigUID != entry.UID {
					t.Errorf("file %v should be owned by user %v, owner=%v", entry.File, expectedConfigGID, entry.UID)
				}
				if expectedConfigGID != entry.GID {
					t.Errorf("file %v should be owned by group %v, group=%v", entry.File, expectedConfigGID, entry.GID)
				}
				return
			}
		}
		t.Errorf("no config file found matching %v", configFilePattern)
	})
}

// Verify that the modules manifest.yml files are installed with a 0644 file mode.
func checkManifestPermissions(t *testing.T, p *packageFile) {
	t.Run(p.Name+" manifest file permissions", func(t *testing.T) {
		for _, entry := range p.Contents {
			if manifestFilePattern.MatchString(entry.File) {
				mode := entry.Mode.Perm()
				if expectedManifestMode != mode {
					t.Errorf("file %v has wrong permissions: expected=%v actual=%v",
						entry.File, expectedManifestMode, mode)
				}
			}
		}
	})
}

// Verify that the manifest owner is root
func checkManifestOwner(t *testing.T, p *packageFile) {
	t.Run(p.Name+" manifest file owner", func(t *testing.T) {
		for _, entry := range p.Contents {
			if manifestFilePattern.MatchString(entry.File) {
				if expectedConfigUID != entry.UID {
					t.Errorf("file %v should be owned by user %v, owner=%v", entry.File, expectedConfigGID, entry.UID)
				}
				if expectedConfigGID != entry.GID {
					t.Errorf("file %v should be owned by group %v, group=%v", entry.File, expectedConfigGID, entry.GID)
				}
			}
		}
	})
}

// Verify the permissions of the modules.d dir and its contents.
func checkModulesPermissions(t *testing.T, p *packageFile) {
	t.Run(p.Name+" modules.d file permissions", func(t *testing.T) {
		for _, entry := range p.Contents {
			if modulesFilePattern.MatchString(entry.File) {
				mode := entry.Mode.Perm()
				if expectedModuleFileMode != mode {
					t.Errorf("file %v has wrong permissions: expected=%v actual=%v",
						entry.File, expectedModuleFileMode, mode)
				}
			} else if modulesDirPattern.MatchString(entry.File) {
				mode := entry.Mode.Perm()
				if expectedModuleDirMode != mode {
					t.Errorf("file %v has wrong permissions: expected=%v actual=%v",
						entry.File, expectedModuleDirMode, mode)
				}
			}
		}
	})
}

// Verify the owner of the modules.d dir and its contents.
func checkModulesOwner(t *testing.T, p *packageFile) {
	t.Run(p.Name+" modules.d file owner", func(t *testing.T) {
		for _, entry := range p.Contents {
			if modulesFilePattern.MatchString(entry.File) || modulesDirPattern.MatchString(entry.File) {
				if expectedConfigUID != entry.UID {
					t.Errorf("file %v should be owned by user %v, owner=%v", entry.File, expectedConfigGID, entry.UID)
				}
				if expectedConfigGID != entry.GID {
					t.Errorf("file %v should be owned by group %v, group=%v", entry.File, expectedConfigGID, entry.GID)
				}
			}
		}
	})
}

// Helpers

type packageFile struct {
	Name     string
	Contents map[string]packageEntry
}

type packageEntry struct {
	File string
	UID  int
	GID  int
	Mode os.FileMode
}

func getFiles(t *testing.T, pattern *regexp.Regexp) []string {
	matches, err := filepath.Glob(*files)
	if err != nil {
		t.Fatal(err)
	}

	files := matches[:0]
	for _, f := range matches {
		if pattern.MatchString(filepath.Base(f)) {
			files = append(files, f)
		}
	}

	return files
}

func readRPM(rpmFile string) (*packageFile, error) {
	p, err := rpm.OpenPackageFile(rpmFile)
	if err != nil {
		return nil, err
	}

	contents := p.Files()
	pf := &packageFile{Name: filepath.Base(rpmFile), Contents: map[string]packageEntry{}}

	for _, file := range contents {
		pf.Contents[file.Name()] = packageEntry{
			File: file.Name(),
			Mode: file.Mode(),
		}
	}

	return pf, nil
}

// readDeb reads the data.tar.gz file from the .deb.
func readDeb(debFile string, dataBuffer *bytes.Buffer) (*packageFile, error) {
	file, err := os.Open(debFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	arReader := ar.NewReader(file)
	for {
		header, err := arReader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		if strings.HasPrefix(header.Name, "data.tar.gz") {
			dataBuffer.Reset()
			_, err := io.Copy(dataBuffer, arReader)
			if err != nil {
				return nil, err
			}

			gz, err := gzip.NewReader(dataBuffer)
			if err != nil {
				return nil, err
			}
			defer gz.Close()

			return readTarContents(filepath.Base(debFile), gz)
		}
	}

	return nil, io.EOF
}

func readTar(tarFile string) (*packageFile, error) {
	file, err := os.Open(tarFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var fileReader io.ReadCloser = file
	if strings.HasSuffix(tarFile, ".gz") {
		if fileReader, err = gzip.NewReader(file); err != nil {
			return nil, err
		}
		defer fileReader.Close()
	}

	return readTarContents(filepath.Base(tarFile), fileReader)
}

func readTarContents(tarName string, data io.Reader) (*packageFile, error) {
	tarReader := tar.NewReader(data)

	p := &packageFile{Name: tarName, Contents: map[string]packageEntry{}}
	for {
		header, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		p.Contents[header.Name] = packageEntry{
			File: header.Name,
			UID:  header.Uid,
			GID:  header.Gid,
			Mode: os.FileMode(header.Mode),
		}
	}

	return p, nil
}

func readZip(zipFile string) (*packageFile, error) {
	r, err := zip.OpenReader(zipFile)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	p := &packageFile{Name: filepath.Base(zipFile), Contents: map[string]packageEntry{}}
	for _, f := range r.File {
		p.Contents[f.Name] = packageEntry{
			File: f.Name,
			Mode: f.Mode(),
		}
	}

	return p, nil
}
