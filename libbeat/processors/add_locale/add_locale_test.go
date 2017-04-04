package actions

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/stretchr/testify/assert"
)

func TestExportTimeZone(t *testing.T) {
	var testConfig, _ = common.NewConfigFrom(map[string]interface{}{
		"timezone": "America/Curacao",
	})

	input := common.MapStr{}

	actual := getActualValue(t, testConfig, input)

	expected := common.MapStr{
		"beat": map[string]string{
			"timezone": "America/Curacao",
		},
	}

	assert.Equal(t, expected.String(), actual.String())
}

func TestExportDefaultTimeZone(t *testing.T) {
	var testConfig, _ = common.NewConfigFrom(map[string]interface{}{
		"timezone": "",
	})
	input := common.MapStr{}

	actual := getActualValue(t, testConfig, input)

	expected := common.MapStr{
		"beat": map[string]string{
			"timezone": "UTC",
		},
	}

	assert.Equal(t, expected.String(), actual.String())
}

func getActualValue(t *testing.T, config *common.Config, input common.MapStr) common.MapStr {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})
	}

	p, err := newAddLocale(*config)
	if err != nil {
		logp.Err("Error initializing add_locale")
		t.Fatal(err)
	}

	actual, err := p.Run(input)

	return actual
}
