// +build !integration

package node

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	s "github.com/elastic/beats/libbeat/common/schema"

	"github.com/stretchr/testify/assert"
)

func TestGetMappings(t *testing.T) {
	files, err := filepath.Glob("./_meta/test/node.*.json")
	assert.NoError(t, err)

	for _, f := range files {
		content, err := ioutil.ReadFile(f)
		assert.NoError(t, err)

		_, errs := eventsMapping(content)
		if errs == nil {
			continue
		}
		errors, ok := errs.(*s.Errors)
		if ok {
			assert.False(t, errors.HasRequiredErrors(), "mapping error: %s", errors)
		} else {
			t.Error(err)
		}
	}
}

func TestInvalid(t *testing.T) {
	file := "./_meta/test/invalid.json"

	content, err := ioutil.ReadFile(file)
	assert.NoError(t, err)

	_, errs := eventsMapping(content)
	errors, ok := errs.(*s.Errors)
	if ok {
		assert.True(t, errors.HasRequiredErrors(), "mapping error: %s", errors)
	} else {
		t.Error(err)
	}
}
