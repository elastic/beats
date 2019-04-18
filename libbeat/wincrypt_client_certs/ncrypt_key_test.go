package wincrypt_client_certs

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha1"
	"fmt"
	"github.com/elastic/beats/libbeat/wincrypt_client_certs/ncrypt"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
)

func testKey() (*NcyptKey) {
	return &NcyptKey{
		private_key: MockNcryptKey,
		public_key:  NytimesX509Cert.PublicKey,
		freeFlag:    true,
	}
}

func TestNcyptKey_Close(t *testing.T) {
	InitMocks(false)

	key := testKey()
	err := key.Close()
	assert.NoError(t, err)
	assert.Equal(t, 1, MockNCryptFreeObjectCalledCounter)

	MockNCryptFreeObjectFail = true
	key = testKey()
	err = key.Close()
	assert.Error(t, err)
}

func TestNcyptKey_Decrypt_OAEP(t *testing.T) {
	InitMocks(false)

	key := testKey()

	randReader := rand.New(rand.NewSource(123))

	msg := "Hello World"
	label := "hello"
	opts := &rsa.OAEPOptions{
		Hash:  crypto.MD5,
		Label: []byte(label),
	}
	decrypted, err := key.Decrypt(randReader, []byte(msg), opts)
	assert.NoError(t, err)

	expected := fmt.Sprintf("%s:OAEP:%s:%d", msg, "hello", &ncrypt.BCRYPT_MD5_ALGORITHM[0])
	assert.Equal(t, expected, string(decrypted))

	MockNCryptDecryptFailOnSizeCall = true
	_, err = key.Decrypt(randReader, []byte(msg), opts)
	assert.Error(t, err)

	MockNCryptDecryptFailOnSizeCall = false
	MockNCryptDecryptFailOnDecryptCall = true
	_, err = key.Decrypt(randReader, []byte(msg), opts)
	assert.Error(t, err)
}

func TestNcyptKey_Decrypt_PKCS1(t *testing.T) {
	InitMocks(false)

	key := testKey()

	randReader := rand.New(rand.NewSource(123))

	msg := "Hello World"
	decrypted, err := key.Decrypt(randReader, []byte(msg), &rsa.PKCS1v15DecryptOptions{0})
	assert.NoError(t, err)

	expected := fmt.Sprintf("%s:PKCS1", msg)
	assert.Equal(t, expected, string(decrypted))

	decrypted, err = key.Decrypt(randReader, []byte(msg), nil)
	assert.NoError(t, err)
	assert.Equal(t, expected, string(decrypted))

	MockNCryptDecryptFailOnSizeCall = true
	_, err = key.Decrypt(randReader, []byte(msg), &rsa.PKCS1v15DecryptOptions{0})
	assert.Error(t, err)

	MockNCryptDecryptFailOnSizeCall = false
	MockNCryptDecryptFailOnDecryptCall = true
	_, err = key.Decrypt(randReader, []byte(msg), &rsa.PKCS1v15DecryptOptions{0})
	assert.Error(t, err)
}

func TestNcyptKey_Decrypt_SessionKey(t *testing.T) {
	InitMocks(false)

	key := testKey()

	randReader := rand.New(rand.NewSource(123))

	msg := "Hello World"
	expected := fmt.Sprintf("%s:PKCS1", msg)
	decrypted, err := key.Decrypt(randReader, []byte(msg), &rsa.PKCS1v15DecryptOptions{len(expected)})
	assert.NoError(t, err)

	assert.Equal(t, expected, string(decrypted))

	// Check that a session key of appropriate length is returned when decryption fails
	MockNCryptDecryptFailOnDecryptCall = true
	decrypted, err = key.Decrypt(randReader, []byte(msg), &rsa.PKCS1v15DecryptOptions{5})
	assert.NoError(t, err)
	assert.Len(t, decrypted, 5)
}

func TestNcyptKey_Sign_PKCS1(t *testing.T) {
	InitMocks(false)

	key := testKey()

	randReader := rand.New(rand.NewSource(123))

	hash := sha1.Sum([]byte("hello"))
	signature, err := key.Sign(randReader, hash[:], crypto.SHA1)

	assert.NoError(t, err)
	expected := fmt.Sprintf("%s:PKCS1:%d", hash, &ncrypt.BCRYPT_SHA1_ALGORITHM[0])
	assert.Equal(t, expected, string(signature))

	MockNCryptSignHashFailOnSizeCall = true
	_, err = key.Sign(randReader, hash[:], crypto.SHA1)
	assert.Error(t, err)

	MockNCryptSignHashFailOnSizeCall = false
	MockNCryptSignHashFailOnSignCall = true
	_, err = key.Sign(randReader, hash[:], crypto.SHA1)
	assert.Error(t, err)
}

func TestNcyptKey_Sign_PSS(t *testing.T) {
	InitMocks(false)

	key := testKey()

	randReader := rand.New(rand.NewSource(123))

	hash := sha1.Sum([]byte("hello"))
	opts := &rsa.PSSOptions{
		SaltLength: 123,
		Hash:       crypto.SHA256,
	}
	signature, err := key.Sign(randReader, hash[:], opts)

	assert.NoError(t, err)
	expected := fmt.Sprintf("%s:PSS:%d:%d", hash, &ncrypt.BCRYPT_SHA256_ALGORITHM[0], 123)
	assert.Equal(t, expected, string(signature))

	MockNCryptSignHashFailOnSizeCall = true
	_, err = key.Sign(randReader, hash[:], opts)
	assert.Error(t, err)

	MockNCryptSignHashFailOnSizeCall = false
	MockNCryptSignHashFailOnSignCall = true
	_, err = key.Sign(randReader, hash[:], opts)
	assert.Error(t, err)
}
