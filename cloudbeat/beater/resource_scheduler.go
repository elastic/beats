package beater

import "github.com/elastic/beats/v7/cloudbeat/resources"

type ResourceScheduler interface {
	ScheduleResources(rmap resources.Map, resourceFunc func(interface{}))
}

type SynchronousScheduler struct {
}

func NewSynchronousScheduler() ResourceScheduler {
	return &SynchronousScheduler{}
}

func (s *SynchronousScheduler) ScheduleResources(rmap resources.Map, resourceFunc func(interface{})) {
	for _, r := range rmap {
		for _, r := range r {
			resourceFunc(r)
		}
	}
}
