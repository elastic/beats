package keystore

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	ucfg "github.com/elastic/go-ucfg"
)

func TestResolverWhenTheKeyDoesntExist(t *testing.T) {
	path := GetTemporaryKeystoreFile()
	defer os.Remove(path)

	keystore := CreateAnExistingKeystore(path)

	resolver := ResolverWrap(keystore)
	_, err := resolver("donotexist")
	assert.Equal(t, err, ucfg.ErrMissing)
}

func TestResolverWhenTheKeyExist(t *testing.T) {
	path := GetTemporaryKeystoreFile()
	defer os.Remove(path)

	keystore := CreateAnExistingKeystore(path)

	resolver := ResolverWrap(keystore)
	v, err := resolver("output.elasticsearch.password")
	assert.NoError(t, err)
	assert.Equal(t, v, "secret")
}
