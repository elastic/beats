package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetModuleInvalid(t *testing.T) {

	registry := Register{}

	config := ModuleConfig{
		Module: "test",
	}
	module, err := registry.GetModule(config)

	assert.Nil(t, module)
	assert.Error(t, err)
}
