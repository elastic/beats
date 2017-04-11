package stats

import (
	"encoding/json"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/metricbeat/schema"
	c "github.com/elastic/beats/metricbeat/schema/mapstriface"
)

var (
	shards = s.Schema{
		"shards": c.Dict("_shards", s.Schema{
			"total":      c.Int("total"),
			"successful": c.Int("successful"),
			"failed":     c.Int("failed"),
		}),
	}

	total = s.Schema{
		"docs": c.Dict("docs", s.Schema{
			"count":   c.Int("count"),
			"deleted": c.Int("deleted"),
		}),
		"store": c.Dict("store", s.Schema{
			"size": s.Object{
				"bytes": c.Int("size_in_bytes"),
			},
		}),
		"segments": c.Dict("segments", s.Schema{
			"count": c.Int("count"),
			"memory": s.Object{
				"bytes": c.Int("memory_in_bytes"),
			},
		}),
	}
)

func eventMapping(content []byte) (common.MapStr, error) {

	// Empty struct needed every time
	var allStruct struct {
		All struct {
			Total map[string]interface{} `json:"total"`
		} `json:"_all"`
	}
	var shardsStruct map[string]interface{}

	json.Unmarshal(content, &allStruct)

	// This happens before elasticsearch has any shards. Return empty document.
	if len(allStruct.All.Total) == 0 {
		return common.MapStr{}, nil
	}

	json.Unmarshal(content, &shardsStruct)

	allData, errs1 := total.Apply(allStruct.All.Total)
	shards, errs2 := shards.Apply(shardsStruct)
	errs1.AddErrors(errs2)

	allData.Update(shards)
	return allData, errs1
}
