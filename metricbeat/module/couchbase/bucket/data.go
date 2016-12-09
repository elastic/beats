package bucket

import (
	"encoding/json"
	"io"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type BucketQuota struct {
	RAM    int64 `json:"ram"`
	RawRAM int64 `json:"rawRAM"`
}

type BucketBasicStats struct {
	QuotaPercentUsed float64 `json:"quotaPercentUsed"`
	OpsPerSec        int64   `json:"opsPerSec"`
	DiskFetches      int64   `json:"diskFetches"`
	ItemCount        int64   `json:"itemCount"`
	DiskUsed         int64   `json:"diskUsed"`
	DataUsed         int64   `json:"dataUsed"`
	MemUsed          int64   `json:"memUsed"`
}

type Buckets []struct {
	Name       string           `json:"name"`
	BucketType string           `json:"bucketType"`
	Quota      BucketQuota      `json:"quota"`
	BasicStats BucketBasicStats `json:"basicStats"`
}

func eventsMapping(body io.Reader) []common.MapStr {

	var d Buckets
	err := json.NewDecoder(body).Decode(&d)
	if err != nil {
		logp.Err("Error: ", err)
	}

	events := []common.MapStr{}

	for _, Bucket := range d {
		event := common.MapStr{
			"name": Bucket.Name,
			"type": Bucket.BucketType,
			"data": common.MapStr{
				"used.bytes": Bucket.BasicStats.DataUsed,
			},
			"disk": common.MapStr{
				"fetches":    Bucket.BasicStats.DiskFetches,
				"used.bytes": Bucket.BasicStats.DiskUsed,
			},
			"memory": common.MapStr{
				"used.bytes": Bucket.BasicStats.MemUsed,
			},
			"quota": common.MapStr{
				"ram.bytes": Bucket.Quota.RAM,
				"use.pct":   Bucket.BasicStats.QuotaPercentUsed,
			},
			"ops_per_sec": Bucket.BasicStats.OpsPerSec,
			"item_count":  Bucket.BasicStats.ItemCount,
		}

		events = append(events, event)
	}

	return events
}
