package stat

import (
	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/metricbeat/schema"
	c "github.com/elastic/beats/metricbeat/schema/mapstrstr"
	"strings"
)

var (
	schema = s.Schema{
		"nb_proc":                        c.Int("Nbproc"),
		"process_num":                    c.Int("Process_num"),
		"pid":                            c.Int("Pid"),
		"uptime_sec":                     c.Int("Uptime_sec"),
		"mem_max_mb":                     c.Int("Memmax_MB"),
		"ulimit_n":                       c.Int("Ulimit-n"),
		"max_sock":                       c.Int("Maxsock"),
		"max_conn":                       c.Int("Maxconn"),
		"hard_max_conn":                  c.Init("Hard_maxconn"),
		"curr_conns":                     c.Init("CurrConns"),
		"cum_conns":                      c.Init("CumConns"),
		"cum_req":                        c.Init("CumReq"),
		"max_ssl_conns":                  c.Init("MaxSslConns"),
		"curr_ssl_conns":                 c.Init("CurrSslConns"),
		"cum_ssl_conns":                  c.Init("CumSslConns"),
		"max_pipes":                      c.Init("Maxpipes"),
		"pipes_used":                     c.Init("PipesUsed"),
		"pipes_free":                     c.Init("PipesFree"),
		"conn_rate":                      c.Init("ConnRate"),
		"conn_rate_limit":                c.Init("ConnRateLimit"),
		"max_conn_rate":                  c.Init("MaxConnRate"),
		"sess_rate":                      c.Init("SessRate"),
		"sess_rate_limit":                c.Init("SessRateLimit"),
		"max_sess_rate":                  c.Init("MaxSessRate"),
		"ssl_rate":                       c.Init("SslRate"),
		"ssl_rate_limit":                 c.Init("SslRateLimit"),
		"max_ssl_rate":                   c.Init("MaxSslRate"),
		"ssl_frontend_key_rate":          c.Init("SslFrontendKeyRate"),
		"ssl_frontend_max_key_rate":      c.Init("SslFrontendMaxKeyRate"),
		"ssl_frontend_session_reuse_pct": c.Init("SslFrontendSessionReuse_pct"),
		"ssl_babckend_key_rate":          c.Init("SslBackendKeyRate"),
		"ssl_backend_max_key_rate":       c.Init("SslBackendMaxKeyRate"),
		"ssl_cached_lookups":             c.Init("SslCacheLookups"),
		"ssl_cache_misses":               c.Init("SslCacheMisses"),
		"compress_bps_in":                c.Init("CompressBpsIn"),
		"compress_bps_out":               c.Init("CompressBpsOut"),
		"compress_bps_rate_limit":        c.Init("CompressBpsRateLim"),
		"zlib_mem_usage":                 c.Init("ZlibMemUsage"),
		"max_zlib_mem_usage":             c.Init("MaxZlibMemUsage"),
		"tasks":                          c.Init("Tasks"),
		"run_queue":                      c.Init("Run_queue"),
		"idle_pct":                       c.Init("Idle_pct"),
	}
)

func parseResponse(data []byte) map[string]string {
	resultMap := map[string]string{}
	str := string(data)
	for _, ln := range strings.Split(str, "\n") {
		parts := strings.Split(strings.Trim(ln, " "), ":")
		if parts[0] == "Name" || parts[0] == "Version" || parts[0] == "Release_date" || parts[0] == "Uptime" {
			continue
		}
		resultMap[parts[0]] = strings.Trim(parts[1], " ")
	}
}

// Map data to MapStr
func eventMapping(info map[string]string) common.MapStr {
	// Full mapping from info
	source := map[string]interface{}{}
	for key, val := range info {
		source[key] = val
	}
	return schema.Apply(source)
}
