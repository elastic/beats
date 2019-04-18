package wincrypt_client_certs

import (
	"github.com/elastic/beats/libbeat/wincrypt_client_certs/wincrypt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/multierr"
	"io"
	"testing"
)

func TestTlsClientCertProvider_GetClientCertificate(t *testing.T) {
	InitMocks(false)

    p, err := New(&Config{
        Stores: []string{"LocalMachine/My"},
        Query: "X509_Subject_CommonName == 'nytimes.com'",
    })
    require.NoError(t, err)

    certificate, err := p.GetClientCertificate(nil)
    require.NoError(t, err)
    require.NotNil(t, certificate)

    assert.Equal(t, MockNyTimesPrivateKey, certificate.PrivateKey)

    if (assert.Len(t, certificate.Certificate, 1)) {
        assert.Equal(t, NytimesRawCert, certificate.Certificate[0])
    }

    assert.Empty(t, certificate.OCSPStaple)

    if assert.NotEmpty(t, certificate.Leaf) {
        assert.Equal(t, "nytimes.com", certificate.Leaf.Subject.CommonName)
    }
}

func TestTlsClientCertProvider_New_QueryNotBool(t *testing.T) {
	InitMocks(false)

    _, err := New(&Config{
        Stores: []string{"LocalMachine/My"},
        Query: "'test'",
    })

    assert.Error(t, err)
	assert.Contains(t, err.Error(), "bool")
}

func TestTlsClientCertProvider_New_EnumarateFail(t *testing.T) {
	InitMocks(false)
	MockCertEnumCertificatesInStoreFail = true

    _, err := New(&Config{
        Stores: []string{"LocalMachine/My"},
		Query: "X509_Subject_CommonName == 'nytimes.com'",
    })

    assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list")
}

func TestTlsClientCertProvider_New_InvalidQuery(t *testing.T) {
	InitMocks(false)

    _, err := New(&Config{
        Stores: []string{"LocalMachine/My"},
        Query: "BOGUS_VAR",
    })

    assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to evaluate query")
}

func TestTlsClientCertProvider_New_InvalidConfig(t *testing.T) {
	InitMocks(false)

	_, err := New(&Config{
        Stores: []string{"bogus/My"},
		Query: "X509_Subject_CommonName == 'nytimes.com'",
    })

    assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid store definition")
}

func TestTlsClientCertProvider_New_OpenStoreFails(t *testing.T) {
	InitMocks(false)

	_, err := New(&Config{
        Stores: []string{"CurrentUser/My"},
		Query: "X509_Subject_CommonName == 'nytimes.com'",
    })

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open certificate store")
}

func TestTlsClientCertProvider_New_NoMatchingCertificates(t *testing.T) {
	InitMocks(false)

	_, err := New(&Config{
        Stores: []string{"LocalMachine/My"},
        Query: "X509_Subject_CommonName == 'bogus.org'",
    })

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "found no Certificates ")
}

func TestTlsClientCertProvider_New_InvalidX509Cert(t *testing.T) {
	InitMocks(false)
	NytimesRawCert = []byte{'B', 'O', 'G', 'U', 'S'}

	_, err := New(&Config{
        Stores: []string{"LocalMachine/My"},
		Query: "X509_Subject_CommonName == 'nytimes.com'",
    })

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "found no Certificates ")
}

func TestTlsClientCertProvider_New_GetPrivateKeyFail(t *testing.T) {
	InitMocks(false)
	MockPrivateKeyFromCertContextFail = true

	_, err := New(&Config{
        Stores: []string{"LocalMachine/My"},
		Query: "X509_Subject_CommonName == 'nytimes.com'",
    })

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "found no Certificates ")
}

func TestTlsClientCertProvider_Close(t *testing.T) {
    InitMocks(false)

    tlsClientCertProvider := &tlsClientCertProvider {
        config: &Config{
			Stores: []string{"LocalMachine/My"},
			Query: "X509_Subject_CommonName == 'nytimes.com'",
		},
		closers: []io.Closer{MockCloserType(0)},
        storeHandles: []wincrypt.HCERTSTORE{MockLocalMachineStore},
    }

    err := tlsClientCertProvider.Close()
    assert.NoError(t, err)
    assert.Equal(t, 1, MockCloserCalledCounter)
    assert.Equal(t, 1, MockCertCloseStoreCalledCounter)
}

func TestTlsClientCertProvider_CloseFailure(t *testing.T) {
	InitMocks(false)
	MockCloserFail = true
	MockCertCloseFailOnStoreCheckFlagSet = true
	MockCertCloseFail = true

    tlsClientCertProvider := &tlsClientCertProvider {
        config: &Config{
			Stores: []string{"LocalMachine/My"},
			Query: "X509_Subject_CommonName == 'nytimes.com'",
		},
		closers: []io.Closer{MockCloserType(0)},
        storeHandles: []wincrypt.HCERTSTORE{MockLocalMachineStore},
    }

    err := tlsClientCertProvider.Close()
    if assert.Error(t, err) {
    	errs := multierr.Errors(err)
        if assert.Len(t, errs, 3) {
            assert.Equal(t, MockError, errs[0])
            assert.Contains(t, errs[1].Error(), "memory")
            assert.Contains(t, errs[2].Error(), "failed to close store")
        }

    }
	assert.Equal(t, 1, MockCloserCalledCounter)
    assert.Equal(t, 2, MockCertCloseStoreCalledCounter)
}
