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

// Package npcap handles fetching and installing Npcap fow Windows.
package npcap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/google/gopacket/pcap"
	"golang.org/x/mod/semver"

	"github.com/elastic/beats/v7/libbeat/logp"
)

// TODO: Currently the latest version is statically defined. When we have
// a location to serve from, we can make this dynamically defined with a
// function find the URL to the latest version and update Npcap for users
// without updating the beat version.

const (
	// CurrentVersion is the current version to install.
	CurrentVersion = "1.55"

	// CurrentInstaller is the executable name of the current versions installer.
	CurrentInstaller = "npcap-" + CurrentVersion + "-oem.exe"

	//  InstallerURL is the URL for the current Npcap version installer.
	InstallerURL = "https://artifacts.elastic.co/downloads/npcap/" + CurrentInstaller // FIXME: This is a placeholder.
)

func init() {
	// This is included to ensure that if NMAP.org change their versioning
	// approach we get a signal to change our ordering implementation to match.
	if !semver.IsValid("v" + CurrentVersion) {
		panic(fmt.Sprintf("npcap: invalid version for semver: %s", CurrentVersion))
	}
}

// Fetch downloads the Npcap installer, writes the content to the given filepath
// and returns the sha256 hash of the downloaded object.
func Fetch(ctx context.Context, log *logp.Logger, url, path string) (hash []byte, err error) {
	if runtime.GOOS != "windows" {
		return nil, errors.New("npcap: called Fetch on non-Windows platform")
	}

	log.Infof("download %s to %s", url, path)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return
	}

	var client http.Client
	res, err := client.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Errorf("failed to read the error response body: %v", err)
		}
		b = bytes.TrimSpace(b)
		if len(b) == 0 {
			return nil, fmt.Errorf("npcap: failed to fetch %s, status: %d, message: empty", url, res.StatusCode)
		}
		return nil, fmt.Errorf("npcap: failed to fetch %s, status: %d, message: %s", url, res.StatusCode, b)
	}

	dst, err := os.Create(path)
	if err != nil {
		return
	}
	defer dst.Close()

	h := sha256.New()
	_, err = io.Copy(io.MultiWriter(h, dst), res.Body)
	if err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

// Verify compares the provided hash against the expected hash for the
// installer at the given path.
func Verify(path string, hash []byte) error {
	base := filepath.Base(path)
	h := hex.EncodeToString(hash)
	want, ok := hashes[base]
	if !ok {
		return fmt.Errorf("npcap: unknown Npcap installer version: %s", base)
	}
	if want != h {
		return fmt.Errorf("npcap: hash mismatch for %s: want:%s got:%s", path, want, h)
	}
	return nil
}

// hashes is the mapping of Npcap installer versions to their sha256 hash.
var hashes = map[string]string{
	"npcap-1.55-oem.exe": "1f035c0498863b41b64df87099ec20f80c6db26b12d27b5afef1c1ad3fa28690",
}

// Install runs the Npcap installer at the provided path.
func Install(ctx context.Context, log *logp.Logger, path string, compat bool) error {
	if runtime.GOOS != "windows" {
		return errors.New("npcap: called Install on non-Windows platform")
	}

	args := []string{"/S", "/winpcap_mode=no"}
	if compat {
		args[1] = "/winpcap_mode=yes"
	}
	cmd := exec.CommandContext(ctx, path, args...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("npcap: failed to start Npcap installer: %w", err)
	}

	err = cmd.Wait()
	if outBuf.Len() != 0 {
		log.Info(&outBuf)
	}
	if err != nil {
		log.Error(&errBuf)
		return fmt.Errorf("npcap: failed to install Npcap: %w", err)
	}

	// gopacket/pcap does not provide a mechanism to reload the pcap DLL
	// so if we are upgrading we wait for the next startup of packetbeat.
	// Otherwise we can make sure that the DLL is loaded by calling
	// pcap.LoadWinPCAP. pcap.LoadWinPCAP is called on pcap package
	// initialization and if successful, subsequent calls are no-op, but
	// if Npcap/WinPCAP was not installed, it will have failed and can be
	// called now. So this is safe in all cases.
	err = loadWinPCAP()

	return err
}

func Upgradeable() bool {
	// pcap.Version() returns a string in the form:
	//
	//  Npcap version 1.55, based on libpcap version 1.10.2-PRE-GIT
	//
	// if an Npcap version is installed. See https://nmap.org/npcap/guide/npcap-devguide.html#npcap-detect
	installed := pcap.Version()
	if !strings.HasPrefix(installed, "Npcap version") {
		return true
	}
	installed = strings.TrimPrefix(installed, "Npcap version ")
	idx := strings.Index(installed, ",")
	if idx < 0 {
		return true
	}
	installed = installed[:idx]
	return semver.Compare("v"+installed, "v"+CurrentVersion) < 0
}

// Uninstall uninstalls the Npcap tools.
func Uninstall(ctx context.Context, log *logp.Logger) error {
	if runtime.GOOS != "windows" {
		return errors.New("npcap: called Uninstall on non-Windows platform")
	}
	if pcap.Version() == "" {
		return nil
	}

	const uninstaller = `C:\Program Files\Npcap\Uninstall.exe`
	cmd := exec.CommandContext(ctx, uninstaller, `/S`)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("npcap: failed to start Npcap uninstaller: %w", err)
	}

	err = cmd.Wait()
	if outBuf.Len() != 0 {
		log.Info(&outBuf)
	}
	if err != nil {
		log.Error(&errBuf)
		return fmt.Errorf("npcap: failed to uninstall Npcap: %w", err)
	}
	return nil
}
