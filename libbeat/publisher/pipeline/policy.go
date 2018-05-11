package pipeline

import (
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/publisher/scheduling"
)

type wrapPolicy struct {
	beat.SchedulingPolicy
}

func wrapPolicies(ps []beat.SchedulingPolicy) []scheduling.Policy {
	policies := make([]scheduling.Policy, len(ps))
	for i, p := range ps {
		policies[i] = wrapPolicy{p}
	}
	return policies
}

func (p wrapPolicy) Connect(ctx scheduling.Context) (scheduling.Handler, error) {
	return p.SchedulingPolicy.Connect(ctx)
}
