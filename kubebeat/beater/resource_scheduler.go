package beater

import "github.com/elastic/beats/v7/libbeat/logp"

type ResourceScheduler interface {
	RunResource(resourcesMap map[string][]interface{}, resourceFunc func(interface{}))
}

type SynchronousScheduler struct {
}

func NewSynchronousScheduler() ResourceScheduler {
	return &SynchronousScheduler{}
}

func (s *SynchronousScheduler) RunResource(resourcesMap map[string][]interface{}, resourceFunc func(interface{})) {
	for _, resources := range resourcesMap {
		for _, r := range resources {
			logp.Info("amiramir single resource %+v", r)
			resourceFunc(r)
		}
	}
}
