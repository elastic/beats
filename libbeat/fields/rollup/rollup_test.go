package rollup

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

func TestGenerate(t *testing.T) {
	fields, err := common.LoadFieldsYamlNoKeys("testdata/fields.yml")
	assert.NoError(t, err)

	processor := NewProcessor()
	err = processor.Process(fields, "system")
	assert.NoError(t, err)

	rollup := processor.Generate()
	data, err := rollup.GetValue("groups.terms.fields")
	assert.NoError(t, err)

	assert.Equal(t, []string{"beat.name", "metricset.name", "metricset.module", "system.network"}, data)
}
