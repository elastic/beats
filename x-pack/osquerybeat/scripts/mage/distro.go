// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mage

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	devtools "github.com/menderesk/beats/v7/dev-tools/mage"
	"github.com/menderesk/beats/v7/x-pack/osquerybeat/internal/distro"
	"github.com/menderesk/beats/v7/x-pack/osquerybeat/internal/fetch"
	"github.com/menderesk/beats/v7/x-pack/osquerybeat/internal/fileutil"
	"github.com/menderesk/beats/v7/x-pack/osquerybeat/internal/hash"
	"github.com/menderesk/beats/v7/x-pack/osquerybeat/internal/tar"
)

// FetchOsqueryDistros fetches Osquery official distros as a part of the build
func FetchOsqueryDistros() error {
	osArchs := OSArchs(devtools.Platforms)
	log.Printf("Fetch Osquery distros for %v", osArchs)

	for _, osarch := range osArchs {
		spec, err := distro.GetSpec(osarch)
		if err != nil {
			if errors.Is(err, distro.ErrUnsupportedOS) {
				log.Printf("The build spec %v is not supported, continue\n", spec)
				continue
			} else {
				return err
			}
		}
		log.Println("Found spec:", spec)

		fetched, err := checkCacheAndFetch(osarch, spec)
		if err != nil {
			return err
		}

		ifp := spec.DistroFilepath(distro.GetDataInstallDir(osarch))
		installFileExists, eerr := fileutil.FileExists(ifp)
		if eerr != nil {
			log.Printf("Failed to check if %s exists, %v", ifp, err)
		}
		// If the new distro is fetched extract osqueryd if allowed according to the spec
		// Currently the only supported is tar.gz extraction.
		// There is no good Go library for extraction the cpio compressed "Payload" from Mac OS X .pkg,
		// the few that I tried are limited and do not work. Maybe something to write for fun when time.
		// The MSI is tricky as well to do the crossplatform extraction, no good Go library.
		// So for Mac OS and Winderz the whole distro package is included and extracted
		// on the first run on the platform for now.
		if fetched || !installFileExists {
			err = extractOrCopy(osarch, spec)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func OSArchs(platforms devtools.BuildPlatformList) []distro.OSArch {
	mp := make(map[distro.OSArch]struct{})

	for _, platform := range platforms {
		var arch string
		name := platform.Name
		if idx := strings.Index(name, "/"); idx != -1 {
			arch = name[idx+1:]
			name = name[:idx]
		}
		mp[distro.OSArch{OS: name, Arch: arch}] = struct{}{}
	}

	res := make([]distro.OSArch, 0, len(mp))
	for name := range mp {
		res = append(res, name)
	}
	return res
}

func checkCacheAndFetch(osarch distro.OSArch, spec distro.Spec) (fetched bool, err error) {
	dir := distro.DataCacheDir
	if err = os.MkdirAll(dir, 0750); err != nil {
		return false, fmt.Errorf("failed to create dir %v, %w", dir, err)
	}

	var fileHash string
	url := spec.URL(osarch.OS)
	fp := spec.DistroFilepath(dir)
	specHash := spec.SHA256Hash

	// Check if file already exists in the cache
	f, err := os.Open(fp)
	if err != nil {
		if !os.IsNotExist(err) {
			return false, err
		}
	}

	// File exists, check hash
	if f != nil {
		log.Print("Cached file found: ", fp)
		fileHash, err = hash.Calculate(f, nil)
		f.Close()
		if err != nil {
			return
		}

		if fileHash == specHash {
			log.Printf("Hash match, file: %s, hash: %s", fp, fileHash)
			return
		}

		log.Printf("Hash mismatch, expected: %s, got: %s.", specHash, fileHash)
	}

	fileHash, err = fetch.Download(context.Background(), url, fp)
	if err != nil {
		log.Printf("File %s fetch failed, err: %v", url, err)
		return
	}

	if fileHash == specHash {
		log.Printf("Hash match, file: %s, hash: %s", fp, fileHash)
		return true, nil
	}
	log.Printf("Hash mismatch, expected: %s, got: %s. Fetch distro %s.", specHash, fileHash, url)

	return false, errors.New("osquery distro hash mismatch")
}

func extractOrCopy(osarch distro.OSArch, spec distro.Spec) error {
	dir := distro.GetDataInstallDir(osarch)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create dir %v, %w", dir, err)
	}

	src := spec.DistroFilepath(distro.DataCacheDir)

	// Include the official osquery msi installer for windows for now
	// until we figure out a better way to crack it open during the build
	if !spec.Extract {
		dst := spec.DistroFilepath(dir)
		log.Printf("Copy file %s to %s", src, dst)
		return devtools.Copy(src, dst)
	}

	// Extract osqueryd
	if strings.HasSuffix(src, ".tar.gz") {
		tmpdir, err := ioutil.TempDir(distro.DataDir, "")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmpdir)

		osdp := distro.OsquerydLinuxDistroPath()
		if err := tar.ExtractFile(src, tmpdir, osdp); err != nil {
			return err
		}

		return devtools.Copy(filepath.Join(tmpdir, osdp), distro.OsquerydPath(dir))
	}
	return fmt.Errorf("unsupported file: %s", src)
}
