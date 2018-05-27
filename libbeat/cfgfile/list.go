package cfgfile

import (
	"sync"

	"github.com/joeshaw/multierror"
	"github.com/mitchellh/hashstructure"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// RunnerList implements a reloadable.List of Runners
type RunnerList struct {
	runners  map[uint64]Runner
	mutex    sync.RWMutex
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
func (r *RunnerList) Reload(configs []*common.Config) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var errs multierror.Errors

	startList := map[uint64]*common.Config{}
	stopList := r.copyRunnerList()

	debugf("Starting reload procedure, current runners: %d", len(stopList))

	// diff current & desired state, create action lists
	for _, config := range configs {
		hash, err := HashConfig(config)
		if err != nil {
			err = errors.Wrap(err, "Unable to hash given config")
			errs = append(errs)
			logp.Error(err)
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
			err = errors.Wrap(err, "Error creating runner from config")
			errs = append(errs)
			logp.Error(err)
			continue
		}

		debugf("Starting runner: %s", runner)
		r.runners[hash] = runner
		runner.Start()
	}

	return errs.Err()
}

// Stop all runners
func (r *RunnerList) Stop() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if len(r.runners) == 0 {
		return
	}

	logp.Info("Stopping %v runners ...", len(r.runners))

	wg := sync.WaitGroup{}
	for hash, runner := range r.runners {
		wg.Add(1)

		// Stop modules in parallel
		go func(h uint64, run Runner) {
			defer wg.Done()
			debugf("Stopping runner: %s", run)
			delete(r.runners, h)
			run.Stop()
			debugf("Stopped runner: %s", run)
		}(hash, runner)
	}

	wg.Wait()
}

// Has returns true if a runner with the given hash is running
func (r *RunnerList) Has(hash uint64) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	_, ok := r.runners[hash]
	return ok
}

// Add the given runner to the list of running runners
func (r *RunnerList) Add(hash uint64, runner Runner) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.runners[hash] = runner
}

// Get returns the runner with the given hash (nil if not found)
func (r *RunnerList) Get(hash uint64) Runner {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.runners[hash]
}

// Remove the Runner with the given hash
func (r *RunnerList) Remove(hash uint64) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
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
