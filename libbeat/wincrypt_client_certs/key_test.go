package wincrypt_client_certs

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_privateKeyFromCertContext(t *testing.T) {
	InitMocks(false)

	k, err := privateKeyFromCertContext(MockNyTimesCertContext, NytimesX509Cert)

	require.NoError(t, err)
	require.IsType(t, &NcyptKey{}, k)

	key := k.(*NcyptKey)

	assert.Equal(t, MockNcryptKey, key.private_key)
	assert.Equal(t, NytimesX509Cert.PublicKey, key.public_key)
	assert.Equal(t, true, key.freeFlag)
}

func Test_privateKeyFromCertContext_SyscallFailure(t *testing.T) {
	InitMocks(false)
	MockCryptAcquireCertificatePrivateKeyFail = true

	_, err := privateKeyFromCertContext(MockNyTimesCertContext, NytimesX509Cert)

	assert.Error(t, err)
}
