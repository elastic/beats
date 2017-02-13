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
}

func newHarvesterRegistry() *harvesterRegistry {
	return &harvesterRegistry{
		harvesters: map[uuid.UUID]*harvester.Harvester{},
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
	for _, hv := range hr.harvesters {
		hr.wg.Add(1)
		go func(h *harvester.Harvester) {
			hr.wg.Done()
			h.Stop()
		}(hv)
	}
	hr.Unlock()
	hr.waitForCompletion()
}

func (hr *harvesterRegistry) waitForCompletion() {
	hr.wg.Wait()
}

func (hr *harvesterRegistry) start(h *harvester.Harvester, r reader.Reader) {

	hr.wg.Add(1)
	hr.add(h)
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
