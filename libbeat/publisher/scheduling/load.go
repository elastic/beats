package scheduling

import (
	"errors"
	"fmt"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

type beatPolicy struct {
	Policy
}

func Load(c Config) (*Scheduling, error) {
	global, err := LoadPolicies(c.Policies)
	if err != nil {
		return nil, err
	}

	groups := make(map[string]Group, len(c.Groups))
	for name, grp := range c.Groups {
		policies, err := LoadPolicies(grp.Policies)
		if err != nil {
			return nil, fmt.Errorf("error configuring scheduling group '%v': %v", name, err)
		}

		groups[name] = Group{
			parent:   grp.Parent,
			policies: policies,
		}
	}

	return &Scheduling{
		groups: groups,
		global: global,
	}, nil
}

func LoadPolicies(cfgs []common.ConfigNamespace) ([]Policy, error) {
	policies := make([]Policy, len(cfgs))
	for i, cfg := range cfgs {
		p, err := LoadPolicy(cfg)
		if err != nil {
			return nil, err
		}

		policies[i] = p
	}

	return policies, nil
}

func LoadLocal(cfg LocalConfig) (string, []beat.SchedulingPolicy, error) {
	policies, err := LoadLocalPolicies(cfg.Policies)
	return cfg.Group, policies, err
}

func LoadLocalPolicies(cfgs []common.ConfigNamespace) ([]beat.SchedulingPolicy, error) {
	policies := make([]beat.SchedulingPolicy, len(cfgs))
	for i, cfg := range cfgs {
		p, err := LoadPolicy(cfg)
		if err != nil {
			return nil, err
		}

		policies[i] = beatPolicy{p}
	}

	return policies, nil
}

func LoadPolicy(cfg common.ConfigNamespace) (Policy, error) {
	if !cfg.IsSet() {
		return nil, errors.New("policy is not set")
	}

	name := cfg.Name()
	factory := PolicyRegistry.Find(name)
	if factory == nil {
		return nil, fmt.Errorf("unknown policy '%v'", name)
	}

	settings := cfg.Config()
	settings.PrintDebugf("policy '%v' config =>", name)
	return factory(settings)
}

func (p beatPolicy) Connect(h beat.Context) (beat.SchedulingHandler, error) {
	return p.Policy.Connect(h)
}
