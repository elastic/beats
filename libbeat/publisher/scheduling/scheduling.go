package scheduling

import (
	"fmt"
)

type Scheduling struct {
	groups map[string]Group
	global []Policy
}

type Group struct {
	parent   string
	policies []Policy
}

func (s *Scheduling) Connect(group string, local []Policy) (*Client, error) {
	var groupPolicies []Policy
	var global []Policy

	if group != "" {
		gp, err := s.collectGroupPolicies(group)
		if err != nil {
			return nil, err
		}

		groupPolicies = gp
	}

	if s != nil {
		global = s.global
	}

	total := len(local) + len(groupPolicies) + len(global)
	if total == 0 {
		return nil, nil
	}

	ctx := newContext()
	handlers := make([]Handler, 0, total)
	createHandlers := func(policies []Policy) error {
		for _, policy := range policies {
			handler, err := policy.Connect(ctx)
			if err != nil {
				return fmt.Errorf("policy initialization failed: %v", err)
			}

			handlers = append(handlers, handler)
		}

		return nil
	}

	err := createHandlers(local)
	if err != nil {
		return nil, fmt.Errorf("local %v", err)
	}

	err = createHandlers(groupPolicies)
	if err != nil {
		return nil, fmt.Errorf("group %v", err)
	}

	err = createHandlers(global)
	if err != nil {
		return nil, fmt.Errorf("global %v", err)
	}

	return newClient(ctx, handlers), nil
}

func (s *Scheduling) collectGroupPolicies(name string) ([]Policy, error) {
	if s == nil {
		if name != "" {
			return nil, fmt.Errorf("unknown policy group '%v'", name)
		}
		return nil, nil
	}

	if name == "" {
		return nil, nil
	}

	var policies []Policy
	visited := map[string]bool{}
	for name != "" {
		if visited[name] {
			return nil, fmt.Errorf("group definition cycle detected when accessing: %v", name)
		}

		grp, exists := s.groups[name]
		if !exists {
			return nil, fmt.Errorf("unknown policy group '%v'", name)
		}

		policies = append(policies, grp.policies...)
		name = grp.parent
	}

	return policies, nil
}
