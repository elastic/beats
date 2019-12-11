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

package registrar

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	helper "github.com/elastic/beats/libbeat/common/file"
	"github.com/elastic/beats/libbeat/logp"
)

const (
	legacyVersion  = "<legacy>"
	currentVersion = "0"
)

func ensureCurrent(home, migrateFile string, perm os.FileMode) error {
	if migrateFile == "" {
		if isFile(home) {
			migrateFile = home
		}
	}

	fbRegHome := filepath.Join(home, "filebeat")
	version, err := readVersion(fbRegHome, migrateFile)
	if err != nil {
		return err
	}

	logp.Debug("registrar", "Registry type '%v' found", version)

	switch version {
	case legacyVersion:
		return migrateLegacy(home, fbRegHome, migrateFile, perm)
	case currentVersion:
		return nil
	case "":
		backupFile := migrateFile + ".bak"
		if isFile(backupFile) {
			return migrateLegacy(home, fbRegHome, backupFile, perm)
		}
		return initRegistry(fbRegHome, perm)
	default:
		return fmt.Errorf("registry file version %v not supported", version)
	}
}

func migrateLegacy(home, regHome, migrateFile string, perm os.FileMode) error {
	logp.Info("Migrate registry file to registry directory")

	if home == migrateFile {
		backupFile := migrateFile + ".bak"
		if isFile(migrateFile) {
			logp.Info("Move registry file to backup file: %v", backupFile)
			if err := helper.SafeFileRotate(backupFile, migrateFile); err != nil {
				return err
			}
			migrateFile = backupFile
		} else if isFile(backupFile) {
			logp.Info("Old registry backup file found, continue migration")
			migrateFile = backupFile
		}
	}

	if err := initRegistry(regHome, perm); err != nil {
		return err
	}

	dataFile := filepath.Join(regHome, "data.json")
	if !isFile(dataFile) && isFile(migrateFile) {
		logp.Info("Migrate old registry file to new data file")
		err := helper.SafeFileRotate(dataFile, migrateFile)
		if err != nil {
			return err
		}
	}

	return nil
}

func initRegistry(regHome string, perm os.FileMode) error {
	if !isDir(regHome) {
		logp.Info("No registry home found. Create: %v", regHome)
		if err := os.MkdirAll(regHome, 0750); err != nil {
			return errors.Wrapf(err, "failed to create registry dir '%v'", regHome)
		}
	}

	metaFile := filepath.Join(regHome, "meta.json")
	if !isFile(metaFile) {
		logp.Info("Initialize registry meta file")
		err := safeWriteFile(metaFile, []byte(`{"version": "0"}`), perm)
		if err != nil {
			return errors.Wrap(err, "failed writing registry meta.json")
		}
	}

	return nil
}

func readVersion(regHome, migrateFile string) (string, error) {
	if isFile(migrateFile) {
		return legacyVersion, nil
	}

	if !isDir(regHome) {
		return "", nil
	}

	metaFile := filepath.Join(regHome, "meta.json")
	if !isFile(metaFile) {
		return "", nil
	}

	tmp, err := ioutil.ReadFile(metaFile)
	if err != nil {
		return "", errors.Wrap(err, "failed to open meta file")
	}

	meta := struct{ Version string }{}
	if err := json.Unmarshal(tmp, &meta); err != nil {
		return "", errors.Wrap(err, "failed reading meta file")
	}

	return meta.Version, nil
}

func isDir(path string) bool {
	fi, err := os.Stat(path)
	exists := err == nil && fi.IsDir()
	logp.Debug("test", "isDir(%v) -> %v", path, exists)
	return exists
}

func isFile(path string) bool {
	fi, err := os.Stat(path)
	exists := err == nil && fi.Mode().IsRegular()
	logp.Debug("test", "isFile(%v) -> %v", path, exists)
	return exists
}

func safeWriteFile(path string, data []byte, perm os.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}

	for len(data) > 0 {
		var n int
		n, err = f.Write(data)
		if err != nil {
			break
		}

		data = data[n:]
	}

	if err == nil {
		err = f.Sync()
	}

	if err1 := f.Close(); err == nil {
		err = err1
	}
	return err
}
