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
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"golang.org/x/crypto/pbkdf2"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/elastic-agent-libs/file"
)

const (
	filePermission = 0600

	// Encryption Related constants
	iVLength        = 12
	saltLength      = 64
	iterationsCount = 10000
	keyLength       = 32
)

// Version of the keystore format, will be added at the beginning of the file.
var version = []byte("v1")

// Packager defines a keystore that we can read the raw bytes and be packaged in an artifact.
type Packager interface {
	Package() ([]byte, error)
	ConfiguredPath() string
}

// FileKeystore Allows to store key / secrets pair securely into an encrypted local file.
type FileKeystore struct {
	sync.RWMutex
	Path     string
	secrets  map[string]serializableSecureString
	dirty    bool
	password *SecureString
}

// Allow the original SecureString type to be correctly serialized to json.
type serializableSecureString struct {
	*SecureString
	Value []byte `json:"value"`
}

// Factory Create the right keystore with the configured options.
func Factory(cfg *common.Config, defaultPath string) (Keystore, error) {
	config := defaultConfig

	if cfg == nil {
		cfg = common.NewConfig()
	}
	err := cfg.Unpack(&config)

	if err != nil {
		return nil, fmt.Errorf("could not read keystore configuration, err: %v", err)
	}

	if config.Path == "" {
		config.Path = defaultPath
	}

	keystore, err := NewFileKeystore(config.Path)
	return keystore, err
}

// NewFileKeystore returns an new File based keystore or an error, currently users cannot set their
// own password on the keystore, the default password will be an empty string. When the keystore
// is initialized the secrets are automatically loaded into memory.
func NewFileKeystore(keystoreFile string) (Keystore, error) {
	return NewFileKeystoreWithPassword(keystoreFile, NewSecureString([]byte("")))
}

// NewFileKeystoreWithPassword return a new File based keystore or an error, allow to define what
// password to use to create the keystore.
func NewFileKeystoreWithPassword(keystoreFile string, password *SecureString) (Keystore, error) {
	keystore := FileKeystore{
		Path:     keystoreFile,
		dirty:    false,
		password: password,
		secrets:  make(map[string]serializableSecureString),
	}

	err := keystore.load()
	if err != nil {
		return nil, err
	}

	return &keystore, nil
}

// Retrieve return a SecureString instance that will contains both the key and the secret.
func (k *FileKeystore) Retrieve(key string) (*SecureString, error) {
	k.RLock()
	defer k.RUnlock()

	secret, ok := k.secrets[key]
	if !ok {
		return nil, ErrKeyDoesntExists
	}
	return NewSecureString(secret.Value), nil
}

// Store add the key pair to the secret store and mark the store as dirty.
func (k *FileKeystore) Store(key string, value []byte) error {
	k.Lock()
	defer k.Unlock()

	k.secrets[key] = serializableSecureString{Value: value}
	k.dirty = true
	return nil
}

// Delete an existing key from the store and mark the store as dirty.
func (k *FileKeystore) Delete(key string) error {
	k.Lock()
	defer k.Unlock()

	delete(k.secrets, key)
	k.dirty = true
	return nil
}

// Save persists the in memory data to disk if needed.
func (k *FileKeystore) Save() error {
	k.Lock()
	err := k.doSave(true)
	k.Unlock()
	return err
}

// List return the availables keys.
func (k *FileKeystore) List() ([]string, error) {
	k.RLock()
	defer k.RUnlock()

	keys := make([]string, 0, len(k.secrets))
	for key := range k.secrets {
		keys = append(keys, key)
	}

	return keys, nil
}

// GetConfig returns common.Config representation of the key / secret pair to be merged with other
// loaded configuration.
func (k *FileKeystore) GetConfig() (*common.Config, error) {
	k.RLock()
	defer k.RUnlock()

	configHash := make(map[string]interface{})
	for key, secret := range k.secrets {
		configHash[key] = string(secret.Value)
	}

	return common.NewConfigFrom(configHash)
}

// Create create an empty keystore, if the store already exist we will return an error.
func (k *FileKeystore) Create(override bool) error {
	k.Lock()
	k.secrets = make(map[string]serializableSecureString)
	k.dirty = true
	err := k.doSave(override)
	k.Unlock()
	return err
}

// IsPersisted return if the keystore is physically persisted on disk.
func (k *FileKeystore) IsPersisted() bool {
	k.Lock()
	defer k.Unlock()

	// We just check if the file is present on disk, we don't need to do any validation
	// for a file based keystore, since all the keys will be fetched when we initialize the object
	// if the file is invalid it will already fails. Creating a new FileKeystore will raise
	// any errors concerning the permissions
	f, err := os.OpenFile(k.Path, os.O_RDONLY, filePermission)
	if err != nil {
		return false
	}
	f.Close()
	return true
}

// doSave lock/unlocking of the resource need to be done by the caller.
func (k *FileKeystore) doSave(override bool) error {
	if !k.dirty {
		return nil
	}

	temporaryPath := fmt.Sprintf("%s.tmp", k.Path)

	w := new(bytes.Buffer)
	jsonEncoder := json.NewEncoder(w)
	if err := jsonEncoder.Encode(k.secrets); err != nil {
		return fmt.Errorf("cannot serialize the keystore before saving it to disk: %v", err)
	}

	encrypted, err := k.encrypt(w)
	if err != nil {
		return fmt.Errorf("cannot encrypt the keystore: %v", err)
	}

	flags := os.O_RDWR | os.O_CREATE
	if override {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_EXCL
	}

	f, err := os.OpenFile(temporaryPath, flags, filePermission)
	if err != nil {
		return fmt.Errorf("cannot open file to save the keystore to '%s', error: %s", k.Path, err)
	}

	f.Write(version)
	base64Encoder := base64.NewEncoder(base64.StdEncoding, f)
	io.Copy(base64Encoder, encrypted)
	base64Encoder.Close()
	f.Sync()
	f.Close()

	err = file.SafeFileRotate(k.Path, temporaryPath)
	if err != nil {
		os.Remove(temporaryPath)
		return fmt.Errorf("cannot replace the existing keystore, with the new keystore file at '%s', error: %s", k.Path, err)
	}
	os.Remove(temporaryPath)

	k.dirty = false
	return nil
}

func (k *FileKeystore) loadRaw() ([]byte, error) {
	f, err := os.OpenFile(k.Path, os.O_RDONLY, filePermission)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	if common.IsStrictPerms() {
		if err := k.checkPermissions(k.Path); err != nil {
			return nil, err
		}
	}

	raw, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	v := raw[0:len(version)]
	if !bytes.Equal(v, version) {
		return nil, fmt.Errorf("keystore format doesn't match expected version: '%s' got '%s'", version, v)
	}

	if len(raw) <= len(version) {
		return nil, fmt.Errorf("corrupt or empty keystore")
	}

	return raw, nil
}

func (k *FileKeystore) load() error {
	k.Lock()
	defer k.Unlock()

	raw, err := k.loadRaw()
	if err != nil {
		return err
	}

	if len(raw) == 0 {
		return nil
	}

	base64Decoder := base64.NewDecoder(base64.StdEncoding, bytes.NewReader(raw[len(version):]))
	plaintext, err := k.decrypt(base64Decoder)
	if err != nil {
		return fmt.Errorf("could not decrypt the keystore: %v", err)
	}
	jsonDecoder := json.NewDecoder(plaintext)
	return jsonDecoder.Decode(&k.secrets)
}

// Encrypt the data payload using a derived keys and the AES-256-GCM algorithm.
func (k *FileKeystore) encrypt(reader io.Reader) (io.Reader, error) {
	// randomly generate the salt and the initialization vector, this information will be saved
	// on disk in the file as part of the header
	iv, err := common.RandomBytes(iVLength)

	if err != nil {
		return nil, err
	}

	salt, err := common.RandomBytes(saltLength)
	if err != nil {
		return nil, err
	}

	// Stretch the user provided key
	password, _ := k.password.Get()
	passwordBytes := k.hashPassword(password, salt)

	// Select AES-256: because len(passwordBytes) == 32 bytes
	block, err := aes.NewCipher(passwordBytes)
	if err != nil {
		return nil, fmt.Errorf("could not create the keystore cipher to encrypt, error: %s", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("could not create the keystore cipher to encrypt, error: %s", err)
	}

	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("could not read unencrypted data, error: %s", err)
	}

	encodedBytes := aesgcm.Seal(nil, iv, data, nil)

	// Generate the payload with all the additional information required to decrypt the
	// output format of the document: VERSION|SALT|IV|PAYLOAD
	buf := bytes.NewBuffer(salt)
	buf.Write(iv)
	buf.Write(encodedBytes)

	return buf, nil
}

// should receive an io.reader...
func (k *FileKeystore) decrypt(reader io.Reader) (io.Reader, error) {
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("could not read all the data from the encrypted file, error: %s", err)
	}

	if len(data) < saltLength+iVLength+1 {
		return nil, fmt.Errorf("missing information in the file for decrypting the keystore")
	}

	// extract the necessary information to decrypt the data from the data payload
	salt := data[0:saltLength]
	iv := data[saltLength : saltLength+iVLength]
	encodedBytes := data[saltLength+iVLength:]

	password, _ := k.password.Get()
	passwordBytes := k.hashPassword(password, salt)

	block, err := aes.NewCipher(passwordBytes)
	if err != nil {
		return nil, fmt.Errorf("could not create the keystore cipher to decrypt the data: %s", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("could not create the keystore cipher to decrypt the data: %s", err)
	}

	decodedBytes, err := aesgcm.Open(nil, iv, encodedBytes, nil)
	if err != nil {
		return nil, fmt.Errorf("could not decrypt keystore data: %s", err)
	}

	return bytes.NewReader(decodedBytes), nil
}

// checkPermission enforces permission on the keystore file itself, the file should have strict
// permission (0600) and the keystore should refuses to start if its not the case.
func (k *FileKeystore) checkPermissions(f string) error {
	if runtime.GOOS == "windows" {
		return nil
	}

	info, err := file.Stat(f)
	if err != nil {
		return err
	}

	euid := os.Geteuid()
	fileUID, _ := info.UID()
	perm := info.Mode().Perm()

	if fileUID != 0 && euid != fileUID {
		return fmt.Errorf(`config file ("%v") must be owned by the user identifier `+
			`(uid=%v) or root`, f, euid)
	}

	// Test if group or other have write permissions.
	if perm != filePermission {
		nameAbs, err := filepath.Abs(f)
		if err != nil {
			nameAbs = f
		}
		return fmt.Errorf(`file ("%v") can only be writable and readable by the `+
			`owner but the permissions are "%v" (to fix the permissions use: `+
			`'chmod go-wrx %v')`,
			f, perm, nameAbs)
	}

	return nil
}

// Package returns the bytes of the encrypted keystore.
func (k *FileKeystore) Package() ([]byte, error) {
	k.Lock()
	defer k.Unlock()
	return k.loadRaw()
}

// ConfiguredPath returns the path to the keystore.
func (k *FileKeystore) ConfiguredPath() string {
	return k.Path
}

func (k *FileKeystore) hashPassword(password, salt []byte) []byte {
	return pbkdf2.Key(password, salt, iterationsCount, keyLength, sha512.New)
}
