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

package mage

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"
	"github.com/magefile/mage/sh"
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
func CopyNCAPInstaller() error {
	if devtools.Platform.GOOS == "windows" && (devtools.Platform.GOARCH == "amd64" || devtools.Platform.GOARCH == "386") {
		err := sh.Copy("./npcap/installer/"+installer, "/installer/"+installer)
		if err != nil {
			return fmt.Errorf("failed to copy Npcap installer into source tree: %w", err)
		}
	}
	return nil
}

// GetNpcapInstaller gets the installer from the Google Cloud Storage service.
//
// On Windows platforms, if getNpcapInstaller is invoked with the environment variables
// CI or NPCAP_LOCAL set to "true" and the OEM Npcap installer is not available it is
// obtained from the cloud storage. This behaviour requires access to the private store.
// If NPCAP_LOCAL is set to "true" and the file is in the npcap/installer directory, no
// fetch will be made.
func GetNpcapInstaller() error {
	// TODO: Consider whether to expose this as a target.
	if runtime.GOOS != "windows" {
		return nil
	}
	if os.Getenv("CI") != "true" && os.Getenv("NPCAP_LOCAL") != "true" {
		return errors.New("only available if running in the CI or with NPCAP_LOCAL=true")
	}
	dstPath := filepath.Join("./npcap/installer", installer)
	if os.Getenv("NPCAP_LOCAL") == "true" {
		fi, err := os.Stat(dstPath)
		if err == nil && !fi.IsDir() {
			fmt.Println("using local Npcap installer with NPCAP_LOCAL=true")
			return nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	ciBucketName := getBucketName()

	fmt.Printf("getting %s from private cache\n", installer)
	return sh.RunV("gsutil", "cp", "gs://"+ciBucketName+"/private/"+installer, dstPath)
}

func getBucketName() string {
	return "ingest-buildkite-ci"
}
