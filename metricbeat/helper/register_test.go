// +build !integration

package helper

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/stretchr/testify/assert"
)

func TestGetModuleInvalid(t *testing.T) {

	config, _ := common.NewConfigFrom(ModuleConfig{
		Module: "test",
	})

	registry := Register{}

	module, err := registry.GetModule(config)

	assert.Nil(t, module)
	assert.Error(t, err)
}
