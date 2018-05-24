package cfgfile

import (
	"github.com/mitchellh/hashstructure"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// RunnerList implements a reloadable.List of Runners
type RunnerList struct {
	runners  map[uint64]Runner
	factory  RunnerFactory
	pipeline beat.Pipeline
}

// NewRunnerList builds and returns a RunnerList
func NewRunnerList(factory RunnerFactory, pipeline beat.Pipeline) *RunnerList {
	return &RunnerList{
		runners:  map[uint64]Runner{},
		factory:  factory,
		pipeline: pipeline,
	}
}

// Reload the list of runners to match the given state
// TODO return list of errors (even we don't stop on them)
func (r *RunnerList) Reload(configs []*common.Config) error {
	startList := map[uint64]*common.Config{}
	stopList := r.copyRunnerList()

	debugf("Starting reload procedure, current runners: %d", len(stopList))

	// diff current & desired state, create action lists
	for _, config := range configs {
		hash, err := HashConfig(config)
		if err != nil {
			logp.Err("Unable to hash given config: %v", err)
			continue
		}

		if _, ok := stopList[hash]; ok {
			delete(stopList, hash)
		} else {
			startList[hash] = config
		}
	}

	// Stop removed runners
	for hash, runner := range stopList {
		debugf("Stopping runner: %s", runner)
		delete(r.runners, hash)
		go runner.Stop()
	}

	// Start new runners
	for hash, config := range startList {
		runner, err := r.factory.Create(r.pipeline, config, nil)
		if err != nil {
			debugf("Error creating runner from config: %s", err)
			continue
		}

		debugf("Starting runner: %s", runner)
		r.runners[hash] = runner
		runner.Start()
	}

	return nil
}

// Stop all runners
func (r *RunnerList) Stop() {
	for h, runner := range r.runners {
		debugf("Stopping runner: %s", runner)
		delete(r.runners, h)
		runner.Stop()
	}
}

// Has returns true if a runner with the given hash is running
func (r *RunnerList) Has(hash uint64) bool {
	_, ok := r.runners[hash]
	return ok
}

// Add the given runner to the list of running runners
func (r *RunnerList) Add(hash uint64, runner Runner) {
	r.runners[hash] = runner
}

// Get returns the runner with the given hash (nil if not found)
func (r *RunnerList) Get(hash uint64) Runner {
	return r.runners[hash]
}

// Remove the Runner with the given hash
func (r *RunnerList) Remove(hash uint64) {
	delete(r.runners, hash)
}

// HashConfig hashes a given common.Config
func HashConfig(c *common.Config) (uint64, error) {
	var config map[string]interface{}
	c.Unpack(&config)
	return hashstructure.Hash(config, nil)
}

func (r *RunnerList) copyRunnerList() map[uint64]Runner {
	list := make(map[uint64]Runner, len(r.runners))
	for k, v := range r.runners {
		list[k] = v
	}
	return list
}
