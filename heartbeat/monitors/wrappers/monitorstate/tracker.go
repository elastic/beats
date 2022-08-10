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
	mst := &MonitorStateTracker{
		states:      map[string]*MonitorState{},
		mtx:         sync.Mutex{},
		stateLoader: NilStateLoader,
	}
	if esClient != nil {
		mst.stateLoader = LoadLastESState
	}
	return mst
}

type MonitorStateTracker struct {
	states      map[string]*MonitorState
	mtx         sync.Mutex
	stateLoader StateLoader
}

// StateLoader has signature as loadLastESState, useful for test mocking, and maybe for a future impl
// other than ES if necessary
type StateLoader func(monitorId string) (*MonitorState, error)

func (mst *MonitorStateTracker) RecordStatus(monitorId string, newStatus StateStatus) (ms *MonitorState) {
	//note: the return values have no concurrency controls, they may be unsafely read unless
	//copied to the stack, copying the structs before  returning
	mst.mtx.Lock()
	defer mst.mtx.Unlock()

	state := mst.getCurrentState(monitorId)
	if state == nil {
		state = newMonitorState(monitorId, newStatus)
		mst.states[monitorId] = state
	} else {
		state.recordCheck(newStatus)
	}
	// return a copy since the state itself is a pointer that is frequently mutated
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
		loadedState, err = mst.stateLoader(monitorId)
		if err != nil {
			sleepFor := (time.Duration(i*i) * time.Second) + (time.Duration(rand.Intn(500)) * time.Millisecond)
			logp.L().Warnf("could not load last externally recorded state, will retry again in %d milliseconds: %w", sleepFor.Milliseconds(), err)
			time.Sleep(sleepFor)
		}
	}
	if err != nil {
		logp.L().Warn("could not load prior state from elasticsearch after %d attempts, will create new state for monitor %s", tries, monitorId)
	}

	// Return what we found, even if nil
	return loadedState
}

// NilStateLoader always returns nil, nil. It's the default when no ES conn is available
// or during testing
func NilStateLoader(_ string) (*MonitorState, error) {
	return nil, nil
}

// LoadLastESState attempts to find a matching prior state in Elasticsearch. If none, returns nil, nil
func LoadLastESState(monitorId string) (*MonitorState, error) {
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

	r, err := esClient.Search(func(sr *esapi.SearchRequest) {
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
