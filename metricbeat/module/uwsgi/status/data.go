package status

import (
	"encoding/json"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type uwsgiCore struct {
	ID                int `json:"id"`
	Requests          int `json:"requests"`
	StaticRequests    int `json:"static_requests"`
	RoutedRequests    int `json:"routed_requests"`
	OffloadedRequests int `json:"offloaded_requests"`
	WriteErrors       int `json:"write_errors"`
	ReadErrors        int `json:"read_errors"`

	// omitted
	// InRequest         int `json:"in_request"`
}

type uwsgiWorker struct {
	ID            int         `json:"id"`
	PID           int         `json:"pid"`
	Accepting     int         `json:"accepting"`
	Requests      int         `json:"requests"`
	DeltaRequests int         `json:"delta_requests"`
	Exceptions    int         `json:"exceptions"`
	HarakiriCount int         `json:"harakiri_count"`
	Signals       int         `json:"signals"`
	SignalQueue   int         `json:"signal_queue"`
	Status        string      `json:"status"`
	RSS           int         `json:"rss"`
	VSZ           int         `json:"vsz"`
	RunningTime   int         `json:"running_time"`
	LastSpawn     int64       `json:"last_spawn"`
	RespawnCount  int         `json:"respawn_count"`
	Tx            int         `json:"tx"`
	AvgRt         int         `json:"avg_rt"`
	Cores         []uwsgiCore `json:"cores"`

	// omitted
	// Apps []UwsgiApp `json:"apps"`
}

type uwsgiStat struct {
	Version           string        `json:"version"`
	ListenQueue       int           `json:"listen_queue"`
	ListenQueueErrors int           `json:"listen_queue_errors"`
	SignalQueue       int           `json:"signal_queue"`
	Load              int           `json:"load"`
	PID               int           `json:"pid"`
	Workers           []uwsgiWorker `json:"workers"`

	// omitted
	// Locks []map[string]int `json:"locks"`
	// Sockets []UwsgiSocket `json:"sockets"`
}

func eventsMapping(content []byte) ([]common.MapStr, error) {
	var stats uwsgiStat
	err := json.Unmarshal(content, &stats)
	if err != nil {
		logp.Err("uwsgi statistics parsing failed with error: ", err)
		return nil, err
	}

	events := []common.MapStr{}
	totalRequests := 0
	totalExceptions := 0
	totalWriteErrors := 0
	totalReadErrors := 0
	coreID := 1

	// worker cores info
	for _, worker := range stats.Workers {
		workerEvent := common.MapStr{
			"worker": common.MapStr{
				"id":             worker.ID,
				"pid":            worker.PID,
				"accepting":      worker.Accepting,
				"requests":       worker.Requests,
				"delta_requests": worker.DeltaRequests,
				"exceptions":     worker.Exceptions,
				"harakiri_count": worker.HarakiriCount,
				"signals":        worker.Signals,
				"signal_queue":   worker.SignalQueue,
				"status":         worker.Status,
				"rss":            worker.RSS,
				"vsz":            worker.VSZ,
				"running_time":   worker.RunningTime,
				"respawn_count":  worker.RespawnCount,
				"tx":             worker.Tx,
				"avg_rt":         worker.AvgRt,
			},
		}
		totalRequests += worker.Requests
		totalExceptions += worker.Exceptions

		for _, core := range worker.Cores {
			totalWriteErrors += core.WriteErrors
			totalReadErrors += core.ReadErrors

			coreEvent := common.MapStr{
				"core": common.MapStr{
					"id":         coreID,
					"worker_pid": worker.PID,
					"requests": common.MapStr{
						"total":     core.Requests,
						"static":    core.StaticRequests,
						"routed":    core.RoutedRequests,
						"offloaded": core.OffloadedRequests,
					},
					"write_errors": core.WriteErrors,
					"read_errors":  core.ReadErrors,
				},
			}
			events = append(events, coreEvent)
			coreID++
		}

		events = append(events, workerEvent)
	}

	// overall
	baseEvent := common.MapStr{
		"total": common.MapStr{
			"requests":     totalRequests,
			"exceptions":   totalExceptions,
			"write_errors": totalWriteErrors,
			"read_errors":  totalReadErrors,
			"pid":          stats.PID,
		},
	}

	events = append(events, baseEvent)
	return events, nil
}
