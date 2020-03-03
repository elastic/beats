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

	"github.com/elastic/beats/v7/filebeat/input/file"
	helper "github.com/elastic/beats/v7/libbeat/common/file"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/registry"
	"github.com/elastic/beats/v7/libbeat/registry/backend/memlog"
)

type registryVersion string

const (
	noRegistry    registryVersion = ""
	legacyVersion                 = "<legacy>"
	version0                      = "0"
	version1                      = "1"
)

const currentVersion = version1

// ensureCurrent migrates old registry versions to the most recent version.
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

	for {
		switch version {
		case legacyVersion:
			// first migrate to verion0 style registry and go from there
			if err := migrateLegacy(home, fbRegHome, migrateFile, perm); err != nil {
				return err
			}
			fallthrough
		case version0:
			return migrateVersion1(home, fbRegHome, perm)

		case currentVersion:
			return nil
		case noRegistry:
			// check if we've been in the middle of a migration from the legacy
			// format to the current version. If so continue with
			// the migration and try again.
			backupFile := migrateFile + ".bak"
			if isFile(backupFile) {
				migrateFile = backupFile
				version = legacyVersion
				break
			}

			// postpone registry creation, until we open and configure it.
			return nil
		default:
			return fmt.Errorf("registry file version %v not supported", version)
		}
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

	if err := initVersion0Registry(regHome, perm); err != nil {
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

func initVersion0Registry(regHome string, perm os.FileMode) error {
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

// migrateVersion1 migrates the filebeat registry from version 0 to version 1
// only. Version 1 is based on the implementation of version 1 in
// libbeat/registry/backend/memlog.
// The migration itself only needs to rename the data file and update
// meta.json.  During migration we take advantage of the fact that the memlog
// snapshot format is backwards compatible to the old filebeat registry file,
// and that timestamp decoding has support for different formats.
func migrateVersion1(home, regHome string, perm os.FileMode) error {
	logp.Info("Migrate registry version 0 to version 1")

	origDataFile := filepath.Join(regHome, "data.json")
	if !isFile(origDataFile) {
		return fmt.Errorf("missing original data file at: %v", origDataFile)
	}

	// read states from file and ensure file is closed immediately.
	states, err := func() ([]file.State, error) {
		origIn, err := os.Open(origDataFile)
		if err != nil {
			return nil, errors.Wrap(err, "failed to open original data file")
		}
		defer origIn.Close()

		var states []file.State
		decoder := json.NewDecoder(origIn)
		if err := decoder.Decode(&states); err != nil {
			return nil, errors.Wrapf(err, "Error decoding original data file '%v'", origDataFile)
		}
		return states, nil
	}()
	if err != nil {
		return err
	}

	registryBackend, err := memlog.New(memlog.Settings{
		Root:       home,
		FileMode:   perm,
		Checkpoint: func(_ uint, _ uint) bool { return true },
	})
	if err != nil {
		return errors.Wrap(err, "failed to create new registry backend")
	}

	reg := registry.New(registryBackend)
	defer reg.Close()

	store, err := reg.Get("filebeat")
	if err != nil {
		return errors.Wrap(err, "failed to open filebeat registry store")
	}
	defer store.Close()

	err = store.Update(func(tx *registry.Tx) error {
		return writeStateUpdates(tx, states)
	})
	if err != nil {
		return errors.Wrap(err, "failed to migrate registry states")
	}

	if err := os.Remove(origDataFile); err != nil {
		return errors.Wrapf(err, "migration complete but failed to remove original data file: %v", origDataFile)
	}

	return nil
}

func writeMeta(path string, version string, perm os.FileMode) error {
	logp.Info("Write registry meta file with version: %v", version)
	doc := struct{ Version string }{version}
	body, err := json.Marshal(doc)
	if err != nil {
		panic(err) // must not fail
	}

	if err = safeWriteFile(path+".tmp", body, perm); err != nil {
		return errors.Wrap(err, "failed writing registry meta.json")
	}

	return helper.SafeFileRotate(path, path+".tmp")
}

func readVersion(regHome, migrateFile string) (registryVersion, error) {
	if isFile(migrateFile) {
		return legacyVersion, nil
	}

	if !isDir(regHome) {
		return noRegistry, nil
	}

	metaFile := filepath.Join(regHome, "meta.json")
	if !isFile(metaFile) {
		return noRegistry, nil
	}

	tmp, err := ioutil.ReadFile(metaFile)
	if err != nil {
		return noRegistry, errors.Wrap(err, "failed to open meta file")
	}

	meta := struct{ Version string }{}
	if err := json.Unmarshal(tmp, &meta); err != nil {
		return noRegistry, errors.Wrap(err, "failed reading meta file")
	}

	return registryVersion(meta.Version), nil
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
