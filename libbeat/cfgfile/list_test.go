package cfgfile

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

type runner struct {
	id      int64
	started bool
	stopped bool
}

func (r *runner) Start() { r.started = true }
func (r *runner) Stop()  { r.stopped = true }

type runnerFactory struct{ runners []*runner }

func (r *runnerFactory) Create(x beat.Pipeline, c *common.Config, meta *common.MapStrPointer) (Runner, error) {
	config := struct {
		ID int64 `config:"id"`
	}{}

	err := c.Unpack(&config)
	if err != nil {
		return nil, err
	}

	// id < 0 is an invalid config
	if config.ID < 0 {
		return nil, errors.New("Invalid config")
	}

	runner := &runner{id: config.ID}
	r.runners = append(r.runners, runner)
	return runner, err
}

func TestNewConfigs(t *testing.T) {
	factory := &runnerFactory{}
	list := NewRunnerList("", factory, nil)

	list.Reload([]*ConfigWithMeta{
		createConfig(1),
		createConfig(2),
		createConfig(3),
	})

	assert.Equal(t, len(list.copyRunnerList()), 3)
}

func TestReloadSameConfigs(t *testing.T) {
	factory := &runnerFactory{}
	list := NewRunnerList("", factory, nil)

	list.Reload([]*ConfigWithMeta{
		createConfig(1),
		createConfig(2),
		createConfig(3),
	})

	state := list.copyRunnerList()
	assert.Equal(t, len(state), 3)

	list.Reload([]*ConfigWithMeta{
		createConfig(1),
		createConfig(2),
		createConfig(3),
	})

	// nothing changed
	assert.Equal(t, state, list.copyRunnerList())
}

func TestReloadStopConfigs(t *testing.T) {
	factory := &runnerFactory{}
	list := NewRunnerList("", factory, nil)

	list.Reload([]*ConfigWithMeta{
		createConfig(1),
		createConfig(2),
		createConfig(3),
	})

	assert.Equal(t, len(list.copyRunnerList()), 3)

	list.Reload([]*ConfigWithMeta{
		createConfig(1),
		createConfig(3),
	})

	assert.Equal(t, len(list.copyRunnerList()), 2)
}

func TestReloadStartStopConfigs(t *testing.T) {
	factory := &runnerFactory{}
	list := NewRunnerList("", factory, nil)

	list.Reload([]*ConfigWithMeta{
		createConfig(1),
		createConfig(2),
		createConfig(3),
	})

	state := list.copyRunnerList()
	assert.Equal(t, len(state), 3)

	list.Reload([]*ConfigWithMeta{
		createConfig(1),
		createConfig(3),
		createConfig(4),
	})

	assert.Equal(t, len(list.copyRunnerList()), 3)
	assert.NotEqual(t, state, list.copyRunnerList())
}

func TestStopAll(t *testing.T) {
	factory := &runnerFactory{}
	list := NewRunnerList("", factory, nil)

	list.Reload([]*ConfigWithMeta{
		createConfig(1),
		createConfig(2),
		createConfig(3),
	})

	assert.Equal(t, len(list.copyRunnerList()), 3)
	list.Stop()
	assert.Equal(t, len(list.copyRunnerList()), 0)

	for _, r := range list.runners {
		assert.False(t, r.(*runner).stopped)
	}
}

func TestHas(t *testing.T) {
	factory := &runnerFactory{}
	list := NewRunnerList("", factory, nil)
	config := createConfig(1)

	hash, err := HashConfig(config.Config)
	if err != nil {
		t.Fatal(err)
	}

	list.Reload([]*ConfigWithMeta{
		config,
	})

	assert.True(t, list.Has(hash))
	assert.False(t, list.Has(0))
}

func createConfig(id int64) *ConfigWithMeta {
	c := common.NewConfig()
	c.SetInt("id", -1, id)
	return &ConfigWithMeta{
		Config: c,
	}
}
