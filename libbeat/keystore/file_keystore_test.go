package keystore

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/docker/docker/pkg/ioutils"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

var keyValue = "output.elasticsearch.password"
var secretValue = []byte("secret")

func TestCanCreateAKeyStore(t *testing.T) {
	path := GetTemporaryKeystoreFile()
	defer os.Remove(path)

	keystore, err := NewFileKeystore(path)
	assert.NoError(t, err)
	assert.Nil(t, keystore.Store(keyValue, secretValue))
	assert.Nil(t, keystore.Save())
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

	keystore, _ := NewFileKeystore(path)
	_, err := keystore.Retrieve(keyValue)
	assert.NoError(t, err)

	keystore.Delete(keyValue)
	_, err = keystore.Retrieve(keyValue)
	assert.Error(t, err)

	_ = keystore.Save()
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

	keystore := CreateAnExistingKeystore(path)
	err := keystore.Store("newkey", []byte("newsecret"))
	assert.NoError(t, err)
	err = keystore.Save()
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

	keystore := CreateAnExistingKeystore(path)

	keys, err := keystore.List()

	assert.NoError(t, err)
	assert.Equal(t, len(keys), 1)
	assert.Equal(t, keys[0], keyValue)
}

func TestCannotDecryptKeyStoreWithWrongPassword(t *testing.T) {
	path := GetTemporaryKeystoreFile()
	defer os.Remove(path)

	keystore, err := NewFileKeystoreWithPassword(path, NewSecureString([]byte("password")))
	keystore.Store("hello", []byte("world"))
	keystore.Save()

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

	keystore := CreateAnExistingKeystore(path)

	// Add a bit more data of different type
	keystore.Store("super.nested", []byte("hello"))
	keystore.Save()

	cfg, err := keystore.GetConfig()
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

	keystore, err := NewFileKeystoreWithPassword(path, NewSecureString(password))
	assert.Nil(t, err)

	keystore.Store(key, value)
	keystore.Save()

	newStore, err := NewFileKeystoreWithPassword(path, NewSecureString(password))
	s, _ := newStore.Retrieve(key)
	v, _ := s.Get()
	assert.Equal(t, v, value)
}

func createAndReadKeystoreWithPassword(t *testing.T, password []byte) {
	path := GetTemporaryKeystoreFile()
	defer os.Remove(path)

	keystore, err := NewFileKeystoreWithPassword(path, NewSecureString(password))
	assert.NoError(t, err)

	keystore.Store("hello", []byte("world"))
	keystore.Save()

	newStore, err := NewFileKeystoreWithPassword(path, NewSecureString(password))
	s, _ := newStore.Retrieve("hello")
	v, _ := s.Get()

	assert.Equal(t, v, []byte("world"))
}

// CreateAnExistingKeystore creates a keystore with an existing key
/// `output.elasticsearch.password` with the value `secret`.
func CreateAnExistingKeystore(path string) Keystore {
	keystore, err := NewFileKeystore(path)
	// Fail fast in the test suite
	if err != nil {
		panic(err)
	}
	keystore.Store(keyValue, secretValue)
	keystore.Save()
	return keystore
}

// GetTemporaryKeystoreFile create a temporary file on disk to save the keystore.
func GetTemporaryKeystoreFile() string {
	path, err := ioutils.TempDir("", "testing")
	if err != nil {
		panic(err)
	}
	return filepath.Join(path, "keystore")
}
