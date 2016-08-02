package lutool

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type runner struct {
	backoff BackoffConfig
	module  Runner
}

type BackoffConfig struct {
	Duration time.Duration `config:"duration" validate:"min=0"`
	Factor   float64       `config:"factor"   validate:"min=1.0"`
	Max      time.Duration `config:"max"      validate:"min=0"`
}

func newRunner(backoff BackoffConfig, backend Runner) (*runner, error) {
	return &runner{backoff, backend}, nil
}

func (r *runner) do(e *binEntry, event common.MapStr) error {
	debugf("Execute lookup for key: ", e.key)

	backoff, err := r.checkBackoff(e)
	if err != nil {
		return err
	}

	value, err := r.module.Exec(event)
	if err != nil {
		logp.Err("Lookup runner returned error: ", err)
		e.value = binError{time.Now(), backoff, err}
		return err
	}

	debugf("Lookup returned fields: ", value)
	e.value = binLookupValue(value)
	return nil
}

// backoff checks if entry is to be run or last error must be returned.
// Returns the last used backoff duration for storing with failed event and
// last known error if command shall not be run yet. Error being nil is returned
// if command shall be executed.
func (r *runner) checkBackoff(e *binEntry) (time.Duration, error) {
	backoff := r.backoff.Duration
	if backoff == 0 { // check backoff being enabled
		return 0, nil
	}

	// do not backoff, if entry is new
	v := e.value
	if v == nil {
		return 0, nil
	}
	err := v.error()
	if err == nil {
		return 0, nil
	}

	// increment backoff up to max
	f := r.backoff.Factor
	backoff = v.backoff() + time.Duration(f*float64(backoff))
	if backoff > r.backoff.Max {
		backoff = r.backoff.Max
	}

	dur := time.Now().Sub(v.lastRun())
	if dur < backoff {
		return v.backoff(), err
	}

	// run command with updated backoff on next fail
	return backoff, nil
}
