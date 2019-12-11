package iis

var (
	website_counters = map[string]string{
		"website.bytes_sent_per_sec":       "\\Web Service(*)\\Bytes Sent/sec",
		"website.total_bytes_sent_per_sec": "\\Web Service(*)\\Total Bytes Sent",
		"website.bytes_recv_per_sec":       "\\Web Service(*)\\Bytes Received/sec",
		"website.total_bytes_recv_per_sec": "\\Web Service(*)\\Total Bytes Received",

		//\Web Service(*)\Total Files Sent
		//\Web Service(*)\Files Sent/sec
		//\Web Service(*)\Total Files Received
		//\Web Service(*)\Files Received/sec
		//\Web Service(*)\Current Connections
		//\Web Service(*)\Maximum Connections
		//\Web Service(*)\Total Connection Attempts (all instances)
		//\Web Service(*)\Total Get Requests
		//\Web Service(*)\Get Requests/sec
		//\Web Service(*)\Total Post Requests
		//\Web Service(*)\Post Requests/sec
	}
	webserver_counters = map[string]string{
		"webserver.total_bytes_sent_per_sec": "\\Web Service(_Total)\\Total Bytes Sent",
		"webserver.total_bytes_recv_per_sec": "\\Web Service(_Total)\\Total Bytes Received",
		//\Web Service(*)\Total Files Sent
		//\Web Service(*)\Files Sent/sec
		//\Web Service(*)\Total Files Received
		//\Web Service(*)\Files Received/sec
		//\Web Service(*)\Current Connections
		//\Web Service(*)\Maximum Connections
		//\Web Service(*)\Total Connection Attempts (all instances)
		//\Web Service(*)\Total Get Requests
		//\Web Service(*)\Get Requests/sec
		//\Web Service(*)\Total Post Requests
		//\Web Service(*)\Post Requests/sec

		//cache
		//"cache": {
		//"file_cache_count": "2",
		//"file_cache_memory_usage": "699",
		//"file_cache_hits": "18506471",
		//"file_cache_misses": "46266060",
		//"total_files_cached": "10",
		//"output_cache_count": "0",
		//"output_cache_memory_usage": "0",
		//"output_cache_hits": "0",
		//"output_cache_misses": "18506478",
		//"uri_cache_count": "2",
		//"uri_cache_hits": "18506452",
		//"uri_cache_misses": "26",
		//"total_uris_cached": "13"
		//}

	}
)

type PerformanceCounter struct {
	InstanceLabel    string
	MeasurementLabel string
	Path             string
	Format           string
}

func GetPerfCounters(metricset string) []PerformanceCounter {
	var counters []PerformanceCounter
	switch metricset {
	case "website":
		for k, v := range website_counters {
			counter := PerformanceCounter{
				InstanceLabel:    "website.name",
				MeasurementLabel: k,
				Path:             v,
				Format:           "float",
			}
			counters = append(counters, counter)
		}
	case "webserver":
		for k, v := range webserver_counters {
			counter := PerformanceCounter{
				InstanceLabel:    "webserver.name",
				MeasurementLabel: k,
				Path:             v,
				Format:           "float",
			}
			counters = append(counters, counter)
		}

	}
	return counters
}
