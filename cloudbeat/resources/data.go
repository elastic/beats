package resources

import (
	"bytes"
	"context"
	"encoding/gob"
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/kubernetes"
	"github.com/elastic/beats/v7/libbeat/logp"
)

// Data maintains a cache that is updated by Fetcher implementations registered
// against it. It sends the cache to an output channel at the defined interval.
type Data struct {
	interval time.Duration
	output   chan Map

	state    Map
	fetchers FetchersRegistry
	wg       *sync.WaitGroup
}

type Map map[string][]FetcherResult

// NewData returns a new Data instance with the given interval.
func NewData(interval time.Duration, fetchers FetchersRegistry) (*Data, error) {

	return &Data{
		interval: interval,
		output:   make(chan Map),

		state:    make(Map),
		fetchers: fetchers,
	}, nil
}

// Output returns the output channel.
func (d *Data) Output() <-chan Map {
	return d.output
}

// Run updates the cache using Fetcher implementations.
func (d *Data) Run(ctx context.Context) error {
	updates := make(chan update)

	var wg sync.WaitGroup
	d.wg = &wg

	for _, key := range d.fetchers.Keys() {
		wg.Add(1)
		go func(k string) {
			defer wg.Done()
			d.fetchWorker(ctx, updates, k)
		}(key)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		d.fetchManager(ctx, updates)
	}()

	return nil
}

// update is a single update sent from a worker to a manager.
type update struct {
	key string
	val []FetcherResult
}

func (d *Data) fetchWorker(ctx context.Context, updates chan update, k string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if !d.fetchers.ShouldRun(k) {
				break
			}

			val, err := d.fetchers.Run(ctx, k)
			if err != nil {
				logp.L().Errorf("error running fetcher for key %q: %v", k, err)
			}

			updates <- update{k, val}
		}
		// Go to sleep in each iteration.
		time.Sleep(d.interval)
	}
}

func (d *Data) fetchManager(ctx context.Context, updates chan update) {
	ticker := time.NewTicker(d.interval)

	for {
		select {
		case <-ticker.C:
			// Generate input ID?

			c, err := copyState(d.state)
			if err != nil {
				logp.L().Errorf("could not copyState data state: %v", err)
				continue
			}

			d.output <- c

		case u := <-updates:
			d.state[u.key] = u.val

		case <-ctx.Done():
			return
		}
	}
}

// Stop cleans up Data resources gracefully.
func (d *Data) Stop(ctx context.Context, cancel context.CancelFunc) {
	cancel()

	d.fetchers.Stop(ctx)
	d.wg.Wait()

	close(d.output)
}

// copyState makes a copyState of the given map.
func copyState(m Map) (Map, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	dec := gob.NewDecoder(&buf)
	err := enc.Encode(m)
	if err != nil {
		return nil, err
	}
	var newState Map
	err = dec.Decode(&newState)
	if err != nil {
		return nil, err
	}
	return newState, nil
}

func init() {
	gob.Register([]interface{}{})
	gob.Register(FetcherResult{})
	gob.Register(ProcessResource{})
	gob.Register(FileSystemResource{})

	gob.Register(kubernetes.Pod{})
	gob.Register(kubernetes.Secret{})
	gob.Register(kubernetes.Role{})
	gob.Register(kubernetes.RoleBinding{})
	gob.Register(kubernetes.ClusterRole{})
	gob.Register(kubernetes.ClusterRoleBinding{})
	gob.Register(kubernetes.NetworkPolicy{})
}
