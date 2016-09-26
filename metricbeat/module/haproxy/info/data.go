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
		"nb_proc":       c.Int("Nbproc"),
		"process_num":   c.Int("ProcessNum"),
		"pid":           c.Int("Pid"),
		"uptime_sec":    c.Int("UptimeSec"),
		"mem_max_bytes": c.Int("MemMax"),
		"ulimit_n":      c.Int("UlimitN"),

		"compress": s.Object{
			"bps": s.Object{
				"in":         c.Int("CompressBpsIn"),
				"out":        c.Int("CompressBpsOut"),
				"rate_limit": c.Int("CompressBpsRateLim"),
			},
		},

		"conn": s.Object{
			"rate": s.Object{
				"value": c.Int("ConnRate"),
				"limit": c.Int("ConnRateLimit"),
			},
		},

		"curr": s.Object{
			"conns":     c.Int("CurrConns"),
			"ssl_conns": c.Int("CurrSslConns"),
		},

		"cum": s.Object{
			"conns":     c.Int("CumConns"),
			"req":       c.Int("CumReq"),
			"ssl_conns": c.Int("CumSslConns"),
		},

		"max": s.Object{
			"hard_conn": c.Int("HardMaxconn"),
			"ssl": s.Object{
				"conns": c.Int("MaxSslConns"),
				"rate":  c.Int("MaxSslRate"),
			},
			"sock": c.Int("Maxsock"),
			"conn": s.Object{
				"value": c.Int("Maxconn"),
				"rate":  c.Int("MaxConnRate"),
			},
			"sess_rate":      c.Int("MaxSessRate"),
			"pipes":          c.Int("Maxpipes"),
			"zlib_mem_usage": c.Int("MaxZlibMemUsage"),
		},

		"pipes": s.Object{
			"used": c.Int("PipesUsed"),
			"free": c.Int("PipesFree"),
		},

		"sess": s.Object{
			"rate": s.Object{
				"value": c.Int("SessRate"),
				"limit": c.Int("SessRateLimit"),
			},
		},

		"ssl": s.Object{
			"rate": s.Object{
				"value": c.Int("SslRate"),
				"limit": c.Int("SslRateLimit"),
			},
			"frontend": s.Object{
				"key_rate":          c.Int("SslFrontendKeyRate"),
				"max_key_rate":      c.Int("SslFrontendMaxKeyRate"),
				"session_reuse_pct": c.Int("SslFrontendSessionReusePct"),
			},
			"backend": s.Object{
				"key_rate":     c.Int("SslBackendKeyRate"),
				"max_key_rate": c.Int("SslBackendMaxKeyRate"),
			},
			"cached_lookups": c.Int("SslCacheLookups"),
			"cache_misses":   c.Int("SslCacheMisses"),
		},

		"zlib_mem_usage": c.Int("ZlibMemUsage"),
		"tasks":          c.Int("Tasks"),
		"run_queue":      c.Int("RunQueue"),
		"idle_pct":       c.Float("IdlePct"),
	}
)

// Map data to MapStr
func eventMapping(info *haproxy.Info) (common.MapStr, error) {
	// Full mapping from info

	st := reflect.ValueOf(info).Elem()
	typeOfT := st.Type()
	source := map[string]interface{}{}

	for i := 0; i < st.NumField(); i++ {
		f := st.Field(i)

		if typeOfT.Field(i).Name == "IdlePct" {
			// Convert this value to a float between 0.0 and 1.0
			fval, err := strconv.ParseFloat(f.Interface().(string), 64)
			if err != nil {
				return nil, err
			}
			source[typeOfT.Field(i).Name] = strconv.FormatFloat(fval/float64(100), 'f', 2, 64)
		} else if typeOfT.Field(i).Name == "Memmax_MB" {
			// Convert this value to bytes
			val, err := strconv.Atoi(strings.TrimSpace(f.Interface().(string)))
			if err != nil {
				return nil, err
			}
			source[typeOfT.Field(i).Name] = strconv.Itoa((val * 1024 * 1024))
		} else {
			source[typeOfT.Field(i).Name] = f.Interface()
		}

	}

	return schema.Apply(source), nil
}
