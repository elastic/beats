package health

import (
	"encoding/json"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"io"
	"time"
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

func eventsMapping(body io.Reader) []common.MapStr {

	var d HealthRequest
	err := json.NewDecoder(body).Decode(&d)
	if err != nil {
		logp.Err("Error: ", err)
	}

	events := []common.MapStr{}

	event := common.MapStr{
		"cluster.overall_status": d.Output.OverallStatus,
		"cluster.timechecks": common.MapStr{
			"round_status": d.Output.Timechecks.RoundStatus,
			"epoch":        d.Output.Timechecks.Epoch,
			"round":        d.Output.Timechecks.Round,
		},
	}

	events = append(events, event)

	for _, HealthService := range d.Output.Health.HealthServices {
		for _, Mon := range HealthService.Mons {
			event := common.MapStr{
				"mon": common.MapStr{
					"last_updated":  Mon.LastUpdated,
					"name":          Mon.Name,
					"avail_percent": Mon.AvailPercent,
					"kb_total":      Mon.KbTotal,
					"kb_avail":      Mon.KbAvail,
					"health":        Mon.Health,
					"kb_used":       Mon.KbUsed,
					"store_stats": common.MapStr{
						"bytes_total":  Mon.StoreStats.BytesTotal,
						"bytes_log":    Mon.StoreStats.BytesLog,
						"last_updated": Mon.StoreStats.LastUpdated,
						"bytes_misc":   Mon.StoreStats.BytesMisc,
						"bytes_sst":    Mon.StoreStats.BytesSSt,
					},
				},
			}

			events = append(events, event)
		}

	}

	return events
}
