package actions

import (
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/stretchr/testify/assert"
)

func TestExportTimeZone(t *testing.T) {
	var testConfig = common.NewConfig()

	input := common.MapStr{}

	zone, _ := time.Now().In(time.Local).Zone()

	actual := getActualValue(t, testConfig, input)

	expected := common.MapStr{
		"beat": map[string]string{
			"timezone": zone,
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
