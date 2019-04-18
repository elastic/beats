package wincrypt_client_certs

import (
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/elastic/beats/libbeat/wincrypt_client_certs/ncrypt"
	"github.com/elastic/beats/libbeat/wincrypt_client_certs/wincrypt"
	"io"
	"syscall"
	"unsafe"
)

const NytimesCertPem = `
-----BEGIN CERTIFICATE-----
MIIJhTCCCG2gAwIBAgIRAL/WIxvqWarWy1Zu0IeNYO0wDQYJKoZIhvcNAQELBQAw
gZYxCzAJBgNVBAYTAkdCMRswGQYDVQQIExJHcmVhdGVyIE1hbmNoZXN0ZXIxEDAO
BgNVBAcTB1NhbGZvcmQxGjAYBgNVBAoTEUNPTU9ETyBDQSBMaW1pdGVkMTwwOgYD
VQQDEzNDT01PRE8gUlNBIE9yZ2FuaXphdGlvbiBWYWxpZGF0aW9uIFNlY3VyZSBT
ZXJ2ZXIgQ0EwHhcNMTgxMTI5MDAwMDAwWhcNMjAwMTE4MjM1OTU5WjCBxDELMAkG
A1UEBhMCVVMxDjAMBgNVBBETBTEwMDE4MREwDwYDVQQIEwhOZXcgWW9yazERMA8G
A1UEBxMITmV3IFlvcmsxFDASBgNVBAkTCzYyMCA4dGggQXZlMRswGQYDVQQKExJU
aGUgTmV3IFlvcmsgVGltZXMxGzAZBgNVBAsTElRoZSBOZXcgWW9yayBUaW1lczEZ
MBcGA1UECxMQTXVsdGktRG9tYWluIFNTTDEUMBIGA1UEAxMLbnl0aW1lcy5jb20w
ggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCqpbxBef7yIpiL7/xkUbY2
RvDRMmjPiv/HMaFM4KjqowJg2JTbqJmFhiJFuzKndVcUIpO37lVQ/Oallob9fPBY
cqAf0e6gFgueeucjHXPnID44qnZGFwj0wtnNmy7ItckEEVhT2OaCpROaeUI4jWHj
83NkAnxKHDDuH472BfRNeBgmsoXwdywV421vL9A1yhOpkvNrZBrj6u32i3Fz1+Gt
Snh4j4LvVC8ewXz3k70YH32gnkAaPOW/X0xTGJ63cqMIuVKq6dBCmhzCbPzVBerr
581FuXJ2Cyq/7242H/+XOu+h86nbETzG44TuoxOG2fnd+WhuUUKGKo6M5D3zjlw9
AgMBAAGjggWcMIIFmDAfBgNVHSMEGDAWgBSa8yvaz61Pti+7KkhIKhK3G0LBJDAd
BgNVHQ4EFgQUhiKsBlfdhdBgPQMAt93uFtyZBTQwDgYDVR0PAQH/BAQDAgWgMAwG
A1UdEwEB/wQCMAAwHQYDVR0lBBYwFAYIKwYBBQUHAwEGCCsGAQUFBwMCMFAGA1Ud
IARJMEcwOwYMKwYBBAGyMQECAQMEMCswKQYIKwYBBQUHAgEWHWh0dHBzOi8vc2Vj
dXJlLmNvbW9kby5jb20vQ1BTMAgGBmeBDAECAjBaBgNVHR8EUzBRME+gTaBLhklo
dHRwOi8vY3JsLmNvbW9kb2NhLmNvbS9DT01PRE9SU0FPcmdhbml6YXRpb25WYWxp
ZGF0aW9uU2VjdXJlU2VydmVyQ0EuY3JsMIGLBggrBgEFBQcBAQR/MH0wVQYIKwYB
BQUHMAKGSWh0dHA6Ly9jcnQuY29tb2RvY2EuY29tL0NPTU9ET1JTQU9yZ2FuaXph
dGlvblZhbGlkYXRpb25TZWN1cmVTZXJ2ZXJDQS5jcnQwJAYIKwYBBQUHMAGGGGh0
dHA6Ly9vY3NwLmNvbW9kb2NhLmNvbTCCAtQGA1UdEQSCAsswggLHggtueXRpbWVz
LmNvbYIVKi5hcGkuZGV2Lm55dGltZXMuY29tghEqLmFwaS5ueXRpbWVzLmNvbYIV
Ki5hcGkuc3RnLm55dGltZXMuY29tgg4qLmJldGEubnl0Lm5ldIITKi5ibG9ncy5u
eXRpbWVzLmNvbYIXKi5ibG9ncy5zdGcubnl0aW1lcy5jb22CGCouYmxvZ3M1LnN0
Zy5ueXRpbWVzLmNvbYISKi5kZXYuYmV0YS5ueXQubmV0ghcqLmRldi5ibG9ncy5u
eXRpbWVzLmNvbYINKi5kZXYubnl0LmNvbYINKi5kZXYubnl0Lm5ldIIRKi5kZXYu
bnl0aW1lcy5jb22CDSoubmV3c2Rldi5uZXSCESoubmV3c2Rldi5ueXQubmV0ghUq
Lm5ld3NkZXYubnl0aW1lcy5jb22CCSoubnl0LmNvbYIJKi5ueXQubmV0ggsqLm55
dGNvLmNvbYINKi5ueXRpbWVzLmNvbYIZKi5wYXlmbG93LnNieC5ueXRpbWVzLmNv
bYIRKi5zYngubnl0aW1lcy5jb22CEiouc3RnLmJldGEubnl0Lm5ldIIXKi5zdGcu
YmxvZ3Mubnl0aW1lcy5jb22CESouc3RnLm5ld3NkZXYubmV0ghUqLnN0Zy5uZXdz
ZGV2Lm55dC5uZXSCGSouc3RnLm5ld3NkZXYubnl0aW1lcy5jb22CDSouc3RnLm55
dC5jb22CDSouc3RnLm55dC5uZXSCESouc3RnLm55dGltZXMuY29tghAqLnRpbWVz
dGFsa3MuY29tggtuZXdzZGV2Lm5ldIIHbnl0LmNvbYIHbnl0Lm5ldIIJbnl0Y28u
Y29tgg50aW1lc3RhbGtzLmNvbYIbd3d3LmJlc3RzZWxsZXJzLm55dGltZXMuY29t
ghx3d3cuaG9tZWRlbGl2ZXJ5Lm55dGltZXMuY29tMIIBAwYKKwYBBAHWeQIEAgSB
9ASB8QDvAHYAu9nfvB+KcbWTlCOXqpJ7RzhXlQqrUugakJZkNo4e0YUAAAFnYAxy
sQAABAMARzBFAiBsjMEzQ01LJnfg8SWtJi+wQ/2NrVih667zOk9JD/KAxwIhAOvJ
ND92OVh2cozY7QXv0vsfzWszxn9tEVaNc3ezXlQDAHUAXqdz+d9WwOe1Nkh90Eng
MnqRmgyEoRIShBh1loFxRVgAAAFnYAxzBgAABAMARjBEAiBFD4mv+quaSJL/sb4J
b0zh1w6xe+NBCLxCgr2DLtCZIwIgICd9NO8Mj0obKpS0eB49ZNlj3J7JjKMXCQJG
GSlfyIgwDQYJKoZIhvcNAQELBQADggEBAEdGZx2Iilb59sTUqgyo92XdwUxpEXUD
25W06NhOezqUJHfw7YxsCuXdSNPAcoMgVuvEo2A4JG9skf62rBFar6sdsBy1OucP
/njdSXTN5XuTOwaxO/g4uF8iGGrdR6pYjyeh9DcaIPCPagOxMi0QLd32twYeBzRu
ZG4sc6JGdmEo9z4Xw5SHkm+x88cLuBiTlcsYgVTFhW+LFwNbILArbP+BCRCTJvOf
yevlPpxHGRs2HA/k0LUOvx6MjCj7Xk8i36OxwBkYLr5HDZ2dapAOGKy+tZrOX2z1
3u4v0J/Ctz93eUTpt9nCObmbPHxOlP3MRRQwnUyMbFh8qrnLPXjl23o=
-----END CERTIFICATE-----
`
var NytimesRawCert []byte
var NytimesX509Cert *x509.Certificate

var MockCloserCalledCounter = 0
var MockCloserFail = false

var MockError = errors.New("Mock")

type MockCloserType int
func(MockCloserType)Close() error {
    MockCloserCalledCounter++
    if MockCloserFail {
        return MockError
    } else {
        return nil
    }
}

var MockLocalMachineStore = wincrypt.HCERTSTORE(123)
var MockNyTimesCertContext *wincrypt.CERT_CONTEXT

var MockNyTimesPrivateKey crypto.PrivateKey = &struct {}{}

var MockNcryptKey = ncrypt.NCRYPT_KEY_HANDLE(uintptr(123))

func mock_wincrypt_CertOpenStore(
	lpszStoreProvider *byte,
	dwEncodingType uint32,
	hCryptProv wincrypt.HCRYPTPROV_LEGACY,
	dwFlags uint32,
	pvPara unsafe.Pointer,
) (store wincrypt.HCERTSTORE, err error) {
	if (dwFlags & wincrypt.CERT_SYSTEM_STORE_LOCAL_MACHINE > 0) {
		return MockLocalMachineStore, nil
	}
	return wincrypt.HCERTSTORE(0), MockError
}

var MockCertEnumCertificatesInStoreFail = false
func mock_wincrypt_CertEnumCertificatesInStore(
	hCertStore wincrypt.HCERTSTORE,
	pPrevCertContext *wincrypt.CERT_CONTEXT,
) (context *wincrypt.CERT_CONTEXT, e error) {
	if MockCertEnumCertificatesInStoreFail {
		return nil, MockError
	}

	if hCertStore == MockLocalMachineStore {
		if (pPrevCertContext == nil) {
			MockNyTimesCertContext = &wincrypt.CERT_CONTEXT{
				DwCertEncodingType: 0,
				PbCertEncoded:      &NytimesRawCert[0],
				CbCertEncoded:      uint32(len(NytimesRawCert)),
				PCertInfo:          nil,
				HCertStore:         0,
			}
			return MockNyTimesCertContext, nil
		} else if (pPrevCertContext == MockNyTimesCertContext) {
			return nil, io.EOF
		}
	}

	return nil, MockError
}

var MockPrivateKeyFromCertContextFail = false
func mock_privateKeyFromCertContext(certContext *wincrypt.CERT_CONTEXT, x509_cert *x509.Certificate) (privateKey crypto.PrivateKey, err error) {
	if MockPrivateKeyFromCertContextFail {
		return nil, MockError
	}

	if certContext == MockNyTimesCertContext && string(x509_cert.Raw) == string(NytimesX509Cert.Raw) {
        return MockNyTimesPrivateKey, nil
	}

	return nil, MockError
}


var MockCertCloseStoreCalledCounter = 0
var MockCertCloseFailOnStoreCheckFlagSet = false
var MockCertCloseFail = false
func mock_wincrypt_CertCloseStore(hCertStore wincrypt.HCERTSTORE, dwFlags uint32) error {
	MockCertCloseStoreCalledCounter++

	if MockCertCloseFailOnStoreCheckFlagSet && ((dwFlags & wincrypt.CERT_CLOSE_STORE_CHECK_FLAG) > 0) {
		return syscall.Errno(wincrypt.CRYPT_E_PENDING_CLOSE)
	}

	if MockCertCloseFail {
		return MockError
	}

	return nil
}

var MockCryptAcquireCertificatePrivateKeyFail = false
func mock_wincrypt_CryptAcquireCertificatePrivateKey(
	pCert *wincrypt.CERT_CONTEXT,
	dwFlags uint32,
	pvParameters unsafe.Pointer,
	phCryptProvOrNCryptKey *wincrypt.HCRYPTPROV_OR_NCRYPT_KEY_HANDLE,
	pdwKeySpec *uint32,
	pfCallerFreeProvOrNCryptKey *int,
) error {
	if MockCryptAcquireCertificatePrivateKeyFail {
		return MockError
	}

	if pCert == MockNyTimesCertContext {
		*phCryptProvOrNCryptKey = wincrypt.HCRYPTPROV_OR_NCRYPT_KEY_HANDLE(MockNcryptKey)
		*pfCallerFreeProvOrNCryptKey = 1
		*pdwKeySpec = wincrypt.CERT_NCRYPT_KEY_SPEC
		return nil
	}

	return MockError
}

var MockNCryptFreeObjectCalledCounter = 0
var MockNCryptFreeObjectFail = false
func mock_ncrypt_NCryptFreeObject(hObject ncrypt.NCRYPT_HANDLE) error {
	MockNCryptFreeObjectCalledCounter++

	if MockNCryptFreeObjectFail {
		return MockError
	}

	return nil
}

var MockNCryptDecryptCalledCounter = 0
var MockNCryptDecryptFailOnSizeCall = false
var MockNCryptDecryptFailOnDecryptCall = false
func mock_ncrypt_NCryptDecrypt(
	hKey ncrypt.NCRYPT_KEY_HANDLE,
	pbInput *byte,
	cbInput uint32,
	pPaddingInfo unsafe.Pointer,
	pbOutput *byte,
	cbOutput uint32,
	pcbResult *uint32,
	dwFlags uint32,
) int {
	MockNCryptDecryptCalledCounter++

	if hKey != MockNcryptKey {
		return 0
	}

	// For testing purposes we'll just mirror the input to output
	// followed by information about the padding used
	suffix := ""
	if dwFlags & ncrypt.NCRYPT_PAD_OAEP_FLAG > 0 {
		padding := (*ncrypt.BCRYPT_OAEP_PADDING_INFO)(pPaddingInfo)
		suffix = fmt.Sprintf(":OAEP:%s:%d", copyCBytesToSlice(padding.PbLabel, int(padding.CbLabel)), padding.PszAlgId)
	} else if dwFlags & ncrypt.NCRYPT_PAD_PKCS1_FLAG > 0 {
		suffix = fmt.Sprintf(":PKCS1")
	}

	*pcbResult = 0
	if pbOutput == nil && cbOutput == 0{
		if MockNCryptDecryptFailOnSizeCall { return 0 }

		*pcbResult = cbInput + uint32(len(suffix))
	} else {
		if MockNCryptDecryptFailOnDecryptCall { return 0 }

		i := uint32(0)
		inaddr := uintptr(unsafe.Pointer(pbInput))
		outaddr := uintptr(unsafe.Pointer(pbOutput))
		for ;i < cbInput && i < cbOutput; i, inaddr, outaddr = i+1, inaddr+1, outaddr+1 {
			*(*byte)(unsafe.Pointer(outaddr)) = *(*byte)(unsafe.Pointer(inaddr))
			(*pcbResult)++
		}

		for j := 0; j < len(suffix) && i < cbOutput; i, j, outaddr = i+1, j+1, outaddr+1 {
			*(*byte)(unsafe.Pointer(outaddr)) = suffix[j]
			(*pcbResult)++
		}
	}

	return 1
}

var MockNCryptSignHashCalledCounter = 0
var MockNCryptSignHashFailOnSizeCall = false
var MockNCryptSignHashFailOnSignCall = false
func mock_ncrypt_NCryptSignHash(
	hKey ncrypt.NCRYPT_KEY_HANDLE,
	pPaddingInfo unsafe.Pointer,
	pbHashValue *byte,
	cbHashValue uint32,
	pbSignature *byte,
	cbSignature uint32,
	pcbResult *uint32,
	dwFlags uint32,
) error {
	MockNCryptSignHashCalledCounter++

	if hKey != MockNcryptKey {
		return MockError
	}

	// For testing purposes we'll just mirror the input to output
	// followed by information about the padding used
	suffix := ""
	if dwFlags & ncrypt.NCRYPT_PAD_PSS_FLAG > 0 {
		padding := (*ncrypt.BCRYPT_PSS_PADDING_INFO)(pPaddingInfo)
		suffix = fmt.Sprintf(":PSS:%d:%d", padding.PszAlgId, padding.CbSalt)
	} else if dwFlags & ncrypt.NCRYPT_PAD_PKCS1_FLAG > 0 {
		padding := (*ncrypt.BCRYPT_PKCS1_PADDING_INFO)(pPaddingInfo)
		suffix = fmt.Sprintf(":PKCS1:%d", padding.PszAlgId)
	}

	*pcbResult = 0
	if pbSignature == nil && cbSignature == 0{
		if MockNCryptSignHashFailOnSizeCall { return MockError }
		*pcbResult = cbHashValue + uint32(len(suffix))
	} else {
		if MockNCryptSignHashFailOnSignCall { return MockError }

		i := uint32(0)
		inaddr := uintptr(unsafe.Pointer(pbHashValue))
		outaddr := uintptr(unsafe.Pointer(pbSignature))
		for ;i < cbHashValue && i < cbSignature; i, inaddr, outaddr = i+1, inaddr+1, outaddr+1 {
			*(*byte)(unsafe.Pointer(outaddr)) = *(*byte)(unsafe.Pointer(inaddr))
			(*pcbResult)++
		}

		for j := 0; j < len(suffix) && i < cbSignature; i, j, outaddr = i+1, j+1, outaddr+1 {
			*(*byte)(unsafe.Pointer(outaddr)) = suffix[j]
			(*pcbResult)++
		}
	}

	return nil
}

func InitMocks(isIntegrationTest bool) {
	pem_decoded, _ := pem.Decode([]byte(NytimesCertPem))
	NytimesRawCert = pem_decoded.Bytes
	NytimesX509Cert, _ = x509.ParseCertificate(NytimesRawCert)

	MockCertCloseStoreCalledCounter = 0
	MockCloserCalledCounter = 0
	MockCloserFail = false
	MockCertCloseFailOnStoreCheckFlagSet = false
	MockCertCloseFail = false
	MockCertEnumCertificatesInStoreFail = false
	MockPrivateKeyFromCertContextFail = false
	MockCryptAcquireCertificatePrivateKeyFail = false

	MockNCryptFreeObjectCalledCounter = 0
	MockNCryptFreeObjectFail = false

	MockNCryptDecryptCalledCounter = 0
	MockNCryptDecryptFailOnSizeCall = false
	MockNCryptDecryptFailOnDecryptCall = false

	MockNCryptSignHashCalledCounter = 0
	MockNCryptSignHashFailOnSizeCall = false
	MockNCryptSignHashFailOnSignCall = false

	MockNyTimesCertContext = &wincrypt.CERT_CONTEXT{}

	if (!isIntegrationTest) {
		wincrypt_CertOpenStore = mock_wincrypt_CertOpenStore
		wincrypt_CertEnumCertificatesInStore = mock_wincrypt_CertEnumCertificatesInStore
		wincrypt_CertCloseStore = mock_wincrypt_CertCloseStore
		wincrypt_CryptAcquireCertificatePrivateKey = mock_wincrypt_CryptAcquireCertificatePrivateKey

		ncrypt_NCryptFreeObject = mock_ncrypt_NCryptFreeObject
		ncrypt_NCryptDecrypt = mock_ncrypt_NCryptDecrypt

		ncrypt_NCryptSignHash = mock_ncrypt_NCryptSignHash

		_privateKeyFromCertContext = mock_privateKeyFromCertContext
	} else {
		wincrypt_CertOpenStore = wincrypt.CertOpenStore
		wincrypt_CertEnumCertificatesInStore = wincrypt.CertEnumCertificatesInStore
		wincrypt_CertCloseStore = wincrypt.CertCloseStore
		wincrypt_CryptAcquireCertificatePrivateKey = wincrypt.CryptAcquireCertificatePrivateKey

		ncrypt_NCryptFreeObject = ncrypt.NCryptFreeObject
		ncrypt_NCryptDecrypt = ncrypt.NCryptDecrypt

		ncrypt_NCryptSignHash = ncrypt.NCryptSignHash

		_privateKeyFromCertContext = privateKeyFromCertContext
	}
}
