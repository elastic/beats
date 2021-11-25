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
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/packetbeat/internal/npcap"
)

func installNpcap(cfg *common.Config) error {
	if runtime.GOOS != "windows" {
		return nil
	}

	reinstall, err := configBool(cfg, "npcap.force_reinstall")
	if err != nil {
		return err
	}
	timeout, err := configDuration(cfg, "npcap.install_timeout")
	if err != nil {
		return err
	}
	rawURI, err := configString(cfg, "npcap.installer_location")
	if err != nil {
		return err
	}
	installDst, err := configString(cfg, "npcap.install_destination")
	if err != nil {
		return err
	}
	if rawURI == "" {
		rawURI = npcap.Registry
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	log := logp.NewLogger("npcap_install")

	uri, err := url.Parse(rawURI)
	if err != nil {
		return err
	}

	// If a file or bare path is specified, go ahead and install it.
	if uri.Scheme == "" || uri.Scheme == "file" {
		return npcap.Install(ctx, log, uri.Path, installDst, false)
	}

	canFail, err := configBool(cfg, "npcap.ignore_misssing_registry")
	if err != nil {
		return err
	}
	version, download, wantHash, err := npcap.CurrentVersion(ctx, log, rawURI)
	if err != nil {
		if canFail && errors.Is(err, npcap.RegistryNotFound) {
			log.Warnf("%v: did not install Npcap", err)
			return nil
		}
		return err
	}

	// Is a more recent version available or are we forcing install.
	if !npcap.Upgradeable(version) && !reinstall {
		return nil
	}

	retain, err := configBool(cfg, "npcap.retain_download")
	if err != nil {
		return err
	}

	dir, err := os.MkdirTemp("", "packetbeat-npcap-*")
	if err != nil {
		return err
	}
	if retain {
		log.Infof("working in %s", dir)
	} else {
		defer os.RemoveAll(dir)
	}
	pth := filepath.Join(dir, path.Base(download))

	gotHash, err := npcap.Fetch(ctx, log, download, pth)
	if err != nil {
		return err
	}

	if gotHash != wantHash {
		return fmt.Errorf("npcap: hash mismatch for %s: want:%s got:%s", download, wantHash, gotHash)
	}

	return npcap.Install(ctx, log, pth, installDst, false)
}

func configBool(cfg *common.Config, path string) (bool, error) {
	ok, err := cfg.Has(path, -1)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	v, err := cfg.Bool(path, -1)
	if err != nil {
		return false, err
	}
	return v, nil
}

func configString(cfg *common.Config, path string) (string, error) {
	ok, err := cfg.Has(path, -1)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", nil
	}
	v, err := cfg.String(path, -1)
	if err != nil {
		return "", err
	}
	return v, nil
}

func configDuration(cfg *common.Config, path string) (time.Duration, error) {
	const defaultTimeout = 120 * time.Second

	v, err := configString(cfg, path)
	if err != nil {
		return 0, err
	}
	if v == "" {
		return defaultTimeout, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, err
	}
	return d, nil
}
