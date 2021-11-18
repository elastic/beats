package beater

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/elastic/beats/v7/libbeat/logp"
)

// Data maintains a cache that is updated by Fetcher implementations registered
// against it. It sends the cache to an output channel at the defined interval.
type Data struct {
	interval time.Duration
	output   chan interface{}

	ctx      context.Context
	cancel   context.CancelFunc
	state    map[string]interface{}
	fetchers map[string]Fetcher
}

// NewData returns a new Data instance with the given interval.
func NewData(ctx context.Context, interval time.Duration) *Data {
	ctx, cancel := context.WithCancel(ctx)

	return &Data{
		interval: interval,
		output:   make(chan interface{}),
		ctx:      ctx,
		cancel:   cancel,
		state:    make(map[string]interface{}),
		fetchers: make(map[string]Fetcher),
	}
}

// Output returns the output channel.
func (d *Data) Output() <-chan interface{} {
	return d.output
}

// RegisterFetcher registers a Fetcher implementation.
func (d *Data) RegisterFetcher(key string, f Fetcher) error {
	if _, ok := d.fetchers[key]; ok {
		return fmt.Errorf("fetcher key collision: %q is already registered", key)
	}

	d.fetchers[key] = f
	return nil
}

// Run updates the cache using Fetcher implementations.
func (d *Data) Run() error {
	updates := make(chan update)

	for key, fetcher := range d.fetchers {
		go d.fetchWorker(updates, key, fetcher)
	}

	go d.fetchManager(updates)

	return nil
}

// update is a sigle update sent from a worker to a manager.
type update struct {
	key string
	val interface{}
}

func (d *Data) fetchWorker(updates chan update, k string, f Fetcher) {
	for {
		select {
		case <-d.ctx.Done():
			return
		default:
			val, err := f.Fetch()
			if err != nil {
				logp.L().Errorf("error running fetcher for key %q: %w", k, err)
			}

			updates <- update{k, val}
		}
	}
}

func (d *Data) fetchManager(updates chan update) {
	ticker := time.NewTicker(d.interval)

	for {
		select {
		case <-ticker.C:
			// Generate input ID?

			c, err := copy(d.state)
			if err != nil {
				logp.L().Errorf("could not copy data state: %w", err)
				return
			}

			d.output <- c

		case u := <-updates:
			d.state[u.key] = u.val

		case <-d.ctx.Done():
			return
		}
	}
}

// Stop cleans up Data resources gracefully.
func (d *Data) Stop() {
	d.cancel()

	for key, fetcher := range d.fetchers {
		fetcher.Stop()
		logp.L().Infof("Fetcher for key %q stopped", key)
	}
}

// copy makes a copy of the given map.
func copy(m map[string]interface{}) (map[string]interface{}, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	dec := gob.NewDecoder(&buf)
	err := enc.Encode(m)
	if err != nil {
		return nil, err
	}
	var copy map[string]interface{}
	err = dec.Decode(&copy)
	if err != nil {
		return nil, err
	}
	return copy, nil
}

func init() {
	gob.Register([]interface{}{})
	gob.Register(Process{})
	gob.Register([]FileSystemResourceData{})
}
