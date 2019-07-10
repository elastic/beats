package wincrypt_client_certs

import (
	"crypto"
	"crypto/rsa"
	"crypto/subtle"
	"errors"
	"fmt"
	"github.com/elastic/beats/libbeat/wincrypt_client_certs/ncrypt"
	"io"
	"unsafe"
)

type NcyptKey struct {
	private_key ncrypt.NCRYPT_KEY_HANDLE
	public_key	crypto.PublicKey
	freeFlag     bool
}

var _ crypto.PrivateKey = &NcyptKey{}
var _ crypto.Signer = &NcyptKey{}
var _ crypto.Decrypter = &NcyptKey{}

// functions to be monkey patched in test classes
var ncrypt_NCryptFreeObject = ncrypt.NCryptFreeObject
var ncrypt_NCryptDecrypt = ncrypt.NCryptDecrypt
var ncrypt_NCryptSignHash = ncrypt.NCryptSignHash

func (k *NcyptKey) Close() (error) {
	if uintptr(k.private_key) != 0 && k.freeFlag {
	 	err := ncrypt_NCryptFreeObject(ncrypt.NCRYPT_HANDLE(k.private_key))
	 	k.private_key = ncrypt.NCRYPT_KEY_HANDLE(uintptr(0))
	 	if err != nil { return err }
	}
	return nil
}


func (k *NcyptKey) Decrypt(rand io.Reader, msg []byte, opts crypto.DecrypterOpts) (plaintext []byte, err error) {
	if opts == nil {
		return k.decryptPKCS1(msg)
	}

	switch opts := opts.(type) {
	case *rsa.OAEPOptions:
		return k.decryptOAEP(msg, opts.Hash, opts.Label)

	case *rsa.PKCS1v15DecryptOptions:
		if l := opts.SessionKeyLen; l > 0 {
			plaintext = make([]byte, l)
			if _, err := io.ReadFull(rand, plaintext); err != nil {
				return nil, err
			}
			if err := k.decryptSessionKey(msg, plaintext); err != nil {
				return nil, err
			}
			return plaintext, nil
		} else {
			return k.decryptPKCS1(msg)
		}

	default:
		return nil, errors.New("wincrypt/ncrypt: invalid options for Decrypt")
	}
}

func (k *NcyptKey) decryptPKCS1(msg []byte) (plaintext []byte, err error) {
	if checkSafeCastToUint32(len(msg)) != nil {
		return nil, errors.New("wincrypt/ncrypt: failed to decrypt pkcs1 encoded msg")
	}

	var plaintextLen uint32

	valid := ncrypt_NCryptDecrypt(
		k.private_key,
		&msg[0],
		uint32(len(msg)),
		unsafe.Pointer(uintptr(0)),
		nil,
		uint32(0),
		&plaintextLen,
		ncrypt.NCRYPT_PAD_PKCS1_FLAG,
	)

	if valid != 1 {
		return nil, errors.New("wincrypt/ncrypt: failed to decrypt pkcs1 encoded msg")
	}

	if plaintextLen == 0 {
		return []byte{}, nil
	}

	buffer := make([]byte, plaintextLen)

	// decrypt message to allocated buffer
	valid = ncrypt_NCryptDecrypt(
		k.private_key,
		&msg[0],
		uint32(len(msg)),
		unsafe.Pointer(uintptr(0)),
		&buffer[0],
		uint32(len(buffer)),
		&plaintextLen,
		ncrypt.NCRYPT_PAD_PKCS1_FLAG,
	)

	if valid != 1 {
		return nil, errors.New("wincrypt/ncrypt: failed to decrypt pkcs1 encoded msg")
	}

	return buffer[0:plaintextLen], nil
}

func (k *NcyptKey) decryptOAEP(msg []byte, hash crypto.Hash, label []byte) (plaintext []byte, err error) {
	if checkSafeCastToUint32(len(msg)) != nil {
		return nil, errors.New("wincrypt/ncrypt: failed to decrypt oeap encoded msg")
	}

	var plaintextLen uint32 = 0

	algId, err := hashToBcryptAlgorithm(hash)
	if err != nil { return nil, err }

	var pbLabel *byte
	if len(label) == 0 {
		pbLabel = nil
	} else {
		pbLabel = (*byte)(unsafe.Pointer(&label[0]))
	}

	padding_info := ncrypt.BCRYPT_OAEP_PADDING_INFO{
		PszAlgId: &algId[0],
		PbLabel: pbLabel,
		CbLabel: uint32(len(label)),
	}

	// get needed length for buffer
	valid := ncrypt_NCryptDecrypt(
		k.private_key,
		&msg[0],
		uint32(len(msg)),
		unsafe.Pointer(&padding_info),
		nil,
		0,
		&plaintextLen,
		ncrypt.NCRYPT_PAD_OAEP_FLAG,
	)

	if valid != 1 {
		return nil, errors.New("wincrypt/ncrypt: failed to decrypt oeap encoded msg")
	}

	if plaintextLen == 0 {
		return []byte{}, nil
	}

	buffer := make([]byte, plaintextLen)

	// decrypt message to allocated buffer
	valid = ncrypt_NCryptDecrypt(
		k.private_key,
		&msg[0],
		uint32(len(msg)),
		unsafe.Pointer(&padding_info),
		&buffer[0],
		uint32(len(buffer)),
		&plaintextLen,
		ncrypt.NCRYPT_PAD_OAEP_FLAG,
	)

	if valid != 1 {
		return nil, errors.New("wincrypt/ncrypt: failed to decrypt oeap encoded msg")
	}

	return buffer[0:plaintextLen], nil
}

func (k *NcyptKey)decryptSessionKey(msg []byte, key []byte) error {
	if checkSafeCastToUint32(len(msg)) != nil {
		return errors.New("wincrypt/ncrypt: failed to decrypt session key")
	}

	em := make([]byte, len(key))
	var plaintextLen uint32 = 0

	valid := ncrypt_NCryptDecrypt(
		k.private_key,
		&msg[0],
		uint32(len(msg)),
		unsafe.Pointer(uintptr(0)),
		&em[0],
		uint32(len(em)),
		&plaintextLen,
		ncrypt.NCRYPT_PAD_PKCS1_FLAG,
	)

	valid &= subtle.ConstantTimeEq(int32(plaintextLen), int32(len(key)))
	subtle.ConstantTimeCopy(valid, key, em[0:len(key)])

	return nil
}

func hashToBcryptAlgorithm(hash crypto.Hash)([]uint16, error) {
		switch hash {
		case crypto.MD4:
			return ncrypt.BCRYPT_MD4_ALGORITHM, nil
		case crypto.MD5:
			return ncrypt.BCRYPT_MD5_ALGORITHM, nil
		case crypto.SHA1:
			return ncrypt.BCRYPT_SHA1_ALGORITHM, nil
		case crypto.SHA224:
			return nil, errors.New("wincrypt/ncrypt: sha224 hashing isn't supported")
		case crypto.SHA256:
			return ncrypt.BCRYPT_SHA256_ALGORITHM, nil
		case crypto.SHA384:
			return ncrypt.BCRYPT_SHA384_ALGORITHM, nil
		case crypto.SHA512:
			return ncrypt.BCRYPT_SHA512_ALGORITHM, nil
		case crypto.MD5SHA1:
			return nil, errors.New("wincrypt/ncrypt: md5sha1 hashing isn't supported")
		case crypto.RIPEMD160:
			return nil, errors.New("wincrypt/ncrypt: ripemd160 hashing isn't supported")
		case crypto.SHA3_224:
			return nil, errors.New("wincrypt/ncrypt: sha3_224 hashing isn't supported")
		case crypto.SHA3_256:
			return nil, errors.New("wincrypt/ncrypt: sha3_256 hashing isn't supported")
		case crypto.SHA3_384:
			return nil, errors.New("wincrypt/ncrypt: sha3_384 hashing isn't supported")
		case crypto.SHA3_512:
			return nil, errors.New("wincrypt/ncrypt: sha3_512 hashing isn't supported")
		case crypto.SHA512_224:
			return nil, errors.New("wincrypt/ncrypt: sha512_224 hashing isn't supported")
		case crypto.SHA512_256:
			return nil, errors.New("wincrypt/ncrypt: sha512_256 hashing isn't supported")
		case crypto.BLAKE2s_256:
			return nil, errors.New("wincrypt/ncrypt: blake2s_256 hashing isn't supported")
		case crypto.BLAKE2b_256:
			return nil, errors.New("wincrypt/ncrypt: blake2b_256 hashing isn't supported")
		case crypto.BLAKE2b_384:
			return nil, errors.New("wincrypt/ncrypt: blake2b_256 hashing isn't supported")
		case crypto.BLAKE2b_512:
			return nil, errors.New("wincrypt/ncrypt: blake2b_512 hashing isn't supported")
		default:
			return nil, fmt.Errorf("wincrypt/ncrypt: unknown hashing algorithm %v", hash)
	}
}

func (k *NcyptKey) Public() crypto.PublicKey {
	return k.public_key
}

func (k *NcyptKey) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) (signature []byte, err error) {
	if pssOpts, ok := opts.(*rsa.PSSOptions); ok {
		return k.signPSS(digest, pssOpts)
	}

	return k.signPKCS1(digest, opts.HashFunc())
}

func (k *NcyptKey) signPSS(digest []byte, pssOpts *rsa.PSSOptions)(signature []byte, err error) {
	if checkSafeCastToUint32(len(digest)) != nil {
		return nil, errors.New("wincrypt/ncrypt: failed to sign with pss")
	}

	bytesWritten := uint32(0)

	algId, err := hashToBcryptAlgorithm(pssOpts.Hash)
	if err != nil { return nil, err }

	paddingInfo := ncrypt.BCRYPT_PSS_PADDING_INFO{
		PszAlgId: &algId[0],
		CbSalt: uint32(pssOpts.SaltLength),
	}

	err = ncrypt_NCryptSignHash(
		k.private_key,
		unsafe.Pointer(&paddingInfo),
		&digest[0],
		uint32(len(digest)),
		nil,
		uint32(0),
		&bytesWritten,
		ncrypt.BCRYPT_PAD_PSS,
	)

	if err != nil {
		return nil, errors.New("wincrypt/ncrypt: failed to sign with pss")
	}

	if bytesWritten == 0 {
		return []byte{}, nil
	}
	buffer := make([]byte, bytesWritten)

	err = ncrypt_NCryptSignHash(
		k.private_key,
		unsafe.Pointer(&paddingInfo),
		&digest[0],
		uint32(len(digest)),
		&buffer[0],
		uint32(len(buffer)),
		&bytesWritten,
		ncrypt.BCRYPT_PAD_PSS,
	)

	if err != nil {
		return nil, errors.New("wincrypt/ncrypt: failed to sign with pss")
	}

	return buffer[0:bytesWritten], nil
}


func (k *NcyptKey) signPKCS1(digest []byte, hash crypto.Hash)(signature []byte, err error) {
	if checkSafeCastToUint32(len(digest)) != nil {
		return nil, errors.New("wincrypt/ncrypt: failed to sign with pkcs1")
	}

	bytesWritten := uint32(0)

	algId, err := hashToBcryptAlgorithm(hash)
	if err != nil { return nil, err }

	paddingInfo := ncrypt.BCRYPT_PKCS1_PADDING_INFO{&algId[0]}

	err = ncrypt_NCryptSignHash(
		k.private_key,
		unsafe.Pointer(&paddingInfo),
		&digest[0],
		uint32(len(digest)),
		nil,
		uint32(0),
		&bytesWritten,
		ncrypt.BCRYPT_PAD_PKCS1,
	)

	if err != nil {
		return nil, errors.New("wincrypt/ncrypt: failed to sign with pkcs1")
	}

	if bytesWritten == 0 {
		return []byte{}, nil
	}
	buffer := make([]byte, bytesWritten)

	err = ncrypt_NCryptSignHash(
		k.private_key,
		unsafe.Pointer(&paddingInfo),
		&digest[0],
		uint32(len(digest)),
		&buffer[0],
		uint32(len(buffer)),
		&bytesWritten,
		ncrypt.BCRYPT_PAD_PKCS1,
	)

	if err != nil {
		return nil, errors.New("wincrypt/ncrypt: failed to sign with pkcs1")
	}

	return buffer[0:bytesWritten], nil
}

func checkSafeCastToUint32(i int) (error) {
	if i != int(uint32(i)) {
		return errors.New("wincrypt/ncrypt: failed to typecast int to uint32 (overflow)")
	} else {
		return nil
	}
}
