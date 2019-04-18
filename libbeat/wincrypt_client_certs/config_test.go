package wincrypt_client_certs

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConfig_Validate(t *testing.T) {
	config := &Config{
		Stores: []string{"CurrentUser/My", "localmachine/root"},
		Query: "true",
	}
	assert.NoError(t, config.Validate())

	config = &Config{
		Stores: []string{"bogus/test"},
		Query: "true",
	}
	assert.Error(t, config.Validate())

	config = &Config{
		Stores: []string{"CurrentUser/My"},
		Query: "A > 0 && B < 123",
	}
	assert.NoError(t, config.Validate())

	config = &Config{
		Stores: []string{"CurrentUser/My"},
		Query: "0 B",
	}
	assert.Error(t, config.Validate())
}
