package info

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/module/haproxy"
	s "github.com/elastic/beats/metricbeat/schema"
	c "github.com/elastic/beats/metricbeat/schema/mapstrstr"
	"reflect"
	"strconv"
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
		"hard_max_conn":                  c.Int("Hard_maxconn"),
		"curr_conns":                     c.Int("CurrConns"),
		"cum_conns":                      c.Int("CumConns"),
		"cum_req":                        c.Int("CumReq"),
		"max_ssl_conns":                  c.Int("MaxSslConns"),
		"curr_ssl_conns":                 c.Int("CurrSslConns"),
		"cum_ssl_conns":                  c.Int("CumSslConns"),
		"max_pipes":                      c.Int("Maxpipes"),
		"pipes_used":                     c.Int("PipesUsed"),
		"pipes_free":                     c.Int("PipesFree"),
		"conn_rate":                      c.Int("ConnRate"),
		"conn_rate_limit":                c.Int("ConnRateLimit"),
		"max_conn_rate":                  c.Int("MaxConnRate"),
		"sess_rate":                      c.Int("SessRate"),
		"sess_rate_limit":                c.Int("SessRateLimit"),
		"max_sess_rate":                  c.Int("MaxSessRate"),
		"ssl_rate":                       c.Int("SslRate"),
		"ssl_rate_limit":                 c.Int("SslRateLimit"),
		"max_ssl_rate":                   c.Int("MaxSslRate"),
		"ssl_frontend_key_rate":          c.Int("SslFrontendKeyRate"),
		"ssl_frontend_max_key_rate":      c.Int("SslFrontendMaxKeyRate"),
		"ssl_frontend_session_reuse_pct": c.Int("SslFrontendSessionReuse_pct"),
		"ssl_babckend_key_rate":          c.Int("SslBackendKeyRate"),
		"ssl_backend_max_key_rate":       c.Int("SslBackendMaxKeyRate"),
		"ssl_cached_lookups":             c.Int("SslCacheLookups"),
		"ssl_cache_misses":               c.Int("SslCacheMisses"),
		"compress_bps_in":                c.Int("CompressBpsIn"),
		"compress_bps_out":               c.Int("CompressBpsOut"),
		"compress_bps_rate_limit":        c.Int("CompressBpsRateLim"),
		"zlib_mem_usage":                 c.Int("ZlibMemUsage"),
		"max_zlib_mem_usage":             c.Int("MaxZlibMemUsage"),
		"tasks":                          c.Int("Tasks"),
		"run_queue":                      c.Int("Run_queue"),
		"idle_pct":                       c.Float("Idle_pct"),
	}
)

func parseResponse(data []byte) map[string]string {

	resultMap := map[string]string{}
	str := string(data)

	for _, ln := range strings.Split(str, "\n") {

		ln := strings.Trim(ln, " ")
		if ln == "" {
			continue
		}

		parts := strings.Split(strings.Trim(ln, " "), ":")
		if len(parts) != 2 {
			continue
		}

		if parts[0] == "Name" || parts[0] == "Version" || parts[0] == "Release_date" || parts[0] == "Uptime" || parts[0] == "node" || parts[0] == "description" {
			continue
		}

		if parts[0] == "Idle_pct" {
			// Convert this value to a float between 0.0 and 1.0
			f, _ := strconv.ParseFloat(parts[1], 64)
			resultMap[parts[0]] = strconv.FormatFloat(f/float64(100), 'f', 2, 64)
		} else if parts[0] == "Memmax_MB" {
			// Convert this value to bytes
			val, _ := strconv.Atoi(strings.Trim(parts[1], " "))
			resultMap[parts[0]] = strconv.Itoa((val * 1024 * 1024))
		} else {
			resultMap[parts[0]] = strings.Trim(parts[1], " ")
		}
	}
	return resultMap
}

// Map data to MapStr
func eventMapping(info *haproxy.Info) common.MapStr {
	// Full mapping from info

	source := map[string]interface{}{}

	st := reflect.ValueOf(info).Elem()
	typeOfT := st.Type()
	source = map[string]interface{}{}

	for i := 0; i < st.NumField(); i++ {
		f := st.Field(i)
		source[typeOfT.Field(i).Name] = f.Interface()

	}

	return schema.Apply(source)
}
