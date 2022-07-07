package monitorstate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/elastic/beats/v7/heartbeat/esutil"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
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

func (mst *MonitorStateTracker) Compute(monitorId string, isUp bool) (curState *MonitorState) {
	// If state is missing load it from ES
	if _, ok := mst.states[monitorId]; !ok && esClient != nil {
		loadedState, err := LoadLastState(monitorId, esClient)
		if err != nil {
			// TODO: What behavior do we really want here?
			logp.Warn("could not load last state from elasticsearch, will create new state: %w", err)
		}

		mst.states[monitorId] = loadedState
	}

	currentStatus := StatusDown
	if isUp {
		currentStatus = StatusUp
	}

	//note: the return values have no concurrency controls, they may be unsafely read unless
	//copied to the stack, copying the structs before  returning
	mst.mtx.Lock()
	defer mst.mtx.Unlock()

	if state, ok := mst.states[monitorId]; ok {
		if state.IsFlapping() {
			// Check to see if there's still an ongoing flap after recording
			// the new status
			if state.flapCompute(currentStatus) {
				state.recordCheck(isUp)
				return state
			} else {
				state.Ends = state
				newState := *NewMonitorState(monitorId, isUp)
				internalNewState := newState // Copy the struct since the returned value is read after the mutex
				mst.states[monitorId] = &internalNewState
				return &newState
			}
		} else if state.Status == currentStatus {
			// The state is stable, no changes needed
			state.recordCheck(isUp)
			return state
		} else if state.StartedAtMs > time.Now().Add(-FlappingThreshold).UnixMilli() {
			// The state changed too quickly, we're now flapping
			// TODO: is the above conditional right?
			state.flapCompute(currentStatus) // record the new state to the flap history
			state.recordCheck(isUp)
			return state
		}
	}

	// No previous state, so make a new one
	newState := *NewMonitorState(monitorId, isUp)
	internalNewState := newState
	// Use a copy of the struct so that return values can safely be used concurrently
	mst.states[monitorId] = &internalNewState
	return &newState
}
func LoadLastState(monitorId string, esc *elasticsearch.Client) (*MonitorState, error) {
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
		logp.Info("no previous state found for monitor %s, will initialize new state", monitorId)
		return NewMonitorState(monitorId, true), nil
	}

	return &sh.Hits.Hits[0].Source.State, nil
}
