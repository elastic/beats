package scheduling

import (
	"fmt"
)

type Scheduling struct {
	groups map[string][]Policy
	global []Policy
}

func (s *Scheduling) Connect(group string, local []Policy) (*Client, error) {

	var groupPolicies []Policy
	var global []Policy

	if group != "" {
		var exist bool

		var groups map[string][]Policy
		if s != nil {
			groups = s.groups
		}

		groupPolicies, exist = groups[group]
		if !exist {
			return nil, fmt.Errorf("scheduling group '%v' does not exist", group)
		}
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
