package add_locale

import (
	"regexp"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher/beat"
	"github.com/stretchr/testify/assert"
)

func TestExportTimezone(t *testing.T) {
	testConfig, err := common.NewConfigFrom(map[string]interface{}{
		"format": "abbreviation",
	})
	if err != nil {
		t.Fatal(err)
	}

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

func TestTimezoneFormat(t *testing.T) {
	// Test positive format

	posLoc, err := time.LoadLocation("Africa/Asmara")
	if err != nil {
		t.Fatal(err)
	}

	posZone, posOffset := time.Now().In(posLoc).Zone()

	posAddLocal := addLocale{TimezoneFormat: Offset}

	posVal := posAddLocal.Format(posZone, posOffset)

	assert.Regexp(t, regexp.MustCompile(`\+[\d]{2}\:[\d]{2}`), posVal)

	// Test negative format

	negLoc, err := time.LoadLocation("America/Curacao")
	if err != nil {
		t.Fatal(err)
	}

	negZone, negOffset := time.Now().In(negLoc).Zone()

	negAddLocal := addLocale{TimezoneFormat: Offset}

	negVal := negAddLocal.Format(negZone, negOffset)

	assert.Regexp(t, regexp.MustCompile(`\-[\d]{2}\:[\d]{2}`), negVal)
}

func getActualValue(t *testing.T, config *common.Config, input common.MapStr) common.MapStr {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})
	}

	p, err := newAddLocale(config)
	if err != nil {
		logp.Err("Error initializing add_locale")
		t.Fatal(err)
	}

	actual, err := p.Run(&beat.Event{Fields: input})
	return actual.Fields
}

func BenchmarkConstruct(b *testing.B) {
	var testConfig = common.NewConfig()

	input := common.MapStr{}

	p, err := newAddLocale(testConfig)
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		_, err = p.Run(&beat.Event{Fields: input})
	}
}
