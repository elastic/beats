package actions

import (
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors"
)

func configChecked(
	constr processors.Constructor,
	checks ...func(*common.Config) error,
) processors.Constructor {
	validator := checkAll(checks...)
	return func(cfg *common.Config) (processors.Processor, error) {
		err := validator(cfg)
		if err != nil {
			return nil, fmt.Errorf("%v in %v", err.Error(), cfg.Path())
		}

		return constr(cfg)
	}
}

func checkAll(checks ...func(*common.Config) error) func(*common.Config) error {
	return func(c *common.Config) error {
		for _, check := range checks {
			if err := check(c); err != nil {
				return err
			}
		}
		return nil
	}
}

func requireFields(fields ...string) func(*common.Config) error {
	return func(cfg *common.Config) error {
		for _, field := range fields {
			if !cfg.HasField(field) {
				return fmt.Errorf("missing %v option", field)
			}
		}
		return nil
	}
}

func allowedFields(fields ...string) func(*common.Config) error {
	return func(cfg *common.Config) error {
		for _, field := range cfg.GetFields() {
			found := false
			for _, allowed := range fields {
				if field == allowed {
					found = true
					break
				}
			}

			if !found {
				return fmt.Errorf("unexpected %v option", field)
			}
		}
		return nil
	}
}
