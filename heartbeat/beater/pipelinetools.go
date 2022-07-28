package beater

import "github.com/elastic/beats/v7/libbeat/beat"

type SyncPipelineClientAdaptor struct {
	C beat.Client
}

func (s SyncPipelineClientAdaptor) Publish(event beat.Event) error {
	s.C.Publish(event)
	return nil
}

func (s SyncPipelineClientAdaptor) PublishAll(events []beat.Event) error {
	s.C.PublishAll(events)
	return nil
}

func (s SyncPipelineClientAdaptor) Close() error {
	s.C.Close()
	return nil
}

func (s SyncPipelineClientAdaptor) Wait() {
	// intentionally blank, async pipelines should be empty
}
