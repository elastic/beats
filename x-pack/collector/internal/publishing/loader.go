package publishing

import (
	"fmt"
	"strings"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

type Loader struct {
	plugins map[string]Plugin
}

// SetupError indicates that the loader initialization has detected
// errors in individual plugin configurations or duplicates.
type SetupError struct {
	Fails []error
}

func NewLoader(plugins []Plugin) (*Loader, error) {
	if errs := validatePlugins(plugins); len(errs) > 0 {
		return nil, &SetupError{errs}
	}

	tbl := make(map[string]Plugin, len(plugins))
	for _, p := range plugins {
		tbl[p.Name] = p
	}

	return &Loader{plugins: tbl}, nil
}

func (l *Loader) ConfigureOutput(log *logp.Logger, cfg *common.Config) (Output, error) {
	typeInfo := struct{ Type string }{}
	if err := cfg.Unpack(&typeInfo); err != nil {
		return nil, err
	}

	log.Debugf("Looking up '%v' output", typeInfo.Type)

	p, exists := l.plugins[typeInfo.Type]
	if !exists {
		return nil, fmt.Errorf("unknown output type %v", typeInfo.Type)
	}

	return p.Configure(log, cfg)
}

// validatePlugins checks if there are multiple plugins with the same name in
// the registry.
func validatePlugins(plugins []Plugin) []error {
	var errs []error

	counts := map[string]int{}
	for _, p := range plugins {
		counts[p.Name]++
	}

	for name, count := range counts {
		if count > 1 {
			errs = append(errs, fmt.Errorf("plugin '%v' found %v times", name, count))
		}
	}
	return errs
}

// Error returns the errors string repesentation
func (e *SetupError) Error() string {
	var buf strings.Builder
	buf.WriteString("invalid plugin setup found:")
	for _, err := range e.Fails {
		fmt.Fprintf(&buf, "\n\t%v", err)
	}
	return buf.String()
}
