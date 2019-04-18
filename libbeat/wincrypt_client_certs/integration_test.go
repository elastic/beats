// +build integration

package wincrypt_client_certs

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/tls"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"runtime"
	"testing"
)

func TestMain(m *testing.M) {
	importTestCert()
	result := m.Run()
	removeTestCert()

	os.Exit(result)
}

func _getTestPath(filename string) (string) {
	_, testfile_path, _, _ := runtime.Caller(0)
	return path.Join(path.Dir(testfile_path), filename)
}

func importTestCert() {
	InitMocks(true)

	command := exec.Command("certutil", "-f", "-p", "test", "-user", "-Silent", "-importpfx", "my", _getTestPath("integration_test.pfx"))
		err := command.Run()
		if err != nil {
			panic(fmt.Sprintf("failed to import test certificate: %v", err))
	}
}

func removeTestCert() {
	InitMocks(true)

	command := exec.Command("certutil", "-f", "-user", "-Silent", "-delstore", "my", "4e8793f6b76b74a04ffb025c9ab89e01")
	err := command.Run()
	if err != nil {
		panic(fmt.Sprintf("failed to remove test certificate: %v", err))
	}
}

func TestIntegration_GetClientCertificate(t *testing.T) {
	InitMocks(true)

	store, _ := _getTestCertificate(t)

	err := store.Close()
	assert.NoError(t, err)
}

func _getTestCertificate(t *testing.T)(*tlsClientCertProvider, *tls.Certificate) {
	store, err := New(&Config{
		Stores: []string{"CurrentUser/My", "LocalMachine/My"},
		Query:  "Subject_CommonName == 'test.example.org'",
	})
	assert.NoError(t, err)
	require.NotNil(t, store)

	cert, err := store.GetClientCertificate(nil)
	assert.NoError(t, err)
	require.NotNil(t, cert)
	require.Len(t, cert.Certificate, 1)

	return store, cert
}

func TestIntegration_DecryptPKCS(t *testing.T) {
	InitMocks(true)

	store, cert := _getTestCertificate(t)

	require.NotNil(t, cert.PrivateKey)
	decrypter, ok := cert.PrivateKey.(crypto.Decrypter)
	require.True(t, ok, "could not typecast private private_key to decrypter")

	randReader := rand.New(rand.NewSource(123))
	publicKey := decrypter.Public().(*rsa.PublicKey)
	msg, err := rsa.EncryptPKCS1v15(randReader, publicKey, []byte("Hello"))
	require.NoError(t, err)

	cleartext, err := decrypter.Decrypt(nil, msg, nil)
	assert.NoError(t, err)
	assert.Equal(t, "Hello", string(cleartext))

	err = store.Close()
	assert.NoError(t, err)
}

func TestIntegration_DecryptOEAPI(t *testing.T) {
	InitMocks(true)

	store, cert := _getTestCertificate(t)

	require.NotNil(t, cert.PrivateKey)
	decrypter, ok := cert.PrivateKey.(crypto.Decrypter)
	require.True(t, ok, "could not typecast private private_key to decrypter")

	randReader := rand.New(rand.NewSource(123))
	publicKey := decrypter.Public().(*rsa.PublicKey)

	label := []byte("Test")

	msg, err := rsa.EncryptOAEP(crypto.MD5.New(), randReader, publicKey, []byte("Hello"), label)
	require.NoError(t, err)

	cleartext, err := decrypter.Decrypt(nil, msg, &rsa.OAEPOptions{
		Hash:  crypto.MD5,
		Label: label,
	})
	assert.NoError(t, err)
	assert.Equal(t, "Hello", string(cleartext))

	err = store.Close()
	assert.NoError(t, err)
}

func TestIntegration_DecryptOEAPI_Empty_Label(t *testing.T) {
	InitMocks(true)

	store, cert := _getTestCertificate(t)

	require.NotNil(t, cert.PrivateKey)
	decrypter, ok := cert.PrivateKey.(crypto.Decrypter)
	require.True(t, ok, "could not typecast private private_key to decrypter")

	randReader := rand.New(rand.NewSource(123))
	publicKey := decrypter.Public().(*rsa.PublicKey)

	label := []byte("")

	msg, err := rsa.EncryptOAEP(crypto.SHA1.New(), randReader, publicKey, []byte("Hello"), label)
	require.NoError(t, err)

	cleartext, err := decrypter.Decrypt(nil, msg, &rsa.OAEPOptions{
		Hash:  crypto.SHA1,
		Label: label,
	})
	assert.NoError(t, err)
	assert.Equal(t, "Hello", string(cleartext))

	err = store.Close()
	assert.NoError(t, err)
}

func TestIntegration_DecryptSessionKey(t *testing.T) {
	InitMocks(true)

	store, cert := _getTestCertificate(t)

	require.NotNil(t, cert.PrivateKey)
	decrypter, ok := cert.PrivateKey.(crypto.Decrypter)
	require.True(t, ok, "could not typecast private private_key to decrypter")

	randReader := rand.New(rand.NewSource(123))
	publicKey := decrypter.Public().(*rsa.PublicKey)
	msg, err := rsa.EncryptPKCS1v15(randReader, publicKey, []byte("0123456789"))
	require.NoError(t, err)

	cleartext, err := decrypter.Decrypt(randReader, msg, &rsa.PKCS1v15DecryptOptions{10})
	assert.NoError(t, err)
	assert.Equal(t, "0123456789", string(cleartext))

	err = store.Close()
	assert.NoError(t, err)
}

func TestIntegration_DecryptSessionKeyInvalid(t *testing.T) {
	InitMocks(true)

	store, cert := _getTestCertificate(t)

	require.NotNil(t, cert.PrivateKey)
	decrypter, ok := cert.PrivateKey.(crypto.Decrypter)
	require.True(t, ok, "could not typecast private private_key to decrypter")


	randReader := rand.New(rand.NewSource(123))

	dummyKey, err := rsa.GenerateKey(randReader, 1024)
	require.NoError(t, err)

	publicKey := dummyKey.Public().(*rsa.PublicKey)
	msg, err := rsa.EncryptPKCS1v15(randReader, publicKey, []byte("0123456789"))
	require.NoError(t, err)

	cleartext, err := decrypter.Decrypt(randReader, msg, &rsa.PKCS1v15DecryptOptions{10})
	assert.NoError(t, err)
	assert.NotEqual(t, "0123456789", string(cleartext))
	assert.Len(t, cleartext, 10)

	err = store.Close()
	assert.NoError(t, err)
}

func TestIntegration_SignPkcs1(t *testing.T) {
	InitMocks(true)

	store, cert := _getTestCertificate(t)

	require.NotNil(t, cert.PrivateKey)
	signer, ok := cert.PrivateKey.(crypto.Signer)
	require.True(t, ok, "could not typecast private private_key to signer")

	randReader := rand.New(rand.NewSource(123))

	hash := sha1.Sum([]byte("hello"))
	signature, err := signer.Sign(randReader, hash[:], crypto.SHA1)
	assert.NoError(t, err)

    err = rsa.VerifyPKCS1v15(signer.Public().(*rsa.PublicKey), crypto.SHA1, hash[:], signature)
    assert.NoError(t, err)

	err = store.Close()
	assert.NoError(t, err)
}

func TestIntegration_SignPss(t *testing.T) {
	InitMocks(true)

	store, cert := _getTestCertificate(t)

	require.NotNil(t, cert.PrivateKey)
	signer, ok := cert.PrivateKey.(crypto.Signer)
	require.True(t, ok, "could not typecast private private_key to signer")

	randReader := rand.New(rand.NewSource(123))

	opts := &rsa.PSSOptions{
		SaltLength: 3,
		Hash: crypto.SHA1,
	}

	hash := sha1.Sum([]byte("hello"))

	signature, err := signer.Sign(randReader, hash[:], opts)
	assert.NoError(t, err)

    err = rsa.VerifyPSS(signer.Public().(*rsa.PublicKey), crypto.SHA1, hash[:], signature, opts)
    assert.NoError(t, err)

	err = store.Close()
	assert.NoError(t, err)
}
