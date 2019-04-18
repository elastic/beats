package wincrypt

import (
	"io"
	"syscall"
	"unsafe"
)

const (
	CERT_STORE_PROV_SYSTEM_W = 10

	CERT_SYSTEM_STORE_LOCAL_MACHINE = 0x00020000
	CERT_SYSTEM_STORE_CURRENT_USER  = 0x00010000

	CERT_STORE_READONLY_FLAG = 0x00008000

	X509_ASN_ENCODING   = 0x00000001
	PKCS_7_ASN_ENCODING = 0x00010000

	CERT_STORE_OPEN_EXISTING_FLAG = 0x00004000

	CERT_CLOSE_STORE_CHECK_FLAG = 0x00000002

	CRYPT_E_PENDING_CLOSE = 0x8009200F

	CERT_NCRYPT_KEY_SPEC = 0xFFFFFFFF

	CRYPT_ACQUIRE_SILENT_FLAG        = 0x00000040

	CRYPT_ACQUIRE_ONLY_NCRYPT_KEY_FLAG   = 0x00040000

	PKCS12_NO_PERSIST_KEY = 0x00008000
)

type HCRYPTPROV_LEGACY uintptr
type HCERTSTORE uintptr

type HCRYPTPROV uintptr
type HCRYPTPROV_OR_NCRYPT_KEY_HANDLE uintptr;

type CRYPTOAPI_BLOB struct {
	CbData uint32
	PbData *byte
}

type CRYPT_DATA_BLOB CRYPTOAPI_BLOB

type CERT_INFO struct {
	// Not implemented
}

type CERT_CONTEXT struct {
	DwCertEncodingType uint32
	PbCertEncoded      *byte
	CbCertEncoded      uint32
	PCertInfo          *CERT_INFO
	HCertStore         uintptr
}

type CTL_USAGE struct {
	CUsageIdentifier uint32
	RgpszUsageIdentifier **byte
}
type CERT_ENHKEY_USAGE CTL_USAGE

var (
	crypt32dll  = syscall.NewLazyDLL("Crypt32.dll")

	certOpenStore                     = crypt32dll.NewProc("CertOpenStore")
	certCloseStore                    = crypt32dll.NewProc("CertCloseStore")
	certEnumCertificatesInStore       = crypt32dll.NewProc("CertEnumCertificatesInStore")
	cryptAcquireCertificatePrivateKey = crypt32dll.NewProc("CryptAcquireCertificatePrivateKey")
	pfxImportCertStore				  = crypt32dll.NewProc("PFXImportCertStore")
)

func CertEnumCertificatesInStore(hCertStore HCERTSTORE, pPrevCertContext *CERT_CONTEXT) (*CERT_CONTEXT, error) {
	context, _, _ := certEnumCertificatesInStore.Call(uintptr(hCertStore), uintptr(unsafe.Pointer(pPrevCertContext)))
	if context == 0 {
		return nil, io.EOF
	}

	return (*CERT_CONTEXT)(unsafe.Pointer(context)), nil
}

func CertOpenStore(lpszStoreProvider *byte, dwEncodingType uint32, hCryptProv HCRYPTPROV_LEGACY, dwFlags uint32, pvPara unsafe.Pointer) (store HCERTSTORE, err error) {
	h, _, err := certOpenStore.Call(
		uintptr(unsafe.Pointer(lpszStoreProvider)),
		uintptr(dwEncodingType),
		uintptr(hCryptProv),
		uintptr(dwFlags),
		uintptr(pvPara),
	)
	if h == 0 {
		return 0, err
	}

	return HCERTSTORE(h), nil
}

func CertCloseStore(hCertStore HCERTSTORE, dwFlags uint32) (error) {
	status, _, err := certCloseStore.Call(uintptr(hCertStore), uintptr(dwFlags))

	if status != 1 {
		return err
	}

	return nil
}

func CryptAcquireCertificatePrivateKey(
	pCert *CERT_CONTEXT,
	dwFlags uint32,
	pvParameters unsafe.Pointer,
	phCryptProvOrNCryptKey *HCRYPTPROV_OR_NCRYPT_KEY_HANDLE,
	pdwKeySpec *uint32,
	pfCallerFreeProvOrNCryptKey *int,
) (error) {
	status, _, err := cryptAcquireCertificatePrivateKey.Call(
		uintptr(unsafe.Pointer(pCert)),
		uintptr(dwFlags),
		uintptr(pvParameters),
		uintptr(unsafe.Pointer(phCryptProvOrNCryptKey)),
		uintptr(unsafe.Pointer(pdwKeySpec)),
		uintptr(unsafe.Pointer(pfCallerFreeProvOrNCryptKey)),
	)


	if status != 1 {
		return err
	}

	return nil
}

func PFXImportCertStore(
	pPFXC *CRYPT_DATA_BLOB,
	szPassword	*uint16,
	dwFlags uint32,
) (HCERTSTORE, error) {
	hcertstore, _ , err := pfxImportCertStore.Call(
		uintptr(unsafe.Pointer(pPFXC)),
		uintptr(unsafe.Pointer(szPassword)),
		uintptr(dwFlags),
	)

	if (hcertstore == 0) {
		return HCERTSTORE(0), err
	}

	return HCERTSTORE(hcertstore), nil
}