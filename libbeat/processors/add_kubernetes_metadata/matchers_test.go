package add_kubernetes_metadata

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

func TestFieldMatcher(t *testing.T) {
	testCfg := map[string]interface{}{
		"lookup_fields": []string{},
	}
	fieldCfg, err := common.NewConfigFrom(testCfg)

	assert.Nil(t, err)
	matcher, err := NewFieldMatcher(*fieldCfg)
	assert.NotNil(t, err)

	testCfg["lookup_fields"] = "foo"
	fieldCfg, _ = common.NewConfigFrom(testCfg)

	matcher, err = NewFieldMatcher(*fieldCfg)
	assert.NotNil(t, matcher)
	assert.Nil(t, err)

	input := common.MapStr{
		"foo": "bar",
	}

	out := matcher.MetadataIndex(input)
	assert.Equal(t, out, "bar")

	nonMatchInput := common.MapStr{
		"not": "match",
	}

	out = matcher.MetadataIndex(nonMatchInput)
	assert.Equal(t, out, "")
}

func TestFieldFormatMatcher(t *testing.T) {
	testCfg := map[string]interface{}{}
	fieldCfg, err := common.NewConfigFrom(testCfg)

	assert.Nil(t, err)
	matcher, err := NewFieldFormatMatcher(*fieldCfg)
	assert.NotNil(t, err)

	testCfg["format"] = `%{[namespace]}/%{[pod]}`
	fieldCfg, _ = common.NewConfigFrom(testCfg)

	matcher, err = NewFieldFormatMatcher(*fieldCfg)
	assert.NotNil(t, matcher)
	assert.Nil(t, err)

	event := common.MapStr{
		"namespace": "foo",
		"pod":       "bar",
	}

	out := matcher.MetadataIndex(event)
	assert.Equal(t, "foo/bar", out)

	event = common.MapStr{
		"foo": "bar",
	}
	out = matcher.MetadataIndex(event)
	assert.Empty(t, out)

	testCfg["format"] = `%{[dimensions.namespace]}/%{[dimensions.pod]}`
	fieldCfg, _ = common.NewConfigFrom(testCfg)
	matcher, err = NewFieldFormatMatcher(*fieldCfg)
	assert.NotNil(t, matcher)
	assert.Nil(t, err)

	event = common.MapStr{
		"dimensions": common.MapStr{
			"pod":       "bar",
			"namespace": "foo",
		},
	}

	out = matcher.MetadataIndex(event)
	assert.Equal(t, "foo/bar", out)
}
