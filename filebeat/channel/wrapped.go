package channel

import (
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

// A wrapper around a generic Outleter that applies the given transformation to
// incoming events before forwarding them.
type wrappedOutlet struct {
	outlet         Outleter
	eventTransform func(beat.Event)
}

// WrapOutlet takes an Outleter and an event transformation function and
// returns an Outleter that applies that transformation before forwarding them.
// The transformation operates in place (it modifies its input events).
// The new Outleter uses the same underlying state, e.g. calling Close on the
// wrapped Outleter will close the original as well. If this is not the intent,
// call SubOutlet first.
func WrapOutlet(outlet Outleter, eventTransform func(beat.Event)) Outleter {
	return &wrappedOutlet{outlet: outlet, eventTransform: eventTransform}
}

func (o *wrappedOutlet) Close() error {
	return o.outlet.Close()
}

func (o *wrappedOutlet) Done() <-chan struct{} {
	return o.outlet.Done()
}

func (o *wrappedOutlet) OnEvent(event beat.Event) bool {
	// Mutate the event then pass it on.
	o.eventTransform(event)
	return o.outlet.OnEvent(event)
}

// A wrapper around a generic Outleter that produces Outleters that apply the
// given transformation to incoming events before sending them.
type wrappedConnector struct {
	connector      Connector
	eventTransform func(beat.Event)
}

func (c *wrappedConnector) Connect(conf *common.Config) (Outleter, error) {
	outleter, err := c.connector.Connect(conf)
	if err != nil {
		return outleter, err
	}
	return WrapOutlet(outleter, c.eventTransform), nil
}

func (c *wrappedConnector) ConnectWith(
	conf *common.Config, clientConf beat.ClientConfig,
) (Outleter, error) {
	outleter, err := c.connector.ConnectWith(conf, clientConf)
	if err != nil {
		return outleter, err
	}
	return WrapOutlet(outleter, c.eventTransform), nil
}

// WrapConnector takes a Connector and an event transformation function and
// returns a new Connector whose generated Outleters apply the given
// transformation to incoming events before forwarding them.
func WrapConnector(
	connector Connector, eventTransform func(beat.Event),
) Connector {
	return &wrappedConnector{connector: connector, eventTransform: eventTransform}
}
