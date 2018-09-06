package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/feature"
	"github.com/elastic/beats/libbeat/logp"
)

func TestBuilder(t *testing.T) {
	provider := "myprovider"
	providerFactory := func(_ *logp.Logger, _ *Registry, _ *common.Config) (Provider, error) {
		return nil, nil
	}

	fnFactory1 := func(_ Provider, _ *common.Config) (Function, error) { return nil, nil }
	fnFactory2 := func(_ Provider, _ *common.Config) (Function, error) { return nil, nil }

	b := MustCreate(
		provider,
		providerFactory,
		feature.NewDetails("myprovider", "myprovider", feature.Experimental),
	).MustAddFunction(
		"f1",
		fnFactory1,
		feature.NewDetails("fn1 description", "fn1", feature.Experimental),
	).MustAddFunction("f2", fnFactory2, feature.NewDetails(
		"fn1 description",
		"fn1",
		feature.Experimental,
	)).Bundle()

	assert.Equal(t, 3, len(b.Features()))
	features := b.Features()

	assert.Equal(t, "myprovider", features[0].Name())
	assert.Equal(t, "beatless.provider", features[0].Namespace())

	assert.Equal(t, "f1", features[1].Name())
	assert.Equal(t, "beatless.provider.myprovider.functions", features[1].Namespace())

	assert.Equal(t, "f2", features[2].Name())
	assert.Equal(t, "beatless.provider.myprovider.functions", features[2].Namespace())
}
