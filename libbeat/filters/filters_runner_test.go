package filters

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"

	"github.com/stretchr/testify/assert"
)

// Nop filter for testing purposes
type Nop struct {
	name string
}

func (nop *Nop) New(name string, config map[string]interface{}) (FilterPlugin, error) {
	return &Nop{name: name}, nil
}

func (nop *Nop) Filter(event common.MapStr) (common.MapStr, error) {
	return event, nil
}

func (nop *Nop) String() string {
	return nop.name
}

func (nop *Nop) Type() Filter {
	return NopFilter
}

func loadPlugins() {
	Filters.Register(NopFilter, new(Nop))
}

func TestFilterRunner(t *testing.T) {
	loadPlugins()

	output := make(chan common.MapStr, 10)

	filter1, err := new(Nop).New("nop1", map[string]interface{}{})
	assert.Nil(t, err)

	filter2, err := new(Nop).New("nop2", map[string]interface{}{})
	assert.Nil(t, err)

	runner := NewFilterRunner(output, []FilterPlugin{filter1, filter2})
	assert.NotNil(t, runner)

	go runner.Run()

	runner.FiltersQueue <- common.MapStr{"hello": "world"}
	runner.FiltersQueue <- common.MapStr{"foo": "bar"}

	res := <-output
	assert.Equal(t, common.MapStr{"hello": "world"}, res)

	res = <-output
	assert.Equal(t, common.MapStr{"foo": "bar"}, res)
}

func TestLoadConfiguredFilters(t *testing.T) {
	loadPlugins()

	type o struct {
		Name string
		Type Filter
	}

	type io struct {
		Input  map[string]interface{}
		Output []o
	}

	tests := []io{
		// should find configuration by types
		{
			Input: map[string]interface{}{
				"filters": []interface{}{"nop1", "nop2"},
				"nop1": map[interface{}]interface{}{
					"type": "nop",
				},
				"nop2": map[interface{}]interface{}{
					"type": "nop",
				},
			},
			Output: []o{
				{
					Name: "nop1",
					Type: NopFilter,
				},
				{
					Name: "nop2",
					Type: NopFilter,
				},
			},
		},
		// should work with implicit configuration by name
		{
			Input: map[string]interface{}{
				"filters": []interface{}{"nop", "sample1"},
				"sample1": map[interface{}]interface{}{
					"type": "nop",
				},
			},
			Output: []o{
				{
					Name: "nop",
					Type: NopFilter,
				},
				{
					Name: "sample1",
					Type: NopFilter,
				},
			},
		},
	}

	for _, test := range tests {
		res, err := LoadConfiguredFilters(test.Input)
		assert.Nil(t, err)

		res_o := []o{}
		for _, r := range res {
			res_o = append(res_o, o{Name: r.String(), Type: r.Type()})
		}

		assert.Equal(t, test.Output, res_o)
	}
}

func TestLoadConfiguredFiltersNegative(t *testing.T) {
	loadPlugins()

	type io struct {
		Input map[string]interface{}
		Err   string
	}

	tests := []io{
		{
			Input: map[string]interface{}{
				"filters": []interface{}{"nop1", "nop2"},
				"nop1": map[interface{}]interface{}{
					"type": "nop",
				},
			},
			Err: "No such filter type and no corresponding configuration: nop2",
		},
		{
			Input: map[string]interface{}{
				"filters": []interface{}{"nop1", "nop"},
				"nop1": map[interface{}]interface{}{
					"hype": "nop",
				},
			},
			Err: "Couldn't get type for filter: nop1",
		},
		{
			Input: map[string]interface{}{
				"filters": []interface{}{"nop1", "nop"},
				"nop1": map[interface{}]interface{}{
					"type": 1,
				},
			},
			Err: "Couldn't get type for filter: nop1",
		},
	}

	for _, test := range tests {
		_, err := LoadConfiguredFilters(test.Input)
		assert.NotNil(t, err)
		assert.Equal(t, test.Err, err.Error())
	}
}
