package cfgfile

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
)

type multiplexedFactory []FactoryMatcher

type FactoryMatcher func(cfg *common.Config) RunnerFactory

var errConfigDoesNotMatch = errors.New("config does not match accepted configurations")

func MultiplexedRunnerFactory(matchers ...FactoryMatcher) RunnerFactory {
	return multiplexedFactory(matchers)
}

func MatchHasField(field string, factory RunnerFactory) FactoryMatcher {
	return func(cfg *common.Config) RunnerFactory {
		if cfg.HasField(field) {
			return factory
		}
		return nil
	}
}

func MatchDefault(factory RunnerFactory) FactoryMatcher {
	return func(cfg *common.Config) RunnerFactory {
		return factory
	}
}

func (f multiplexedFactory) Create(
	p beat.Pipeline,
	config *common.Config,
	meta *common.MapStrPointer,
) (Runner, error) {
	factory, err := f.findFactory(config)
	if err != nil {
		return nil, err
	}
	return factory.Create(p, config, meta)
}

func (f multiplexedFactory) CheckConfig(c *common.Config) error {
	factory, err := f.findFactory(c)
	if err == nil {
		err = factory.CheckConfig(c)
	}
	return err
}

func (f multiplexedFactory) findFactory(c *common.Config) (RunnerFactory, error) {
	for _, matcher := range f {
		if factory := matcher(c); factory != nil {
			return factory, nil
		}
	}

	return nil, errConfigDoesNotMatch
}
