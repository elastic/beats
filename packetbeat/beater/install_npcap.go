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
	"net/url"
	"os"
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
	if !npcap.Upgradeable() && !reinstall {
		return nil
	}

	timeout, err := configDuration(cfg, "npcap.install_timeout")
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	rawURI, err := configString(cfg, "npcap.installer_location")
	if err != nil {
		return err
	}
	if rawURI == "" {
		rawURI = npcap.InstallerURL
	}
	uri, err := url.Parse(rawURI)
	if err != nil {
		return err
	}

	log := logp.NewLogger("npcap_install")

	if uri.Scheme == "" || uri.Scheme == "file" {
		return npcap.Install(ctx, log, uri.Path, false)
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
	path := filepath.Join(dir, npcap.CurrentVersion)

	h, err := npcap.Fetch(ctx, log, uri.String(), path)
	if err != nil {
		return err
	}
	err = npcap.Verify(path, h)
	if err != nil {
		return err
	}
	return npcap.Install(ctx, log, path, false)
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
