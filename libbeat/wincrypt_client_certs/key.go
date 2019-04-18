package wincrypt_client_certs

import (
	"crypto"
	"crypto/x509"
	"errors"
	"github.com/elastic/beats/libbeat/wincrypt_client_certs/ncrypt"
	"github.com/elastic/beats/libbeat/wincrypt_client_certs/wincrypt"
	"unsafe"
)

// functions to be monkey patched in test classes
var wincrypt_CryptAcquireCertificatePrivateKey = wincrypt.CryptAcquireCertificatePrivateKey

func privateKeyFromCertContext(certContext *wincrypt.CERT_CONTEXT, x509_cert *x509.Certificate)(privateKey crypto.PrivateKey, err error) {
	var handle wincrypt.HCRYPTPROV_OR_NCRYPT_KEY_HANDLE
    var keySpec uint32
	var freeFlag int = 0
	err = wincrypt_CryptAcquireCertificatePrivateKey(
		certContext,
		wincrypt.CRYPT_ACQUIRE_ONLY_NCRYPT_KEY_FLAG | wincrypt.CRYPT_ACQUIRE_SILENT_FLAG,
		unsafe.Pointer(uintptr(0)),
		&handle,
		&keySpec,
		&freeFlag,
	)
	if err != nil {
		return nil, errors.New("wincrypt: failed to get private key from certificate")
	}

	switch keySpec {
		case wincrypt.CERT_NCRYPT_KEY_SPEC:
				return &NcyptKey{
					private_key: ncrypt.NCRYPT_KEY_HANDLE(handle),
					public_key: x509_cert.PublicKey,
					freeFlag: freeFlag == 1,
				}, nil
		default:
			return nil, errors.New("wincrypt: only ncrypt is currently implemented")
	}
}
