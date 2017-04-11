// +build !integration

package stats

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/schema"
	"github.com/stretchr/testify/assert"
)

func TestStats(t *testing.T) {

	files, err := filepath.Glob("./_meta/test/stats.*.json")
	assert.NoError(t, err)

	for _, f := range files {
		content, err := ioutil.ReadFile(f)
		assert.NoError(t, err)

		_, errs := eventMapping(content)
		if errs == nil {
			continue
		}
		errors, ok := errs.(*schema.Errors)
		if ok {
			assert.False(t, errors.HasRequiredErrors(), "mapping error: %s", errors)
		} else {
			t.Error(err)
		}
	}
}

func TestEmptyStats(t *testing.T) {

	file := "./_meta/test/stats.512.empty.json"

	content, err := ioutil.ReadFile(file)
	assert.NoError(t, err)

	event, errs := eventMapping(content)
	assert.Equal(t, event, common.MapStr{})
	assert.Nil(t, errs)
}
