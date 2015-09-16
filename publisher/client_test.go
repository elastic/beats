package publisher

import "github.com/elastic/libbeat/common"

type nilClient struct{}

// NilClient will ignore all events being published
var NilClient Client = nilClient{}

func (c nilClient) PublishEvent(event common.MapStr, opts ...ClientOption) bool {
	return true
}

func (c nilClient) PublishEvents(events []common.MapStr, opts ...ClientOption) bool {
	return true
}
