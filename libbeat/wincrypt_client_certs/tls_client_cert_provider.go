package wincrypt_client_certs

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/Knetic/govaluate"
	"github.com/elastic/beats/libbeat/wincrypt_client_certs/wincrypt"
	"go.uber.org/multierr"
	"io"
	"strings"
	"syscall"
	"unsafe"
)

type tlsClientCertProvider struct {
	config       *Config
	closers      []io.Closer
	storeHandles []wincrypt.HCERTSTORE
	govaluate.EvaluableExpression
	Certificates []*tls.Certificate
}


type ClientCertificateGetter interface {
	GetClientCertificate(requestInfo *tls.CertificateRequestInfo) (*tls.Certificate, error)
}

// functions to be monkey patched in test classes
var wincrypt_CertOpenStore = wincrypt.CertOpenStore
var wincrypt_CertCloseStore = wincrypt.CertCloseStore
var wincrypt_CertEnumCertificatesInStore = wincrypt.CertEnumCertificatesInStore
var _privateKeyFromCertContext = privateKeyFromCertContext

func New(config *Config) (store *tlsClientCertProvider, err error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	storeHandles := make([]wincrypt.HCERTSTORE, len(config.Stores))
	for i, storeDef := range (config.Stores) {
		storeHandles[i], err = openStore(storeDef)
		if err != nil {
			return nil, err
		}
	}

	t := &tlsClientCertProvider{
		config:       config,
		closers:      make([]io.Closer, 0),
		storeHandles: storeHandles,
	}

	err = t.scanStores()
	if err != nil {
		return nil, err
	}

	return t, nil
}

func openStore(storeDef string) (hcertstore wincrypt.HCERTSTORE, err error) {
	s := strings.Split(storeDef, "/")
	location := s[0]
	provider := s[1]

	storeProviderUTF16, _ := syscall.UTF16FromString(provider)

	var flags uint32 = wincrypt.CERT_STORE_READONLY_FLAG | wincrypt.CERT_STORE_OPEN_EXISTING_FLAG
	if strings.EqualFold(location, "LocalMachine") {
		flags = flags | wincrypt.CERT_SYSTEM_STORE_LOCAL_MACHINE
	} else {
		flags = flags | wincrypt.CERT_SYSTEM_STORE_CURRENT_USER
	}

	h, err := wincrypt_CertOpenStore(
		(*byte)(unsafe.Pointer(uintptr(wincrypt.CERT_STORE_PROV_SYSTEM_W))),
		0,
		wincrypt.HCRYPTPROV_LEGACY(0),
		flags,
		unsafe.Pointer(&storeProviderUTF16[0]),
	)

	if err != nil {
		return 0, fmt.Errorf("wincrypt: failed to open certificate store %q", storeDef)
	}

	return h, nil
}

func (t *tlsClientCertProvider) Close() (error) {
	var result error

	for _, c := range t.closers {
		err := c.Close()
		if err != nil {
			result = multierr.Append(result, err)
		}
	}

	for i, h := range t.storeHandles {
		if uintptr(h) != 0 {
			err := wincrypt_CertCloseStore(h, wincrypt.CERT_CLOSE_STORE_CHECK_FLAG)

			// If we fail to close the store because not all attached certificate handlers have been closed:
			// 	record error and retry without pending flag
			retry := false
			if err != nil {
				if syscall_e, ok := err.(syscall.Errno); ok {
					if syscall_e == wincrypt.CRYPT_E_PENDING_CLOSE {
						format := "wincrypt: not all attached handlers to store %q have been closed (leaking memory)"
						result = multierr.Append(result, fmt.Errorf(format, t.config.Stores[i]))
						retry = true
					}
				}
			}

			if retry {
				err = wincrypt_CertCloseStore(h, 0)
			}

			if err != nil {
				result = multierr.Append(result, fmt.Errorf("wincrypt: failed to close store %q", t.config.Stores[i]))
			}
		}

		t.storeHandles[i] = wincrypt.HCERTSTORE(uintptr(0))
	}

	return result
}

func (t *tlsClientCertProvider) GetClientCertificate(requestInfo *tls.CertificateRequestInfo) (*tls.Certificate, error) {
	return t.Certificates[0], nil
}

func (t *tlsClientCertProvider) scanStores() (error) {
	// Scan the given stores for Certificates that fulfill our requirements.
	// Ignore all Certificates that do not contain a private key or a inaccessible for some reason.

	expr, _ := govaluate.NewEvaluableExpression(t.config.Query)

	t.Certificates = make([]*tls.Certificate, 0)

	for i, store := range (t.storeHandles) {
		var certContext *wincrypt.CERT_CONTEXT = nil
		var err error
	certLoop:
		for true {
			certContext, err = wincrypt_CertEnumCertificatesInStore(store, certContext)
			switch err {
			case io.EOF:
				break certLoop
			case nil:
				break
			default:
				return fmt.Errorf("wincrypt: failed to list Certificates in store %q", t.config.Stores[i])
			}

			x509_cert, err := x509.ParseCertificate(copyCBytesToSlice(certContext.PbCertEncoded, int(certContext.CbCertEncoded)))
			if err != nil {
				continue certLoop
			}

			result, err := expr.Eval((*X509Parameters)(x509_cert))
			if err != nil {
				return fmt.Errorf("wincrypt: failed to evaluate query: %s", err.Error())
			}

			boolVal, ok := result.(bool)
			if !ok {
				return fmt.Errorf("wincrypt: query did not return a boolean value")
			}

			if boolVal {
				// try to get the private key
				privateKey, err := _privateKeyFromCertContext(certContext, x509_cert)
				if err != nil {
					continue certLoop
				}

				if c, ok := privateKey.(io.Closer); ok {
					t.closers = append(t.closers, c)
				}

				cert := &tls.Certificate{
					Certificate: [][]byte{x509_cert.Raw},
					PrivateKey:  privateKey,
					Leaf:        x509_cert,
				}
				t.Certificates = append(t.Certificates, cert)
			}
		}
	}

	if len(t.Certificates) < 1 {
		return errors.New("wincrypt: found no Certificates machting criteria")
	}

	return nil
}

func copyCBytesToSlice(p *byte, n int) []byte {
	s := make([]byte, n, n)

	for i, addr := 0, uintptr(unsafe.Pointer(p)); i < n; i, addr = i+1, addr+1 {
		s[i] = *(*byte)(unsafe.Pointer(addr))
	}

	return s
}
