// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mage

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/magefile/mage/sh"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"
)

// NpcapVersion specifies the version of the OEM Npcap installer to bundle with
// the packetbeat executable. It is used to specify which npcap builder crossbuild
// image to use and the installer to obtain from the cloud store for testing.
const (
	NpcapVersion = "1.80"
	installer    = "npcap-" + NpcapVersion + "-oem.exe"
)

func ImageSelector(platform string) (string, error) {
	image, err := devtools.CrossBuildImage(platform)
	if err != nil {
		return "", err
	}
	if os.Getenv("CI") != "true" && os.Getenv("NPCAP_LOCAL") != "true" {
		return image, nil
	}
	if platform == "windows/amd64" {
		image = strings.ReplaceAll(image, "beats-dev", "observability-ci") // Temporarily work around naming of npcap image.
		image = strings.ReplaceAll(image, "main", "npcap-"+NpcapVersion+"-debian9")
	}
	return image, nil
}

func CopyNPCAPInstaller(dir string) error {
	if devtools.Platform.GOOS == "windows" && (devtools.Platform.GOARCH == "amd64" || devtools.Platform.GOARCH == "386") {
		err := sh.Copy(dir+installer, "/installer/"+installer)
		if err != nil {
			return fmt.Errorf("failed to copy Npcap installer into source tree: %w", err)
		}
	}
	return nil
}

// GetNpcapInstallerFn gets function that gets the installer from the Google Cloud Storage service.
//
// On Windows platforms, if getNpcapInstaller is invoked with the environment variables
// CI or NPCAP_LOCAL set to "true" and the OEM Npcap installer is not available it is
// obtained from the cloud storage. This behaviour requires access to the private store.
// If NPCAP_LOCAL is set to "true" and the file is in the npcap/installer directory, no
// fetch will be made.
func GetNpcapInstallerFn(dir string) func() error {
	if dir == "" {
		dir = "./"
	}
	return func() error {
		// TODO: Consider whether to expose this as a target.
		if runtime.GOOS != "windows" {
			return nil
		}
		if os.Getenv("CI") != "true" && os.Getenv("NPCAP_LOCAL") != "true" {
			return errors.New("only available if running in the CI or with NPCAP_LOCAL=true")
		}
		dstPath := filepath.Join(dir, "npcap/installer", installer)
		if os.Getenv("NPCAP_LOCAL") == "true" {
			fi, err := os.Stat(dstPath)
			if err == nil && !fi.IsDir() {
				fmt.Println("using local Npcap installer with NPCAP_LOCAL=true") //nolint:forbidigo // fmt.Println is ok here
				return nil
			}
			if !errors.Is(err, fs.ErrNotExist) {
				return err
			}
		}
		ciBucketName := getBucketName()

		fmt.Printf("getting %s from private cache to %q\n", installer, dstPath) //nolint:forbidigo // fmt.Println is ok here
		return sh.RunV("gsutil", "cp", "gs://"+ciBucketName+"/private/"+installer, dstPath)
	}
}

func getBucketName() string {
	return "ingest-buildkite-ci"
}
