package tls

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	EmptyMD5    = "d41d8cd98f00b204e9800998ecf8427e"
	EmptySHA1   = "da39a3ee5e6b4b0d3255bfef95601890afd80709"
	EmptySHA256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
)

func TestGetFingerprintAlgorithm(t *testing.T) {
	for _, testCase := range []struct {
		requested, name string
		sum             string
	}{
		{"md5", "md5", EmptyMD5},
		{"sha1", "sha1", EmptySHA1},
		{"SHA-1", "sha1", EmptySHA1},
		{"SHA256", "sha256", EmptySHA256},
		{"sha-256", "sha256", EmptySHA256},
		{"md4", "", ""},
	} {
		result, err := GetFingerprintAlgorithm(testCase.requested)
		if len(testCase.name) == 0 {
			assert.Error(t, err, testCase.requested)
			continue
		}
		assert.Equal(t, nil, err, testCase.requested)
		assert.NotNil(t, result)
		assert.Equal(t, testCase.name, result.name)
		assert.Equal(t, testCase.sum, result.algo.Hash(nil))
	}
}
