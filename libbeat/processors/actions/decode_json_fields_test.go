package actions

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

var fields = [1]string{"msg"}
var testConfig, _ = common.NewConfigFrom(map[string]interface{}{
	"fields":       fields,
	"processArray": false,
})

func TestMissingKey(t *testing.T) {
	input := common.MapStr{
		"pipeline": "us1",
	}

	actual := getActualValue(t, testConfig, input)

	expected := common.MapStr{
		"pipeline": "us1",
	}

	assert.Equal(t, expected.String(), actual.String())
}

func TestFieldNotString(t *testing.T) {
	input := common.MapStr{
		"msg":      123,
		"pipeline": "us1",
	}

	actual := getActualValue(t, testConfig, input)

	expected := common.MapStr{
		"msg":      123,
		"pipeline": "us1",
	}

	assert.Equal(t, expected.String(), actual.String())
}

func TestInvalidJSON(t *testing.T) {
	input := common.MapStr{
		"msg":      "{\"log\":\"{\\\"level\\\":\\\"info\\\"}\",\"stream\":\"stderr\",\"count\":3",
		"pipeline": "us1",
	}

	actual := getActualValue(t, testConfig, input)

	expected := common.MapStr{
		"msg":      "{\"log\":\"{\\\"level\\\":\\\"info\\\"}\",\"stream\":\"stderr\",\"count\":3",
		"pipeline": "us1",
	}
	assert.Equal(t, expected.String(), actual.String())
}

func TestInvalidJSONMultiple(t *testing.T) {
	input := common.MapStr{
		"msg":      "11:38:04,323 |-INFO testing",
		"pipeline": "us1",
	}

	actual := getActualValue(t, testConfig, input)

	expected := common.MapStr{
		"msg":      "11:38:04,323 |-INFO testing",
		"pipeline": "us1",
	}
	assert.Equal(t, expected.String(), actual.String())
}

func TestValidJSONDepthOne(t *testing.T) {
	input := common.MapStr{
		"msg":      "{\"log\":\"{\\\"level\\\":\\\"info\\\"}\",\"stream\":\"stderr\",\"count\":3}",
		"pipeline": "us1",
	}

	actual := getActualValue(t, testConfig, input)

	expected := common.MapStr{
		"msg": map[string]interface{}{
			"log":    "{\"level\":\"info\"}",
			"stream": "stderr",
			"count":  3,
		},
		"pipeline": "us1",
	}

	assert.Equal(t, expected.String(), actual.String())
}

func TestValidJSONDepthTwo(t *testing.T) {
	input := common.MapStr{
		"msg":      "{\"log\":\"{\\\"level\\\":\\\"info\\\"}\",\"stream\":\"stderr\",\"count\":3}",
		"pipeline": "us1",
	}

	testConfig, _ = common.NewConfigFrom(map[string]interface{}{
		"fields":        fields,
		"process_array": false,
		"max_depth":     2,
	})

	actual := getActualValue(t, testConfig, input)

	expected := common.MapStr{
		"msg": map[string]interface{}{
			"log": map[string]interface{}{
				"level": "info",
			},
			"stream": "stderr",
			"count":  3,
		},
		"pipeline": "us1",
	}

	assert.Equal(t, expected.String(), actual.String())
}

func TestTargetOption(t *testing.T) {
	input := common.MapStr{
		"msg":      "{\"log\":\"{\\\"level\\\":\\\"info\\\"}\",\"stream\":\"stderr\",\"count\":3}",
		"pipeline": "us1",
	}

	testConfig, _ = common.NewConfigFrom(map[string]interface{}{
		"fields":        fields,
		"process_array": false,
		"max_depth":     2,
		"target":        "doc",
	})

	actual := getActualValue(t, testConfig, input)

	expected := common.MapStr{
		"doc": map[string]interface{}{
			"log": map[string]interface{}{
				"level": "info",
			},
			"stream": "stderr",
			"count":  3,
		},
		"msg":      "{\"log\":\"{\\\"level\\\":\\\"info\\\"}\",\"stream\":\"stderr\",\"count\":3}",
		"pipeline": "us1",
	}

	assert.Equal(t, expected.String(), actual.String())
}

func TestTargetRootOption(t *testing.T) {
	input := common.MapStr{
		"msg":      "{\"log\":\"{\\\"level\\\":\\\"info\\\"}\",\"stream\":\"stderr\",\"count\":3}",
		"pipeline": "us1",
	}

	testConfig, _ = common.NewConfigFrom(map[string]interface{}{
		"fields":        fields,
		"process_array": false,
		"max_depth":     2,
		"target":        "",
	})

	actual := getActualValue(t, testConfig, input)

	expected := common.MapStr{
		"log": map[string]interface{}{
			"level": "info",
		},
		"stream":   "stderr",
		"count":    3,
		"msg":      "{\"log\":\"{\\\"level\\\":\\\"info\\\"}\",\"stream\":\"stderr\",\"count\":3}",
		"pipeline": "us1",
	}

	assert.Equal(t, expected.String(), actual.String())
}

func getActualValue(t *testing.T, config *common.Config, input common.MapStr) common.MapStr {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})
	}

	p, err := newDecodeJSONFields(config)
	if err != nil {
		logp.Err("Error initializing decode_json_fields")
		t.Fatal(err)
	}

	actual, _ := p.Run(&beat.Event{Fields: input})
	return actual.Fields
}
