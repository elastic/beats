package actions

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

func TestCommonPaths(t *testing.T) {
	var tests = []struct {
		Value, Field, Separator, Target, Result string
		Index                                   int
	}{
		// Common docker case
		{
			Value:     "/var/lib/docker/containers/f1510836197d7c34da22cf796dba5640f87c04de5c95cf0adc11b85f1e1c1528/f1510836197d7c34da22cf796dba5640f87c04de5c95cf0adc11b85f1e1c1528-json.log",
			Field:     "source",
			Separator: "/",
			Target:    "docker.container.id",
			Index:     4,
			Result:    "f1510836197d7c34da22cf796dba5640f87c04de5c95cf0adc11b85f1e1c1528",
		},
		{
			Value:     "/var/lib/foo/bar",
			Field:     "other_field",
			Separator: "/",
			Target:    "destination",
			Index:     3,
			Result:    "bar",
		},
		{
			Value:     "-var-lib-foo-bar",
			Field:     "source",
			Separator: "-",
			Target:    "destination",
			Index:     2,
			Result:    "foo",
		},
		{
			Value:     "*var*lib*foo*bar",
			Field:     "source",
			Separator: "*",
			Target:    "destination",
			Index:     0,
			Result:    "var",
		},
	}

	for _, test := range tests {
		var testConfig, _ = common.NewConfigFrom(map[string]interface{}{
			"field":     test.Field,
			"separator": test.Separator,
			"index":     test.Index,
			"target":    test.Target,
		})

		// Configure input to
		input := common.MapStr{
			test.Field: test.Value,
		}

		actual := runExtractField(t, testConfig, input)

		result, err := actual.GetValue(test.Target)
		if err != nil {
			t.Fatalf("could not get target field: %s", err)
		}
		assert.Equal(t, result.(string), test.Result)
	}
}

func runExtractField(t *testing.T, config *common.Config, input common.MapStr) common.MapStr {
	logp.TestingSetup()

	p, err := NewExtractField(config)
	if err != nil {
		t.Fatalf("error initializing extract_field: %s", err)
	}

	actual, err := p.Run(&beat.Event{Fields: input})
	if err != nil {
		t.Fatalf("error running extract_field: %s", err)
	}

	return actual.Fields
}
