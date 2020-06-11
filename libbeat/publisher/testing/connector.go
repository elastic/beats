package testing

import "github.com/elastic/beats/v7/libbeat/beat"

type FakeConnector struct {
	ConnectFunc func(beat.ClientConfig) (beat.Client, error)
}

type FakeClient struct {
	PublishFunc func(beat.Event)
	CloseFunc   func() error
}

var _ beat.PipelineConnector = FakeConnector{}
var _ beat.Client = (*FakeClient)(nil)

func (c FakeConnector) ConnectWith(cfg beat.ClientConfig) (beat.Client, error) {
	return c.ConnectFunc(cfg)
}

func (c FakeConnector) Connect() (beat.Client, error) {
	return c.ConnectWith(beat.ClientConfig{})
}

func (c *FakeClient) Publish(event beat.Event) {
	if c.PublishFunc != nil {
		c.PublishFunc(event)
	}
}

func (c *FakeClient) Close() error {
	if c.CloseFunc == nil {
		return nil
	}
	return c.CloseFunc()
}

func (c *FakeClient) PublishAll(events []beat.Event) {
	for _, event := range events {
		c.Publish(event)
	}
}
