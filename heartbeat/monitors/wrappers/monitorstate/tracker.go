package monitorstate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"

	"github.com/elastic/beats/v7/heartbeat/esutil"
)

var esClient *elasticsearch.Client

func SetEsClient(c *elasticsearch.Client) {
	esClient = c
}

func NewMonitorStateTracker() *MonitorStateTracker {
	return &MonitorStateTracker{
		states: map[string]*MonitorState{},
		mtx:    sync.Mutex{},
	}
}

type MonitorStateTracker struct {
	states map[string]*MonitorState
	mtx    sync.Mutex
}

func (mst *MonitorStateTracker) RecordStatus(monitorId string, newStatus MonitorStatus) (urState *MonitorState) {
	//note: the return values have no concurrency controls, they may be unsafely read unless
	//copied to the stack, copying the structs before  returning
	mst.mtx.Lock()
	defer mst.mtx.Unlock()

	state := mst.getCurrentState(monitorId)
	state.recordCheck(newStatus)
	return state.copy()
}

func (mst *MonitorStateTracker) getCurrentState(monitorId string) (state *MonitorState) {
	if state, ok := mst.states[monitorId]; ok {
		return state
	}

	// If there's no ES client then we just work off memory
	if esClient == nil {
		return nil
	}

	tries := 3
	var loadedState *MonitorState
	var err error
	for i := 0; i < tries; i++ {
		loadedState, err = loadLastESState(monitorId, esClient)
		if err != nil {
			sleepFor := (time.Duration(i*i) * time.Second) + (time.Duration(rand.Intn(500)) * time.Millisecond)
			logp.L().Warnf("could not load last state from elasticsearch, will retry again in %d milliseconds: %w", sleepFor.Milliseconds(), err)
			time.Sleep(sleepFor)
			return nil
		}
	}
	if err != nil {
		logp.Warn("could not load prior state from elasticsearch after %d attempts, will create new state for monitor %s", tries, monitorId)
		return nil
	}

	// loadedState could be nil if we have no previous state history
	if loadedState != nil {
		mst.states[monitorId] = loadedState
	}

	// Return what we found, even if nil
	return loadedState
}

func loadLastESState(monitorId string, esc *elasticsearch.Client) (*MonitorState, error) {
	reqBody, err := json.Marshal(mapstr.M{
		"sort": mapstr.M{"@timestamp": "desc"},
		"query": mapstr.M{
			"bool": mapstr.M{
				"must": []mapstr.M{
					{
						"match": mapstr.M{"monitor.id": monitorId},
					},
					{
						"exists": mapstr.M{"field": "summary"},
					},
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("could not serialize query for state save: %w", err)
	}

	r, err := esc.Search(func(sr *esapi.SearchRequest) {
		sr.Index = []string{"synthetics-*"}
		size := 1
		sr.Size = &size
		sr.Body = bytes.NewReader(reqBody)
	})

	type stateHits struct {
		Hits struct {
			Hits []struct {
				DocId  string `json:"string"`
				Source struct {
					State MonitorState `json:"state"`
				} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	respBody, err := esutil.CheckRetResp(r, err)
	if err != nil {
		return nil, fmt.Errorf("error executing state search for %s: %w", monitorId, err)
	}

	sh := stateHits{}
	err = json.Unmarshal(respBody, &sh)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal state hits for %s: %w", monitorId, err)
	}

	if len(sh.Hits.Hits) == 0 {
		logp.L().Infof("no previous state found for monitor %s", monitorId)
		return nil, nil
	}

	return &sh.Hits.Hits[0].Source.State, nil
}
