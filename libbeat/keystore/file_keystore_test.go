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

package keystore

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/docker/docker/pkg/ioutils"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v8/libbeat/common"
)

var keyValue = "output.elasticsearch.password"
var secretValue = []byte("secret")

func TestCanCreateAKeyStore(t *testing.T) {
	path := GetTemporaryKeystoreFile()
	defer os.Remove(path)

	keyStore, err := NewFileKeystore(path)
	assert.NoError(t, err)

	writableKeystore, err := AsWritableKeystore(keyStore)
	assert.NoError(t, err)

	assert.Nil(t, writableKeystore.Store(keyValue, secretValue))
	assert.Nil(t, writableKeystore.Save())
}

func TestCanReadAnExistingKeyStoreWithEmptyString(t *testing.T) {
	path := GetTemporaryKeystoreFile()
	defer os.Remove(path)

	CreateAnExistingKeystore(path)

	keystoreRead, err := NewFileKeystore(path)
	assert.NoError(t, err)

	secure, err := keystoreRead.Retrieve(keyValue)
	assert.NoError(t, err)

	v, err := secure.Get()
	assert.NoError(t, err)
	assert.Equal(t, v, secretValue)
}

func TestCanDeleteAKeyFromTheStoreAndPersistChanges(t *testing.T) {
	path := GetTemporaryKeystoreFile()
	defer os.Remove(path)

	CreateAnExistingKeystore(path)

	keyStore, _ := NewFileKeystore(path)
	_, err := keyStore.Retrieve(keyValue)
	assert.NoError(t, err)

	writableKeystore, err := AsWritableKeystore(keyStore)
	assert.NoError(t, err)

	writableKeystore.Delete(keyValue)
	_, err = keyStore.Retrieve(keyValue)
	assert.Error(t, err)

	_ = writableKeystore.Save()
	newKeystore, err := NewFileKeystore(path)
	_, err = newKeystore.Retrieve(keyValue)
	assert.Error(t, err)
}

func TestFilePermissionOnCreate(t *testing.T) {
	// Skip check on windows
	if runtime.GOOS == "windows" {
		t.Skip("Permission check is not running on windows")
	}
	if !common.IsStrictPerms() {
		t.Skip("Skipping test because strict.perms is disabled")
	}

	path := GetTemporaryKeystoreFile()
	defer os.Remove(path)
	CreateAnExistingKeystore(path)

	stats, err := os.Stat(path)
	assert.NoError(t, err)
	permissions := stats.Mode().Perm()
	if permissions != 0600 {
		t.Fatalf("Expecting the file what only readable/writable by the owner, permission found: %v", permissions)
	}
}

func TestFilePermissionOnUpdate(t *testing.T) {
	// Skip check on windows
	if runtime.GOOS == "windows" {
		t.Skip("Permission check is not running on windows")
	}
	if !common.IsStrictPerms() {
		t.Skip("Skipping test because strict.perms is disabled")
	}

	path := GetTemporaryKeystoreFile()
	defer os.Remove(path)

	keyStore := CreateAnExistingKeystore(path)

	writableKeystore, err := AsWritableKeystore(keyStore)
	assert.NoError(t, err)

	err = writableKeystore.Store("newkey", []byte("newsecret"))
	assert.NoError(t, err)
	err = writableKeystore.Save()
	assert.NoError(t, err)
	stats, err := os.Stat(path)
	assert.NoError(t, err)
	permissions := stats.Mode().Perm()
	if permissions != 0600 {
		t.Fatalf("Expecting the file what only readable/writable by the owner, permission found: %v", permissions)
	}
}

func TestFilePermissionOnLoadWhenStrictIsOn(t *testing.T) {
	// Skip check on windows
	if runtime.GOOS == "windows" {
		t.Skip("Permission check is not running on windows")
	}

	if !common.IsStrictPerms() {
		t.Skip("Skipping test because strict.perms is disabled")
	}

	path := GetTemporaryKeystoreFile()
	defer os.Remove(path)

	// Create a world readable keystore file
	fd, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	assert.NoError(t, err)
	fd.WriteString("bad permission")
	assert.NoError(t, fd.Close())
	_, err = NewFileKeystore(path)
	assert.Error(t, err)
}

func TestReturnsUsedKeysInTheStore(t *testing.T) {
	path := GetTemporaryKeystoreFile()
	defer os.Remove(path)

	keyStore := CreateAnExistingKeystore(path)

	listingKeystore, err := AsListingKeystore(keyStore)
	assert.NoError(t, err)

	keys, err := listingKeystore.List()

	assert.NoError(t, err)
	assert.Equal(t, len(keys), 1)
	assert.Equal(t, keys[0], keyValue)
}

func TestCannotDecryptKeyStoreWithWrongPassword(t *testing.T) {
	path := GetTemporaryKeystoreFile()
	defer os.Remove(path)

	keyStore, err := NewFileKeystoreWithPassword(path, NewSecureString([]byte("password")))

	writableKeystore, err := AsWritableKeystore(keyStore)
	assert.NoError(t, err)

	writableKeystore.Store("hello", []byte("world"))
	writableKeystore.Save()

	_, err = NewFileKeystoreWithPassword(path, NewSecureString([]byte("wrongpassword")))
	if assert.Error(t, err, "should fail to decrypt the keystore") {
		m := `could not decrypt the keystore: could not decrypt keystore data: ` +
			`cipher: message authentication failed`
		assert.Equal(t, err, fmt.Errorf(m))
	}
}

func TestUserDefinedPasswordUTF8(t *testing.T) {
	createAndReadKeystoreWithPassword(t, []byte("mysecret¥¥password"))
}

func TestUserDefinedPasswordASCII(t *testing.T) {
	createAndReadKeystoreWithPassword(t, []byte("mysecret"))
}

func TestSecretWithUTF8EncodedSecret(t *testing.T) {
	content := []byte("ありがとうございます") // translation: thank you
	createAndReadKeystoreSecret(t, []byte("mysuperpassword"), "mykey", content)
}

func TestSecretWithASCIIEncodedSecret(t *testing.T) {
	content := []byte("good news everyone") // translation: thank you
	createAndReadKeystoreSecret(t, []byte("mysuperpassword"), "mykey", content)
}

func TestGetConfig(t *testing.T) {
	path := GetTemporaryKeystoreFile()
	defer os.Remove(path)

	keyStore := CreateAnExistingKeystore(path)

	writableKeystore, err := AsWritableKeystore(keyStore)
	assert.NoError(t, err)

	// Add a bit more data of different type
	writableKeystore.Store("super.nested", []byte("hello"))
	writableKeystore.Save()

	cfg, err := keyStore.GetConfig()
	assert.NotNil(t, cfg)
	assert.NoError(t, err)

	secret, err := cfg.String("output.elasticsearch.password", 0)
	assert.NoError(t, err)
	assert.Equal(t, secret, "secret")

	port, err := cfg.String("super.nested", 0)
	assert.Equal(t, port, "hello")
}

func TestShouldRaiseAndErrorWhenVersionDontMatch(t *testing.T) {
	temporaryPath := GetTemporaryKeystoreFile()
	defer os.Remove(temporaryPath)

	badVersion := `v2D/EQwnDNO7yZsjsRFVWGgbkZudhPxVhBkaQAVud66+tK4HRdfPrNrNNgSmhioDGrQ0z/VZpvbw68gb0G
	G2QHxlP5s4HGRU/GQge3Nsnx0+kDIcb/37gPN1D1TOPHSiRrzzPn2vInmgaLUfEgBgoa9tuXLZEKdh3JPh/q`

	f, err := os.OpenFile(temporaryPath, os.O_CREATE|os.O_WRONLY, 0600)
	assert.NoError(t, err)
	f.WriteString(badVersion)
	err = f.Close()
	assert.NoError(t, err)

	_, err = NewFileKeystoreWithPassword(temporaryPath, NewSecureString([]byte("")))
	if assert.Error(t, err, "Expect version check error") {
		assert.Equal(t, err, fmt.Errorf("keystore format doesn't match expected version: 'v1' got 'v2'"))
	}
}

func TestMissingEncryptedBlock(t *testing.T) {
	temporaryPath := GetTemporaryKeystoreFile()
	defer os.Remove(temporaryPath)

	badVersion := "v1"

	f, err := os.OpenFile(temporaryPath, os.O_CREATE|os.O_WRONLY, 0600)
	assert.NoError(t, err)
	f.WriteString(badVersion)
	err = f.Close()
	assert.NoError(t, err)

	_, err = NewFileKeystoreWithPassword(temporaryPath, NewSecureString([]byte("")))
	if assert.Error(t, err) {
		assert.Equal(t, err, fmt.Errorf("corrupt or empty keystore"))
	}
}

func createAndReadKeystoreSecret(t *testing.T, password []byte, key string, value []byte) {
	path := GetTemporaryKeystoreFile()
	defer os.Remove(path)

	keyStore, err := NewFileKeystoreWithPassword(path, NewSecureString(password))
	assert.NoError(t, err)

	writableKeystore, err := AsWritableKeystore(keyStore)
	assert.NoError(t, err)

	writableKeystore.Store(key, value)
	writableKeystore.Save()

	newStore, err := NewFileKeystoreWithPassword(path, NewSecureString(password))
	s, _ := newStore.Retrieve(key)
	v, _ := s.Get()
	assert.Equal(t, v, value)
}

func createAndReadKeystoreWithPassword(t *testing.T, password []byte) {
	path := GetTemporaryKeystoreFile()
	defer os.Remove(path)

	keyStore, err := NewFileKeystoreWithPassword(path, NewSecureString(password))
	assert.NoError(t, err)

	writableKeystore, err := AsWritableKeystore(keyStore)
	assert.NoError(t, err)

	writableKeystore.Store("hello", []byte("world"))
	writableKeystore.Save()

	newStore, err := NewFileKeystoreWithPassword(path, NewSecureString(password))
	s, _ := newStore.Retrieve("hello")
	v, _ := s.Get()

	assert.Equal(t, v, []byte("world"))
}

// CreateAnExistingKeystore creates a keystore with an existing key
/// `output.elasticsearch.password` with the value `secret`.
func CreateAnExistingKeystore(path string) Keystore {
	keyStore, err := NewFileKeystore(path)
	// Fail fast in the test suite
	if err != nil {
		panic(err)
	}

	writableKeystore, err := AsWritableKeystore(keyStore)
	if err != nil {
		panic(err)
	}

	writableKeystore.Store(keyValue, secretValue)
	writableKeystore.Save()
	return keyStore
}

// GetTemporaryKeystoreFile create a temporary file on disk to save the keystore.
func GetTemporaryKeystoreFile() string {
	path, err := ioutils.TempDir("", "testing")
	if err != nil {
		panic(err)
	}
	return filepath.Join(path, "keystore")
}
