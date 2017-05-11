package prospector

import (
	"sync"

	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/harvester/reader"
	uuid "github.com/satori/go.uuid"
)

type harvesterRegistry struct {
	sync.Mutex
	harvesters map[uuid.UUID]*harvester.Harvester
	wg         sync.WaitGroup
	done       chan struct{}
}

func newHarvesterRegistry() *harvesterRegistry {
	return &harvesterRegistry{
		harvesters: map[uuid.UUID]*harvester.Harvester{},
		done:       make(chan struct{}),
	}
}

func (hr *harvesterRegistry) add(h *harvester.Harvester) {
	hr.Lock()
	defer hr.Unlock()
	hr.harvesters[h.ID] = h
}

func (hr *harvesterRegistry) remove(h *harvester.Harvester) {
	hr.Lock()
	defer hr.Unlock()
	delete(hr.harvesters, h.ID)
}

func (hr *harvesterRegistry) Stop() {

	hr.Lock()
	defer func() {
		hr.Unlock()
		hr.waitForCompletion()
	}()
	close(hr.done)
	for _, hv := range hr.harvesters {
		hr.wg.Add(1)
		go func(h *harvester.Harvester) {
			hr.wg.Done()
			h.Stop()
		}(hv)
	}

}

func (hr *harvesterRegistry) waitForCompletion() {
	hr.wg.Wait()
}

func (hr *harvesterRegistry) start(h *harvester.Harvester, r reader.Reader) {

	// Make sure stop is not called during starting a harvester
	hr.Lock()

	// Make sure no new harvesters are started after stop was called
	select {
	case <-hr.done:
		return
	default:
	}

	hr.wg.Add(1)
	h.StopWg.Add(1)

	hr.Unlock()

	// TODO: It could happen that stop is called here

	hr.add(h)

	// Update state before staring harvester
	// This makes sure the states is set to Finished: false
	// This is synchronous state update as part of the scan
	h.SendStateUpdate()

	go func() {
		defer func() {
			hr.remove(h)
			hr.wg.Done()
		}()
		// Starts harvester and picks the right type. In case type is not set, set it to default (log)
		h.Harvest(r)
	}()
}

func (hr *harvesterRegistry) len() uint64 {
	hr.Lock()
	defer hr.Unlock()
	return uint64(len(hr.harvesters))
}
