package scheduling

import "github.com/elastic/beats/libbeat/beat"

type Client struct {
	ctx      *context
	handlers []Handler
}

func newClient(ctx *context, handler []Handler) *Client {
	return &Client{
		ctx:      ctx,
		handlers: handler,
	}
}

func (c *Client) Close() {
	type closingHandler interface {
		Close()
	}

	c.ctx.Close()

	for _, h := range c.handlers {
		if c, ok := h.(closingHandler); ok {
			c.Close()
		}
	}
}

func (c *Client) OnEvent(evt beat.Event) (beat.Event, error) {
	if !c.ctx.Active() {
		return evt, SigClose
	}

	for _, h := range c.handlers {
		var err error

		evt, err = h.OnEvent(evt)
		if err != nil {
			return evt, err
		}
	}

	return evt, nil
}
