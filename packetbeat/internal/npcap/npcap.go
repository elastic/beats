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
//
// The npcap package interacts with a registry and download server that
// provides a current_version end point that serves a JSON message that
// corresponds to the this Go type:
//
//  struct {
//  	Version string // The semverish version of the Npcap installer.
//  	URL     string // The location of the Npcap installer.
//  	Hash    string // The sha256 hash of the Npcap installer.
//  }
//
// The URL field will point to the location of anb Npcap installer.
package npcap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/google/gopacket/pcap"
	"golang.org/x/mod/semver"

	"github.com/elastic/beats/v7/libbeat/logp"
)

// Registry is the location of current Npcap version information.
const Registry = "https://artifacts.elastic.co/downloads/npcap/current_version"

// Fetch downloads the Npcap installer, writes the content to the given filepath
// and returns the sha256 hash of the downloaded object.
func CurrentVersion(ctx context.Context, log *logp.Logger, registry string) (version, url, hash string, err error) {
	if runtime.GOOS != "windows" {
		return "", "", "", errors.New("npcap: called Fetch on non-Windows platform")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", registry, nil)
	if err != nil {
		return "", "", "", err
	}

	var client http.Client
	res, err := client.Do(req)
	if err != nil {
		return "", "", "", err
	}
	defer res.Body.Close()

	b, err := ioutil.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		if err != nil {
			log.Errorf("failed to read the error response body: %v", err)
		}
		b = bytes.TrimSpace(b)
		if len(b) == 0 {
			return "", "", "", fmt.Errorf("npcap: failed to fetch %s, status: %d, message: empty", url, res.StatusCode)
		}
		return "", "", "", fmt.Errorf("npcap: failed to fetch %s, status: %d, message: %s", url, res.StatusCode, b)
	}

	var info struct {
		Version string
		URL     string
		Hash    string
	}
	err = json.Unmarshal(b, &info)
	if err != nil {
		return "", "", "", err
	}

	return info.Version, info.URL, info.Hash, nil
}

// Fetch downloads the Npcap installer, writes the content to the given filepath
// and returns the sha256 hash of the downloaded object.
func Fetch(ctx context.Context, log *logp.Logger, url, path string) (hash string, err error) {
	if runtime.GOOS != "windows" {
		return "", errors.New("npcap: called Fetch on non-Windows platform")
	}

	log.Infof("download %s to %s", url, path)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	var client http.Client
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Errorf("failed to read the error response body: %v", err)
		}
		b = bytes.TrimSpace(b)
		if len(b) == 0 {
			return "", fmt.Errorf("npcap: failed to fetch %s, status: %d, message: empty", url, res.StatusCode)
		}
		return "", fmt.Errorf("npcap: failed to fetch %s, status: %d, message: %s", url, res.StatusCode, b)
	}

	dst, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	h := sha256.New()
	_, err = io.Copy(io.MultiWriter(h, dst), res.Body)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
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

func Upgradeable(version string) bool {
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
	return semver.Compare("v"+installed, "v"+version) < 0
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
