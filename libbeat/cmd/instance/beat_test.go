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

//go:build !integration
// +build !integration

package instance

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/menderesk/beats/v7/libbeat/cfgfile"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewInstance(t *testing.T) {
	b, err := NewBeat("testbeat", "testidx", "0.9", false)
	if err != nil {
		panic(err)
	}

	assert.Equal(t, "testbeat", b.Info.Beat)
	assert.Equal(t, "testidx", b.Info.IndexPrefix)
	assert.Equal(t, "0.9", b.Info.Version)

	// UUID4 should be 36 chars long
	assert.Equal(t, 16, len(b.Info.ID))
	assert.Equal(t, 36, len(b.Info.ID.String()))

	// indexPrefix set to name if empty
	b, err = NewBeat("testbeat", "", "0.9", false)
	if err != nil {
		panic(err)
	}
	assert.Equal(t, "testbeat", b.Info.Beat)
	assert.Equal(t, "testbeat", b.Info.IndexPrefix)

}

func TestNewInstanceUUID(t *testing.T) {
	b, err := NewBeat("testbeat", "", "0.9", false)
	if err != nil {
		panic(err)
	}

	// Make sure the ID's are different
	differentUUID, err := uuid.NewV4()
	if err != nil {
		t.Fatalf("error while generating ID: %v", err)
	}
	assert.NotEqual(t, b.Info.ID, differentUUID)
}

func TestInitKibanaConfig(t *testing.T) {
	b, err := NewBeat("filebeat", "testidx", "0.9", false)
	if err != nil {
		panic(err)
	}

	assert.Equal(t, "filebeat", b.Info.Beat)
	assert.Equal(t, "testidx", b.Info.IndexPrefix)
	assert.Equal(t, "0.9", b.Info.Version)

	const configPath = "../test/filebeat_test.yml"

	// Ensure that the config has owner-exclusive write permissions.
	// This is necessary on some systems which have a default umask
	// of 0o002, meaning that files are checked out by git with mode
	// 0o664. This would cause cfgfile.Load to fail.
	err = os.Chmod(configPath, 0o644)
	assert.NoError(t, err)

	cfg, err := cfgfile.Load(configPath, nil)
	assert.NoError(t, err)
	err = cfg.Unpack(&b.Config)
	assert.NoError(t, err)

	kibanaConfig := InitKibanaConfig(b.Config)
	username, err := kibanaConfig.String("username", -1)
	assert.NoError(t, err)
	password, err := kibanaConfig.String("password", -1)
	assert.NoError(t, err)
	api_key, err := kibanaConfig.String("api_key", -1)
	assert.NoError(t, err)
	protocol, err := kibanaConfig.String("protocol", -1)
	assert.NoError(t, err)
	host, err := kibanaConfig.String("host", -1)
	assert.NoError(t, err)

	assert.Equal(t, "elastic-test-username", username)
	assert.Equal(t, "elastic-test-password", password)
	assert.Equal(t, "elastic-test-api-key", api_key)
	assert.Equal(t, "https", protocol)
	assert.Equal(t, "127.0.0.1:5601", host)
}

func TestEmptyMetaJson(t *testing.T) {
	b, err := NewBeat("filebeat", "testidx", "0.9", false)
	if err != nil {
		panic(err)
	}

	// prepare empty meta file
	metaFile, err := ioutil.TempFile("../test", "meta.json")
	assert.Equal(t, nil, err, "Unable to create temporary meta file")

	metaPath := metaFile.Name()
	metaFile.Close()
	defer os.Remove(metaPath)

	// load metadata
	err = b.loadMeta(metaPath)

	assert.Equal(t, nil, err, "Unable to load meta file properly")
	assert.NotEqual(t, uuid.Nil, b.Info.ID, "Beats UUID is not set")
}

func TestMetaJsonWithTimestamp(t *testing.T) {
	firstBeat, err := NewBeat("filebeat", "testidx", "0.9", false)
	if err != nil {
		panic(err)
	}
	firstStart := firstBeat.Info.FirstStart

	metaFile, err := ioutil.TempFile("../test", "meta.json")
	assert.Equal(t, nil, err, "Unable to create temporary meta file")

	metaPath := metaFile.Name()
	metaFile.Close()
	defer os.Remove(metaPath)

	err = firstBeat.loadMeta(metaPath)
	assert.Equal(t, nil, err, "Unable to load meta file properly")

	secondBeat, err := NewBeat("filebeat", "testidx", "0.9", false)
	if err != nil {
		panic(err)
	}
	assert.False(t, firstStart.Equal(secondBeat.Info.FirstStart), "Before meta.json is loaded, first start must be different")
	secondBeat.loadMeta(metaPath)

	assert.Equal(t, nil, err, "Unable to load meta file properly")
	assert.True(t, firstStart.Equal(secondBeat.Info.FirstStart), "Cannot load first start")
}
