// +build darwin,cgo freebsd linux windows

package diskio

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"

	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/disk"
)

func init() {
	if err := mb.Registry.AddMetricSet("system", "diskio", New, parse.EmptyHostParser); err != nil {
		panic(err)
	}
}

// MetricSet for fetching system disk IO metrics.
type MetricSet struct {
	mb.BaseMetricSet
	statistics *DiskIOStat
}

// New is a mb.MetricSetFactory that returns a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms := &MetricSet{
		BaseMetricSet: base,
		statistics:    NewDiskIOStat(),
	}
	return ms, nil
}

// Fetch fetches disk IO metrics from the OS.
func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	stats, err := disk.IOCounters()
	if err != nil {
		return nil, errors.Wrap(err, "disk io counters")
	}

	// open a sampling means sample the current cpu counter
	m.statistics.OpenSampling()

	events := make([]common.MapStr, 0, len(stats))
	for _, counters := range stats {

		event := common.MapStr{
			"name": counters.Name,
			"read": common.MapStr{
				"count": counters.ReadCount,
				"time":  counters.ReadTime,
				"bytes": counters.ReadBytes,
			},
			"write": common.MapStr{
				"count": counters.WriteCount,
				"time":  counters.WriteTime,
				"bytes": counters.WriteBytes,
			},
			"io": common.MapStr{
				"time": counters.IoTime,
			},
		}

		extraMetrics, err := m.statistics.CalIOStatistics(counters)
		if err == nil {
			event["iostat"] = common.MapStr{
				"read_request_merged_per_sec":  extraMetrics.ReadRequestMergeCountPerSec,
				"write_request_merged_per_sec": extraMetrics.WriteRequestMergeCountPerSec,
				"read_request_per_sec":         extraMetrics.ReadRequestCountPerSec,
				"write_request_per_sec":        extraMetrics.WriteRequestCountPerSec,
				"read_byte_per_sec":            extraMetrics.ReadBytesPerSec,
				"write_byte_per_sec":           extraMetrics.WriteBytesPerSec,
				"avg_request_size":             extraMetrics.AvgRequestSize,
				"avg_queue_size":               extraMetrics.AvgQueueSize,
				"await":                        extraMetrics.AvgAwaitTime,
				"service_time":                 extraMetrics.AvgServiceTime,
				"busy":                         extraMetrics.BusyPct,
			}
		}

		events = append(events, event)

		if counters.SerialNumber != "" {
			event["serial_number"] = counters.SerialNumber
		}
	}

	// open a sampling means store the last cpu counter
	m.statistics.CloseSampling()

	return events, nil
}
