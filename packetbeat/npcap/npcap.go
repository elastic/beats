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

// Package npcap handles fetching and installing Npcap for Windows.
package npcap

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/google/gopacket/pcap"
	"golang.org/x/mod/semver"

	"github.com/elastic/beats/v7/libbeat/logp"
)

var (
	// Installer holds the embedded installer when run with x-pack.
	Installer []byte

	// EmbeddedInstallerVersion holds the version of the embedded installer.
	EmbeddedInstallerVersion string
)

// Install runs the Npcap installer at the provided path. The install
// destination is specified by dst and installation using WinPcap
// API-compatible Mode is specifed by compat. If dst is the empty string
// the default install location is used.
//
// See https://nmap.org/npcap/guide/npcap-users-guide.html#npcap-installation-uninstall-options
// for details.
func Install(ctx context.Context, log *logp.Logger, path, dst string, compat bool) error {
	if runtime.GOOS != "windows" {
		return errors.New("npcap: called Install on non-Windows platform")
	}
	return install(ctx, log, path, dst, compat)
}

func install(ctx context.Context, log *logp.Logger, path, dst string, compat bool) error {
	if pcap.Version() != "" {
		// If we are here there is a runtime Npcap DLL loaded. We need to
		// unload this to prevent the application being killed during the
		// install.
		//
		// See https://npcap.com/guide/npcap-users-guide.html#npcap-installation-uninstall-options.
		err := unloadWinPCAP()
		if err != nil {
			return fmt.Errorf("npcap: failed to unload Npcap DLL: %w", err)
		}
	}

	args := []string{"/S", "/winpcap_mode=no"}
	if compat {
		args[1] = "/winpcap_mode=yes"
	}
	if dst != "" {
		// The destination switch must be last as it uses unquoted spaces.
		// See https://nmap.org/npcap/guide/npcap-users-guide.html#npcap-installation-uninstall-options.
		args = append(args, "/D="+dst)
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

	return loadWinPCAP()
}

func Upgradeable() bool {
	// This is only set when a real installer is placed in
	// x-pack/packetbeat/npcap/installer.
	if EmbeddedInstallerVersion == "" {
		return false
	}

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
	return semver.Compare("v"+installed, "v"+EmbeddedInstallerVersion) < 0
}

// Uninstall uninstalls the Npcap tools. The path to the uninstaller can
// be provided, otherwise the default install location in used.
//
// See https://nmap.org/npcap/guide/npcap-users-guide.html#npcap-installation-uninstall-options
// for details.
func Uninstall(ctx context.Context, log *logp.Logger, path string) error {
	if runtime.GOOS != "windows" {
		return errors.New("npcap: called Uninstall on non-Windows platform")
	}
	if pcap.Version() == "" {
		return nil
	}
	return uninstall(ctx, log, path)
}

func uninstall(ctx context.Context, log *logp.Logger, path string) error {
	const uninstaller = `C:\Program Files\Npcap\Uninstall.exe`
	if path == "" {
		path = uninstaller
	}
	cmd := exec.CommandContext(ctx, path, `/S`)
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
