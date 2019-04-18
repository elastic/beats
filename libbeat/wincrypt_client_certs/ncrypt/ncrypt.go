package ncrypt

import (
	"fmt"
	"syscall"
	"unsafe"
)

const (
	NCRYPT_NO_PADDING_FLAG = 0x1
	NCRYPT_PAD_PKCS1_FLAG = 0x2
	NCRYPT_PAD_OAEP_FLAG = 0x4
	NCRYPT_PAD_PSS_FLAG = 0x8
	NCRYPT_PAD_CIPHER_FLAG = 0x10
	NCRYPT_CIPHER_NO_PADDING_FLAG = 0x0
	NCRYPT_CIPHER_BLOCK_PADDING_FLAG = 0x1
	NCRYPT_CIPHER_OTHER_PADDING_FLAG = 0x2

	BCRYPT_PAD_NONE =0x00000001
	BCRYPT_PAD_PKCS1 = 0x00000002
	BCRYPT_PAD_OAEP = 0x00000004
	BCRYPT_PAD_PSS = 0x00000008
	BCRYPT_PAD_PKCS1_OPTIONAL_HASH_OID = 0x00000010

	NCRYPT_SILENT_FLAG = 0x00000040
)


type NCRYPT_HANDLE uintptr
type NCRYPT_KEY_HANDLE uintptr

var BCRYPT_3DES_ALGORITHM, _ = syscall.UTF16FromString("3DES")
var BCRYPT_3DES_112_ALGORITHM, _ = syscall.UTF16FromString("3DES_112")
var BCRYPT_AES_ALGORITHM, _ = syscall.UTF16FromString("AES")
var BCRYPT_AES_CMAC_ALGORITHM, _ = syscall.UTF16FromString("AES-CMAC")
var BCRYPT_AES_GMAC_ALGORITHM, _ = syscall.UTF16FromString("AES-GMAC")
var BCRYPT_CAPI_KDF_ALGORITHM, _ = syscall.UTF16FromString("CAPI_KDF")
var BCRYPT_DES_ALGORITHM, _ = syscall.UTF16FromString("DES")
var BCRYPT_DESX_ALGORITHM, _ = syscall.UTF16FromString("DESX")
var BCRYPT_DH_ALGORITHM, _ = syscall.UTF16FromString("DH")
var BCRYPT_DSA_ALGORITHM, _ = syscall.UTF16FromString("DSA")
var BCRYPT_ECDH_P256_ALGORITHM, _ = syscall.UTF16FromString("ECDH_P256")
var BCRYPT_ECDH_P384_ALGORITHM, _ = syscall.UTF16FromString("ECDH_P384")
var BCRYPT_ECDH_P521_ALGORITHM, _ = syscall.UTF16FromString("ECDH_P521")
var BCRYPT_ECDSA_P256_ALGORITHM, _ = syscall.UTF16FromString("ECDSA_P256")
var BCRYPT_ECDSA_P384_ALGORITHM, _ = syscall.UTF16FromString("ECDSA_P384")
var BCRYPT_ECDSA_P521_ALGORITHM, _ = syscall.UTF16FromString("ECDSA_P521")
var BCRYPT_MD2_ALGORITHM, _ = syscall.UTF16FromString("MD2")
var BCRYPT_MD4_ALGORITHM, _ = syscall.UTF16FromString("MD4")
var BCRYPT_MD5_ALGORITHM, _ = syscall.UTF16FromString("MD5")
var BCRYPT_RC2_ALGORITHM, _ = syscall.UTF16FromString("RC2")
var BCRYPT_RC4_ALGORITHM, _ = syscall.UTF16FromString("RC4")
var BCRYPT_RNG_ALGORITHM, _ = syscall.UTF16FromString("RNG")
var BCRYPT_RNG_DUAL_EC_ALGORITHM, _ = syscall.UTF16FromString("DUALECRNG")
var BCRYPT_RNG_FIPS186_DSA_ALGORITHM, _ = syscall.UTF16FromString("FIPS186DSARNG")
var BCRYPT_RSA_ALGORITHM, _ = syscall.UTF16FromString("RSA")
var BCRYPT_RSA_SIGN_ALGORITHM, _ = syscall.UTF16FromString("RSA_SIGN")
var BCRYPT_SHA1_ALGORITHM, _ = syscall.UTF16FromString("SHA1")
var BCRYPT_SHA256_ALGORITHM, _ = syscall.UTF16FromString("SHA256")
var BCRYPT_SHA384_ALGORITHM, _ = syscall.UTF16FromString("SHA384")
var BCRYPT_SHA512_ALGORITHM, _ = syscall.UTF16FromString("SHA512")
var BCRYPT_SP800108_CTR_HMAC_ALGORITHM, _ = syscall.UTF16FromString("SP800_108_CTR_HMAC")
var BCRYPT_SP80056A_CONCAT_ALGORITHM, _ = syscall.UTF16FromString("SP800_56A_CONCAT")
var BCRYPT_PBKDF2_ALGORITHM, _ = syscall.UTF16FromString("PBKDF2")
var BCRYPT_ECDSA_ALGORITHM, _ = syscall.UTF16FromString("ECDSA")
var BCRYPT_ECDH_ALGORITHM, _ = syscall.UTF16FromString("ECDH")
var BCRYPT_XTS_AES_ALGORITHM, _ = syscall.UTF16FromString("XTS-AES")

type BCRYPT_OAEP_PADDING_INFO struct {
	PszAlgId *uint16
	PbLabel *byte
	CbLabel uint32
}

type BCRYPT_PKCS1_PADDING_INFO struct {
	PszAlgId *uint16
}

type BCRYPT_PSS_PADDING_INFO struct {
	PszAlgId *uint16
	CbSalt uint32;
}

var (
	ncryptDll = syscall.NewLazyDLL("Ncrypt.dll")

	nCryptFreeObject				  = ncryptDll.NewProc("NCryptFreeObject")
	nCryptDecrypt				      = ncryptDll.NewProc("NCryptDecrypt")
	nCryptSignHash 				      = ncryptDll.NewProc("NCryptSignHash")
)

func NCryptFreeObject(hObject NCRYPT_HANDLE ) (error) {
	status, _, err := nCryptFreeObject.Call(uintptr(hObject))
	if status != 0 {
		return err
	}
	return nil
}

func NCryptDecrypt(
	hKey NCRYPT_KEY_HANDLE,
	pbInput *byte,
	cbInput uint32,
	pPaddingInfo unsafe.Pointer,
	pbOutput *byte,
	cbOutput uint32,
	pcbResult *uint32,
	dwFlags uint32,
) (int) {
	status, _, _ := nCryptDecrypt.Call(
		uintptr(hKey),
		uintptr(unsafe.Pointer(pbInput)),
		uintptr(cbInput),
		uintptr(pPaddingInfo),
		uintptr(unsafe.Pointer(pbOutput)),
		uintptr(cbOutput),
		uintptr(unsafe.Pointer(pcbResult)),
		uintptr(dwFlags),
	)

	if status != 0 {
		return 0
	} else {
		return 1
	}
}

func NCryptSignHash(
	hKey NCRYPT_KEY_HANDLE,
	pPaddingInfo unsafe.Pointer,
	pbHashValue *byte,
	cbHashValue uint32,
	pbSignature *byte,
	cbSignature uint32,
	pcbResult *uint32,
	dwFlags uint32,
) (error) {
	status, _, _ := nCryptSignHash.Call(
		uintptr(hKey),
		uintptr(pPaddingInfo),
		uintptr(unsafe.Pointer(pbHashValue)),
		uintptr(cbHashValue),
		uintptr(unsafe.Pointer(pbSignature)),
		uintptr(cbSignature),
		uintptr(unsafe.Pointer(pcbResult)),
		uintptr(dwFlags),
	)

	if status != 0 {
		return fmt.Errorf("NCryptSignHash failed with error code %v", status)
	}

	return nil
}
