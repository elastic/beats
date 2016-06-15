package status

import (
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

func eventMapping(status map[string]interface{}) common.MapStr {
	errs := map[string]error{}
	event := common.MapStr{
		"version": mustBeString("version", status, errs),
		"uptime": common.MapStr{
			"ms": mustBeInteger("uptimeMillis", status, errs),
		},
		"local_time":         mustBeTime("localTime", status, errs),
		"write_backs_queued": mustBeBool("writeBacksQueued", status, errs),
	}
	removeErroredKeys(event, errs)

	asserts, ok := status["asserts"].(map[string]interface{})
	if ok {
		errs := map[string]error{}
		asserts := common.MapStr{
			"regular":   mustBeInteger("regular", asserts, errs),
			"warning":   mustBeInteger("warning", asserts, errs),
			"msg":       mustBeInteger("msg", asserts, errs),
			"user":      mustBeInteger("user", asserts, errs),
			"rollovers": mustBeInteger("rollovers", asserts, errs),
		}
		removeErroredKeys(asserts, errs)
		event["asserts"] = asserts
	}

	backgroundFlushing, ok := status["backgroundFlushing"].(map[string]interface{})
	if ok {
		errs := map[string]error{}
		backgroundFlushing = common.MapStr{
			"flushes": mustBeInteger("flushes", backgroundFlushing, errs),
			"total": common.MapStr{
				"ms": mustBeInteger("total_ms", backgroundFlushing, errs),
			},
			"average": common.MapStr{
				"ms": mustBeInteger("average_ms", backgroundFlushing, errs),
			},
			"last": common.MapStr{
				"ms": mustBeInteger("last_ms", backgroundFlushing, errs),
			},
			"last_finished": mustBeTime("last_finished", backgroundFlushing, errs),
		}
		removeErroredKeys(backgroundFlushing, errs)
		event["background_flushing"] = backgroundFlushing
	}

	connections, ok := status["connections"].(map[string]interface{})
	if ok {
		errs := map[string]error{}
		eventConnections := common.MapStr{
			"current":       mustBeInteger("current", connections, errs),
			"available":     mustBeInteger("available", connections, errs),
			"total_created": mustBeInteger("totalCreated", connections, errs),
		}
		removeErroredKeys(eventConnections, errs)
		event["connections"] = eventConnections
	}

	dur, ok := status["dur"].(map[string]interface{})
	if ok {
		errs := map[string]error{}
		eventDur := common.MapStr{
			"commits": mustBeInteger("commits", dur, errs),
			"journaled": common.MapStr{
				"mb": mustBeInteger("journaledMB", dur, errs),
			},
			"write_to_data_files": common.MapStr{
				"mb": mustBeInteger("writeToDataFilesMB", dur, errs),
			},
			"compression":           mustBeInteger("compression", dur, errs),
			"commits_in_write_lock": mustBeInteger("commitsInWriteLock", dur, errs),
			"early_commits":         mustBeInteger("earlyCommits", dur, errs),
		}
		times, ok := dur["timeMs"].(map[string]interface{})
		if ok {
			errs := map[string]error{}
			times = common.MapStr{
				"dt":                    common.MapStr{"ms": mustBeInteger("dt", times, errs)},
				"prep_log_buffer":       common.MapStr{"ms": mustBeInteger("prepLogBuffer", times, errs)},
				"write_to_journal":      common.MapStr{"ms": mustBeInteger("writeToJournal", times, errs)},
				"write_to_data_files":   common.MapStr{"ms": mustBeInteger("writeToDataFiles", times, errs)},
				"remap_private_view":    common.MapStr{"ms": mustBeInteger("remapPrivateView", times, errs)},
				"commits":               common.MapStr{"ms": mustBeInteger("commits", times, errs)},
				"commits_in_write_lock": common.MapStr{"ms": mustBeInteger("commitsInWriteLock", times, errs)},
			}
			removeErroredKeys(times, errs)
			eventDur["times"] = times
		}
		removeErroredKeys(eventDur, errs)
		event["journaling"] = eventDur
	}

	extraInfo, ok := status["extra_info"].(map[string]interface{})
	if ok {
		errs := map[string]error{}
		eventExtraInfo := common.MapStr{
			"heap_usage":  common.MapStr{"bytes": mustBeInteger("heap_usage_bytes", extraInfo, errs)},
			"page_faults": mustBeInteger("page_faults", extraInfo, errs),
		}
		removeErroredKeys(eventExtraInfo, errs)
		event["extra_info"] = eventExtraInfo
	}

	network, ok := status["network"].(map[string]interface{})
	if ok {
		errs := map[string]error{}
		eventNetwork := common.MapStr{
			"in":       common.MapStr{"bytes": mustBeInteger("bytesIn", network, errs)},
			"out":      common.MapStr{"bytes": mustBeInteger("bytesOut", network, errs)},
			"requests": mustBeInteger("numRequests", network, errs),
		}
		removeErroredKeys(eventNetwork, errs)
		event["network"] = eventNetwork
	}

	opcounters, ok := status["opcounters"].(map[string]interface{})
	if ok {
		errs := map[string]error{}
		eventOpcounters := common.MapStr{}
		for key, _ := range opcounters {
			eventOpcounters[key] = mustBeInteger(key, opcounters, errs)
		}
		removeErroredKeys(eventOpcounters, errs)
		event["opcounters"] = eventOpcounters
	}

	opcountersRepl, ok := status["opcountersRepl"].(map[string]interface{})
	if ok {
		errs := map[string]error{}
		eventOpcountersRepl := common.MapStr{}
		for key, _ := range opcountersRepl {
			eventOpcountersRepl[key] = mustBeInteger(key, opcountersRepl, errs)
		}
		removeErroredKeys(eventOpcountersRepl, errs)
		event["opcounters_replicated"] = eventOpcountersRepl
	}

	mem, ok := status["mem"].(map[string]interface{})
	if ok {
		errs := map[string]error{}
		eventMem := common.MapStr{
			"bits":                mustBeInteger("bits", mem, errs),
			"resident":            common.MapStr{"mb": mustBeInteger("resident", mem, errs)},
			"virtual":             common.MapStr{"mb": mustBeInteger("virtual", mem, errs)},
			"mapped":              common.MapStr{"mb": mustBeInteger("mapped", mem, errs)},
			"mapped_with_journal": common.MapStr{"mb": mustBeInteger("mappedWithJournal", mem, errs)},
		}
		removeErroredKeys(eventMem, errs)
		event["memory"] = eventMem
	}

	storageEngine, ok := status["storageEngine"].(map[string]interface{})
	if ok {
		errs := map[string]error{}
		eventStorageEngine := common.MapStr{
			"name": mustBeString("name", storageEngine, errs),
		}
		removeErroredKeys(eventStorageEngine, errs)
		event["storage_engine"] = eventStorageEngine
	}

	return event
}

func mustBeString(key string, data map[string]interface{}, errs map[string]error) string {
	emptyIface, exists := data[key]
	if !exists {
		errs[key] = fmt.Errorf("Key not found")
		return ""
	}
	str, ok := emptyIface.(string)
	if !ok {
		errs[key] = fmt.Errorf("Expected string, found %T", emptyIface)
		return ""
	}
	return str
}

func mustBeBool(key string, data map[string]interface{}, errs map[string]error) bool {
	emptyIface, exists := data[key]
	if !exists {
		errs[key] = fmt.Errorf("Key not found")
		return false
	}
	boolean, ok := emptyIface.(bool)
	if !ok {
		errs[key] = fmt.Errorf("Expected bool, found %T", emptyIface)
		return false
	}
	return boolean
}

func mustBeInteger(key string, data map[string]interface{}, errs map[string]error) int64 {
	emptyIface, exists := data[key]
	if !exists {
		errs[key] = fmt.Errorf("Key not found")
		return 0
	}
	switch emptyIface.(type) {
	case int64:
		return emptyIface.(int64)
	case int:
		return int64(emptyIface.(int))
	case float64:
		return int64(emptyIface.(float64))
	default:
		errs[key] = fmt.Errorf("Expected integer, found %T", emptyIface)
		return 0
	}
}

func mustBeTime(key string, data map[string]interface{}, errs map[string]error) common.Time {
	emptyIface, exists := data[key]
	if !exists {
		errs[key] = fmt.Errorf("Key not found")
		return common.Time(time.Unix(0, 0))
	}
	ts, ok := emptyIface.(time.Time)
	if !ok {
		errs[key] = fmt.Errorf("Expected date, found %T", emptyIface)
		return common.Time(time.Unix(0, 0))
	}
	return common.Time(ts)
}

func removeErroredKeys(event common.MapStr, errs map[string]error) {
	for key, err := range errs {
		logp.Err("Error on key `%s`: %v", key, err)
		delete(event, key)
	}
}
