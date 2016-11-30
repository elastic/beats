package bucket

import (
	"encoding/json"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"io"
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
			"name":               Bucket.Name,
			"bucketType":         Bucket.BucketType,
			"quota_RAM":          Bucket.Quota.RAM,
			"quota_RawRAM":       Bucket.Quota.RawRAM,
			"stats_QuotaPercUse": Bucket.BasicStats.QuotaPercentUsed,
			"stats_OpsPerSec":    Bucket.BasicStats.OpsPerSec,
			"stats_DiskFetches":  Bucket.BasicStats.DiskFetches,
			"stats_ItemCount":    Bucket.BasicStats.ItemCount,
			"stats_DiskUsed":     Bucket.BasicStats.DiskUsed,
			"stats_DataUsed":     Bucket.BasicStats.DataUsed,
			"stats_MemUsed":      Bucket.BasicStats.MemUsed,
		}

		events = append(events, event)
	}

	return events
}
