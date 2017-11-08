package keystore

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

var secret = []byte("mysecret")

func TestGet(t *testing.T) {
	s := NewSecureString(secret)
	v, err := s.Get()
	assert.Equal(t, secret, v)
	assert.Nil(t, err)
}

func TestStringMarshalingS(t *testing.T) {
	s := NewSecureString(secret)
	v := fmt.Sprintf("%s", s)

	assert.Equal(t, v, "<SecureString>")
}

func TestStringMarshalingF(t *testing.T) {
	s := NewSecureString(secret)
	v := fmt.Sprintf("%v", s)

	assert.Equal(t, v, "<SecureString>")
}

func TestStringGoStringerMarshaling(t *testing.T) {
	s := NewSecureString(secret)
	v := fmt.Sprintf("%#v", s)

	assert.Equal(t, v, "<SecureString>")
}
