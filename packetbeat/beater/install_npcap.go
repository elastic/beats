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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/google/gopacket/pcap"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/packetbeat/npcap"
)

type npcapConfig struct {
	NeverInstall       bool          `config:"npcap.never_install"`
	ForceReinstall     bool          `config:"npcap.force_reinstall"`
	InstallTimeout     time.Duration `config:"npcap.install_timeout"`
	InstallDestination string        `config:"npcal.install_destination"`
}

func (c *npcapConfig) Init() {
	// Set defaults.
	c.InstallTimeout = 120 * time.Second
}

func installNpcap(b *beat.Beat) error {
	if !b.Info.ElasticLicensed {
		return nil
	}
	if runtime.GOOS != "windows" {
		return nil
	}

	defer func() {
		log := logp.NewLogger("npcap")
		npcapVersion := pcap.Version()
		if npcapVersion == "" {
			log.Warn("no version available for npcap")
		} else {
			log.Infof("npcap version: %s", npcapVersion)
		}
	}()

	var cfg npcapConfig
	err := b.BeatConfig.Unpack(&cfg)
	if err != nil {
		return fmt.Errorf("failed to unpack npcap config: %w", err)
	}
	if cfg.NeverInstall {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.InstallTimeout)
	defer cancel()

	log := logp.NewLogger("npcap_install")

	if npcap.Installer == nil {
		return nil
	}
	if !cfg.ForceReinstall && !npcap.Upgradeable() {
		npcap.Installer = nil
		return nil
	}
	tmp, err := os.MkdirTemp("", "")
	if err != nil {
		return fmt.Errorf("could not create installation temporary directory: %w", err)
	}
	defer func() {
		// The init sequence duplicates the embedded binary.
		// Get rid of the part we can. The remainder is in
		// the packetbeat text section as a string.
		npcap.Installer = nil
		// Remove the installer from the file system.
		os.RemoveAll(tmp)
	}()
	installerPath := filepath.Join(tmp, "npcap.exe")
	err = os.WriteFile(installerPath, npcap.Installer, 0o700)
	if err != nil {
		return fmt.Errorf("could not create installation temporary file: %w", err)
	}
	return npcap.Install(ctx, log, installerPath, cfg.InstallDestination, false)
}
