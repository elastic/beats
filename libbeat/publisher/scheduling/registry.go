package scheduling

import (
	"fmt"
	"sync"
)

type policyRegistry struct {
	reg map[string]PolicyFactory
	mu  sync.Mutex
}

var PolicyRegistry = &policyRegistry{}

func (p *policyRegistry) Register(name string, factory PolicyFactory) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exist := p.reg[name]; exist {
		panic(fmt.Sprintf("policy '%v' already registered", name))
	}

	p.reg[name] = factory
}

func (p *policyRegistry) Find(name string) PolicyFactory {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.reg[name]
}
