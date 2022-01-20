package beater

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/kubernetes"
	"github.com/elastic/beats/v7/libbeat/logp"
)

// Data maintains a cache that is updated by Fetcher implementations registered
// against it. It sends the cache to an output channel at the defined interval.
type Data struct {
	interval time.Duration
	output   chan map[string][]interface{}

	ctx             context.Context
	cancel          context.CancelFunc
	state           map[string][]interface{}
	fetcherRegistry map[string]registeredFetcher
	leaseInfo       *LeaseInfo
	wg              *sync.WaitGroup
}

type registeredFetcher struct {
	f          Fetcher
	leaderOnly bool
}

// NewData returns a new Data instance with the given interval.
func NewData(ctx context.Context, interval time.Duration) (*Data, error) {
	ctx, cancel := context.WithCancel(ctx)

	li, err := NewLeaseInfo(ctx)
	if err != nil {
		return nil, err
	}

	return &Data{
		interval:        interval,
		output:          make(chan map[string][]interface{}),
		ctx:             ctx,
		cancel:          cancel,
		state:           make(map[string][]interface{}),
		fetcherRegistry: make(map[string]registeredFetcher),
		leaseInfo:       li,
	}, nil
}

// Output returns the output channel.
func (d *Data) Output() <-chan map[string][]interface{} {
	return d.output
}

// RegisterFetcher registers a Fetcher implementation.
// If leaderOnly is true, then this given Fetcher will only be invoked when
// the current instance is an elected leader.
func (d *Data) RegisterFetcher(key string, f Fetcher, leaderOnly bool) error {
	if _, ok := d.fetcherRegistry[key]; ok {
		return fmt.Errorf("fetcher key collision: %q is already registered", key)
	}

	d.fetcherRegistry[key] = registeredFetcher{
		f:          f,
		leaderOnly: leaderOnly,
	}

	return nil
}

func (d *Data) IsLeader() bool {
	l, err := d.leaseInfo.IsLeader()
	if err != nil {
		logp.L().Errorf("could not read leader value, using default value %v: %v", l, err)
	}

	return l
}

// Run updates the cache using Fetcher implementations.
func (d *Data) Run() error {
	updates := make(chan update)

	var wg sync.WaitGroup
	d.wg = &wg

	for key, rfetcher := range d.fetcherRegistry {
		wg.Add(1)
		go func(k string, rf registeredFetcher) {
			defer wg.Done()
			d.fetchWorker(updates, k, rf)
		}(key, rfetcher)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		d.fetchManager(updates)
	}()

	return nil
}

// update is a single update sent from a worker to a manager.
type update struct {
	key string
	val []interface{}
}

func (d *Data) fetchWorker(updates chan update, k string, rf registeredFetcher) {
	for {
		// Go to sleep in each iteration.
		time.Sleep(d.interval)

		select {
		case <-d.ctx.Done():
			return
		default:
			if rf.leaderOnly {
				if !d.IsLeader() {
					continue
				}
			}

			val, err := rf.f.Fetch()
			if err != nil {
				logp.L().Errorf("error running fetcher for key %q: %v", k, err)
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
				logp.L().Errorf("could not copy data state: %v", err)
				continue
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

	for key, rfetcher := range d.fetcherRegistry {
		rfetcher.f.Stop()
		logp.L().Infof("Fetcher for key %q stopped", key)
	}

	d.wg.Wait()
}

// copy makes a copy of the given map.
func copy(m map[string][]interface{}) (map[string][]interface{}, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	dec := gob.NewDecoder(&buf)
	err := enc.Encode(m)
	if err != nil {
		return nil, err
	}
	var copy map[string][]interface{}
	err = dec.Decode(&copy)
	if err != nil {
		return nil, err
	}
	return copy, nil
}

func init() {
	gob.Register([]interface{}{})
	gob.Register(Process{})
	gob.Register(FileSystemResourceData{})

	gob.Register(KubeAPIResource{})
	gob.Register(kubernetes.Pod{})
	gob.Register(kubernetes.Secret{})
	gob.Register(kubernetes.Role{})
	gob.Register(kubernetes.RoleBinding{})
	gob.Register(kubernetes.ClusterRole{})
	gob.Register(kubernetes.ClusterRoleBinding{})
	gob.Register(kubernetes.NetworkPolicy{})
}
