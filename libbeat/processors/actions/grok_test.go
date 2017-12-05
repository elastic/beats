package actions

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/stretchr/testify/assert"
)

var field = "msg"
var testGrokConfig, _ = common.NewConfigFrom(map[string]interface{}{
	"field":    "msg",
	"patterns": []string{`(?P<timestamp>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z),(?P<client_ip>\d+\.\d+\.\d+\.\d+)?`},
})

func TestGrokMissingKey(t *testing.T) {
	input := common.MapStr{
		"datacenter": "watson",
	}

	actual := getGrokActualValue(t, testGrokConfig, input)

	expected := common.MapStr{
		"datacenter": "watson",
	}

	assert.Equal(t, expected.String(), actual.String())
}

func TestGrokSimpleMessage(t *testing.T) {
	input := common.MapStr{
		"datacenter": "watson",
		"msg":        "2012-03-04T22:33:01.003Z,127.0.0.1",
	}

	actual := getGrokActualValue(t, testGrokConfig, input)

	expected := common.MapStr{
		"datacenter": "watson",
		"msg":        "2012-03-04T22:33:01.003Z,127.0.0.1",
		"timestamp":  "2012-03-04T22:33:01.003Z",
		"client_ip":  "127.0.0.1",
	}

	assert.Equal(t, expected.String(), actual.String())
}

func getGrokActualValue(t *testing.T, config *common.Config, input common.MapStr) common.MapStr {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})
	}

	p, err := newGrok(*config)
	if err != nil {
		logp.Err("Error initializing Grok ")
		t.Fatal(err)
	}

	actual, err := p.Run(input)

	return actual
}
