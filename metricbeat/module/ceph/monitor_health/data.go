package monitor_health

import (
	"encoding/json"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type Tick struct {
	time.Time
}

var format = "2006-01-02 15:04:05"

func (t *Tick) MarshalJSON() ([]byte, error) {
	return []byte(t.Time.Format(format)), nil
}

func (t *Tick) UnmarshalJSON(b []byte) (err error) {
	b = b[1 : len(b)-1]
	t.Time, err = time.Parse(format, string(b))
	return
}

type StoreStats struct {
	BytesTotal  int64  `json:"bytes_total"`
	BytesLog    int64  `json:"bytes_log"`
	LastUpdated string `json:"last_updated"`
	BytesMisc   int64  `json:"bytes_misc"`
	BytesSSt    int64  `json:"bytes_sst"`
}

type Mon struct {
	LastUpdated  Tick       `json:"last_updated"`
	Name         string     `json:"name"`
	AvailPercent int64      `json:"avail_percent"`
	KbTotal      int64      `json:"kb_total"`
	KbAvail      int64      `json:"kb_avail"`
	Health       string     `json:"health"`
	KbUsed       int64      `json:"kb_used"`
	StoreStats   StoreStats `json:"store_stats"`
}

type HealthServices struct {
	Mons []Mon `json:"mons"`
}

type Health struct {
	HealthServices []HealthServices `json:"health_services"`
}

type Timecheck struct {
	RoundStatus string `json:"round_status"`
	Epoch       int64  `json:"epoch"`
	Round       int64  `json:"round"`
}

type Output struct {
	OverallStatus string    `json:"overall_status"`
	Timechecks    Timecheck `json:"timechecks"`
	Health        Health    `json:"health"`
}

type HealthRequest struct {
	Status string `json:"status"`
	Output Output `json:"output"`
}

func eventsMapping(content []byte) []common.MapStr {
	var d HealthRequest
	err := json.Unmarshal(content, &d)
	if err != nil {
		logp.Err("Error: ", err)
	}

	events := []common.MapStr{}

	for _, HealthService := range d.Output.Health.HealthServices {
		for _, Mon := range HealthService.Mons {
			event := common.MapStr{
				"last_updated": Mon.LastUpdated,
				"name":         Mon.Name,
				"available": common.MapStr{
					"pct": Mon.AvailPercent,
					"kb":  Mon.KbAvail,
				},
				"total": common.MapStr{
					"kb": Mon.KbTotal,
				},
				"health": Mon.Health,
				"used": common.MapStr{
					"kb": Mon.KbUsed,
				},
				"store_stats": common.MapStr{
					"log": common.MapStr{
						"bytes": Mon.StoreStats.BytesLog,
					},
					"misc": common.MapStr{
						"bytes": Mon.StoreStats.BytesMisc,
					},
					"sst": common.MapStr{
						"bytes": Mon.StoreStats.BytesSSt,
					},
					"total": common.MapStr{
						"bytes": Mon.StoreStats.BytesTotal,
					},
					"last_updated": Mon.StoreStats.LastUpdated,
				},
			}

			events = append(events, event)
		}
	}

	return events
}
