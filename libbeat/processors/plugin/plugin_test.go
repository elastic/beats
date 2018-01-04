package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

func TestFoo(t *testing.T) {
	testConfig, err := common.NewConfigFrom(map[string]interface{}{
		"file": "foo_plugin.so",
	})
	if err != nil {
		t.Fatal(err)
	}

	logp.TestingSetup()

	p, err := newPlugin(testConfig)

	if err != nil {
		logp.Err("Error initializing plugin")
		t.Fatal(err)
	}

	event, err := p.Run(&beat.Event{Fields: common.MapStr{}})

	expected := common.MapStr{
		"foo": "bar",
	}

	assert.Equal(t, expected.String(), event.Fields.String())
}

func BenchmarkConstruct(b *testing.B) {
	var testConfig = common.NewConfig()

	input := common.MapStr{}

	p, err := newPlugin(testConfig)
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		_, err = p.Run(&beat.Event{Fields: input})
	}
}
