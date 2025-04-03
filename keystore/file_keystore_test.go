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
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var keyValue = "output.elasticsearch.password"

// Include uppercase, lowercase, symbols, and whitespace in the password.
// Commas in particular have caused parsing issues before: https://github.com/elastic/beats/issues/29789
var secretValue = []byte(",s3cRet~`! @#$%^&*()_-+={[}]|\\:;\"'<,>.?/")

func TestCanCreateAKeyStore(t *testing.T) {
	path := GetTemporaryKeystoreFile(t)
	defer os.Remove(path)

	keyStore, err := NewFileKeystore(path)
	require.NoError(t, err)

	writableKeystore, err := AsWritableKeystore(keyStore)
	require.NoError(t, err)

	require.Nil(t, writableKeystore.Store(keyValue, secretValue))
	require.Nil(t, writableKeystore.Save())
}

func TestCanReadAnExistingKeyStoreWithEmptyString(t *testing.T) {
	path := GetTemporaryKeystoreFile(t)
	defer os.Remove(path)

	CreateAnExistingKeystore(path)

	keystoreRead, err := NewFileKeystore(path)
	require.NoError(t, err)

	secure, err := keystoreRead.Retrieve(keyValue)
	require.NoError(t, err)

	v, err := secure.Get()
	require.NoError(t, err)
	require.Equal(t, v, secretValue)
}

func TestCanDeleteAKeyFromTheStoreAndPersistChanges(t *testing.T) {
	path := GetTemporaryKeystoreFile(t)
	defer os.Remove(path)

	CreateAnExistingKeystore(path)

	keyStore, _ := NewFileKeystore(path)
	_, err := keyStore.Retrieve(keyValue)
	require.NoError(t, err)

	writableKeystore, err := AsWritableKeystore(keyStore)
	require.NoError(t, err)

	err = writableKeystore.Delete(keyValue)
	require.NoError(t, err)
	_, err = keyStore.Retrieve(keyValue)
	require.Error(t, err)

	_ = writableKeystore.Save()
	newKeystore, err := NewFileKeystore(path)
	require.NoError(t, err)
	_, err = newKeystore.Retrieve(keyValue)
	require.Error(t, err)
}

func TestFilePermissionOnCreate(t *testing.T) {
	// Skip check on windows
	if runtime.GOOS == "windows" {
		t.Skip("Permission check is not running on windows")
	}

	path := GetTemporaryKeystoreFile(t)
	defer os.Remove(path)
	CreateAnExistingKeystore(path)

	stats, err := os.Stat(path)
	require.NoError(t, err)
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

	path := GetTemporaryKeystoreFile(t)
	defer os.Remove(path)

	keyStore := CreateAnExistingKeystore(path)

	writableKeystore, err := AsWritableKeystore(keyStore)
	require.NoError(t, err)

	err = writableKeystore.Store("newkey", []byte("newsecret"))
	require.NoError(t, err)
	err = writableKeystore.Save()
	require.NoError(t, err)
	stats, err := os.Stat(path)
	require.NoError(t, err)
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

	path := GetTemporaryKeystoreFile(t)
	defer os.Remove(path)

	// Create a world readable keystore file
	fd, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	require.NoError(t, err)
	_, _ = fd.WriteString("bad permission")
	require.NoError(t, fd.Close())
	_, err = NewFileKeystore(path)
	require.Error(t, err)
}

func TestReturnsUsedKeysInTheStore(t *testing.T) {
	path := GetTemporaryKeystoreFile(t)
	defer os.Remove(path)

	keyStore := CreateAnExistingKeystore(path)

	listingKeystore, err := AsListingKeystore(keyStore)
	require.NoError(t, err)

	keys, err := listingKeystore.List()

	require.NoError(t, err)
	require.Equal(t, len(keys), 1)
	require.Equal(t, keys[0], keyValue)
}

func TestCannotDecryptKeyStoreWithWrongPassword(t *testing.T) {
	path := GetTemporaryKeystoreFile(t)
	defer os.Remove(path)

	keyStore, err := NewFileKeystoreWithPassword(path, NewSecureString([]byte("password")))
	require.NoError(t, err)

	writableKeystore, err := AsWritableKeystore(keyStore)
	require.NoError(t, err)

	err = writableKeystore.Store("hello", []byte("world"))
	require.NoError(t, err)
	err = writableKeystore.Save()
	require.NoError(t, err)

	_, err = NewFileKeystoreWithPassword(path, NewSecureString([]byte("wrongpassword")))
	if assert.Error(t, err, "should fail to decrypt the keystore") {
		m := `could not decrypt the keystore: could not decrypt keystore data: ` +
			`cipher: message authentication failed`
		assert.Equal(t, m, err.Error())
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
	path := GetTemporaryKeystoreFile(t)
	defer os.Remove(path)

	keyStore := CreateAnExistingKeystore(path)

	writableKeystore, err := AsWritableKeystore(keyStore)
	require.NoError(t, err)

	// Add a bit more data of different type
	err = writableKeystore.Store("super.nested", []byte("hello"))
	require.NoError(t, err)
	err = writableKeystore.Save()
	require.NoError(t, err)

	cfg, err := keyStore.GetConfig()
	require.NotNil(t, cfg)
	require.NoError(t, err)

	secret, err := cfg.String("output.elasticsearch.password", 0)
	require.NoError(t, err)
	require.Equal(t, string(secretValue), secret)

	port, err := cfg.String("super.nested", 0)
	require.NoError(t, err)
	require.Equal(t, port, "hello")
}

func TestMissingEncryptedBlock(t *testing.T) {
	temporaryPath := GetTemporaryKeystoreFile(t)
	defer os.Remove(temporaryPath)

	versionString := string(version)

	f, err := os.OpenFile(temporaryPath, os.O_CREATE|os.O_WRONLY, 0600)
	require.NoError(t, err)
	_, _ = f.WriteString(versionString)
	err = f.Close()
	require.NoError(t, err)

	_, err = NewFileKeystoreWithPassword(temporaryPath, NewSecureString([]byte("")))
	if assert.Error(t, err) {
		assert.Equal(t, err, fmt.Errorf("corrupt or empty keystore"))
	}
}

func createAndReadKeystoreSecret(t *testing.T, password []byte, key string, value []byte) {
	path := GetTemporaryKeystoreFile(t)
	defer os.Remove(path)

	keyStore, err := NewFileKeystoreWithPassword(path, NewSecureString(password))
	require.NoError(t, err)

	writableKeystore, err := AsWritableKeystore(keyStore)
	require.NoError(t, err)

	err = writableKeystore.Store(key, value)
	require.NoError(t, err)
	err = writableKeystore.Save()
	require.NoError(t, err)

	newStore, err := NewFileKeystoreWithPassword(path, NewSecureString(password))
	require.NoError(t, err)
	s, _ := newStore.Retrieve(key)
	v, _ := s.Get()
	require.Equal(t, v, value)
}

func createAndReadKeystoreWithPassword(t *testing.T, password []byte) {
	path := GetTemporaryKeystoreFile(t)
	defer os.Remove(path)

	keyStore, err := NewFileKeystoreWithPassword(path, NewSecureString(password))
	require.NoError(t, err)

	writableKeystore, err := AsWritableKeystore(keyStore)
	require.NoError(t, err)

	err = writableKeystore.Store("hello", []byte("world"))
	require.NoError(t, err)
	err = writableKeystore.Save()
	require.NoError(t, err)

	newStore, err := NewFileKeystoreWithPassword(path, NewSecureString(password))
	require.NoError(t, err)
	s, _ := newStore.Retrieve("hello")
	v, _ := s.Get()

	require.Equal(t, v, []byte("world"))
}

// CreateAnExistingKeystore creates a keystore with an existing key
// "output.elasticsearch.password" with the value "secret".
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

	err = writableKeystore.Store(keyValue, secretValue)
	if err != nil {
		panic(err)
	}
	err = writableKeystore.Save()
	if err != nil {
		panic(err)
	}
	return keyStore
}

// GetTemporaryKeystoreFile create a temporary file on disk to save the keystore.
func GetTemporaryKeystoreFile(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "keystore")
}

func TestRandomBytesLength(t *testing.T) {
	r1, _ := randomBytes(5)
	require.Equal(t, len(r1), 5)

	r2, _ := randomBytes(4)
	require.Equal(t, len(r2), 4)
	require.NotEqual(t, string(r1[:]), string(r2[:]))
}

func TestRandomBytes(t *testing.T) {
	v1, err := randomBytes(10)
	require.NoError(t, err)
	v2, err := randomBytes(10)
	require.NoError(t, err)

	// unlikely to get 2 times the same results
	require.False(t, bytes.Equal(v1, v2))
}
