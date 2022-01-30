package beater

import "github.com/elastic/beats/v7/kubebeat/resources"

type ResourceScheduler interface {
	ScheduleResources(rmap resources.Map, resourceFunc func(interface{}))
}

type SynchronousScheduler struct {
}

func NewSynchronousScheduler() ResourceScheduler {
	return &SynchronousScheduler{}
}

func (s *SynchronousScheduler) ScheduleResources(rmap resources.Map, resourceFunc func(interface{})) {
	for _, resources := range rmap {
		for _, r := range resources {
			resourceFunc(r)
		}
	}
}
