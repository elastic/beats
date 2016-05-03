// +build darwin linux openbsd windows

/*

An example event looks as following:

	{
	  "@timestamp": "2016-05-03T15:11:04.610Z",
	  "beat": {
	    "hostname": "ruflin",
	    "name": "ruflin"
	  },
	  "metricset": "fsstats",
	  "module": "system",
	  "rtt": 84,
	  "system-fsstats": {
	    "count": 4,
	    "total_files": 60982450,
	    "total_size": {
	      "free": 32586960896,
	      "total": 249779548160,
	      "used": 217192587264
	    }
	  },
	  "type": "metricsets"
	}



*/

package fsstats

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/topbeat/system"
)

func init() {
	if err := mb.Registry.AddMetricSet("system", "fsstats", New); err != nil {
		panic(err)
	}
}

type MetricSet struct {
	mb.BaseMetricSet
}

// New creates new instance of MetricSeter
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return &MetricSet{
		BaseMetricSet: base,
	}, nil
}

func (m *MetricSet) Fetch(host string) (events common.MapStr, err error) {

	fss, err := system.GetFileSystemList()
	if err != nil {
		logp.Warn("Getting filesystem list: %v", err)
		return nil, err
	}

	// These values are optional and could also be calculated by Kibana
	var totalFiles, totalSize, totalSizeFree, totalSizeUsed uint64

	for _, fs := range fss {
		fsStat, err := system.GetFileSystemStat(fs)
		if err != nil {
			logp.Debug("filesystem", "Skip filesystem %d: %v", fsStat, err)
			continue
		}

		totalFiles += fsStat.Files
		totalSize += fsStat.Total
		totalSizeFree += fsStat.Free
		totalSizeUsed += fsStat.Used
	}

	event := common.MapStr{
		"total_size": common.MapStr{
			"free":  totalSizeFree,
			"used":  totalSizeUsed,
			"total": totalSize,
		},
		"count":       len(fss),
		"total_files": totalFiles,
	}

	return event, nil
}
