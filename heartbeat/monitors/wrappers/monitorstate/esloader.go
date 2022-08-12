package monitorstate

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"

	"github.com/elastic/beats/v7/heartbeat/esutil"
)

func MakeESLoader(esc *elasticsearch.Client, indexPattern string) StateLoader {
	if indexPattern == "" {
		// Should never happen, but if we ever make a coding error...
		logp.L().Warn("ES state loader initialized with no index pattern, will not load states from ES")
		return NilStateLoader
	}
	return func(monitorId string) (*State, error) {
		reqBody, err := json.Marshal(mapstr.M{
			"sort": mapstr.M{"@timestamp": "desc"},
			"query": mapstr.M{
				"bool": mapstr.M{
					"must": []mapstr.M{
						{
							"match": mapstr.M{"monitor.id": monitorId},
						},
						{
							"exists": mapstr.M{"field": "state"},
						},
						{
							// Only search the past 6h of data for perf, otherwise we reset the state
							// Monitors should run more frequently than that.
							"range": mapstr.M{"@timestamp": mapstr.M{"gt": "now-6h"}},
						},
					},
				},
			},
		})
		if err != nil {
			return nil, fmt.Errorf("could not serialize query for state save: %w", err)
		}

		r, err := esc.Search(func(sr *esapi.SearchRequest) {
			sr.Index = []string{indexPattern}
			size := 1
			sr.Size = &size
			sr.Body = bytes.NewReader(reqBody)
		})

		type stateHits struct {
			Hits struct {
				Hits []struct {
					DocId  string `json:"string"`
					Source struct {
						State State `json:"state"`
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

		state := &sh.Hits.Hits[0].Source.State

		return state, nil
	}
}
