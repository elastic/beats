package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/urso/ucfg"
)

func TestGetModuleInvalid(t *testing.T) {

	config, _ := ucfg.NewFrom(ModuleConfig{
		Module: "test",
	})

	registry := Register{}

	module, err := registry.GetModule(config)

	assert.Nil(t, module)
	assert.Error(t, err)
}
