package tls

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"strings"
)

// AlgorithmFactory represents a factory method for a hash algorithm.
type AlgorithmFactory func() hash.Hash

// FingerprintAlgorithm associates a hash name with its factory method.
type FingerprintAlgorithm struct {
	name string
	algo AlgorithmFactory
}

var hashMap = make(map[string]*FingerprintAlgorithm)
var hashNames []string

func init() {
	registerAlgo(func() hash.Hash { return md5.New() }, "md5", "")
	registerAlgo(func() hash.Hash { return sha1.New() }, "sha1", "sha-1")
	registerAlgo(func() hash.Hash { return sha256.New() }, "sha256", "sha-256")
}

func registerAlgo(fn AlgorithmFactory, name string, alias string) {
	algo := &FingerprintAlgorithm{
		name: name,
		algo: fn,
	}
	hashMap[strings.ToLower(name)] = algo
	hashNames = append(hashNames, name)
	if len(alias) != 0 {
		hashMap[strings.ToLower(alias)] = algo
	}
}

// GetFingerprintAlgorithm returns a FingerprintAlgorithm by name, or an
// error if the algorithm is not supported.
func GetFingerprintAlgorithm(name string) (*FingerprintAlgorithm, error) {
	if hasher, found := hashMap[strings.ToLower(name)]; found {
		return hasher, nil
	}
	return nil, fmt.Errorf("fingerprint algorithm '%s' not found. Use one of %v", name, hashNames)
}

// Hash returns the hash of the given data in hexadecimal format.
func (algo AlgorithmFactory) Hash(data []byte) string {
	hash := algo()
	hash.Write(data) // according to docs "never returns an error"
	return hex.EncodeToString(hash.Sum(nil))
}
