package kafka

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/sarama/mocks"
)

func TestClient(t *testing.T) {
	logger := logp.NewTestingLogger(t, "")
	cfg, err := config.NewConfigFrom(map[string]interface{}{
		"hosts":   []string{"localhost:9094"},
		"topic":   "testTopic",
		"timeout": "1s",
	})
	require.NoError(t, err, "could not create config from map")

	outGrup, err := makeKafka(
		nil,
		beat.Info{
			Beat:        "libbeat",
			IndexPrefix: "testbeat",
			Logger:      logger},
		outputs.NewNilObserver(), cfg)
	if err != nil {
		t.Fatal(err)
	}

	c, ok := outGrup.Clients[0].(*client)
	require.Truef(t, ok, "Expected output to be of type %T", &client{})

	c.producer = mocks.NewAsyncProducer(t, nil)
	go c.successWorker(c.producer.Successes())
	go c.errorWorker(c.producer.Errors())

}
