package stat

import (
	"reflect"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/module/haproxy"
	s "github.com/elastic/beats/metricbeat/schema"
	c "github.com/elastic/beats/metricbeat/schema/mapstrstr"
)

var (
	schema = s.Schema{
		"pxname":         c.Str("PxName"),
		"svname":         c.Str("SvName"),
		"qcur":           c.Int("Qcur"),
		"qmax":           c.Int("Qmax"),
		"scur":           c.Int("Scur"),
		"smax":           c.Int("Smax"),
		"slim":           c.Int("Slim"),
		"stot":           c.Int("Stot"),
		"bin":            c.Int("Bin"),
		"bout":           c.Int("Bout"),
		"dreq":           c.Int("Dreq"),
		"dresp":          c.Int("Dresp"),
		"ereq":           c.Int("Ereq"),
		"econ":           c.Int("Econ"),
		"eresp":          c.Int("Eresp"),
		"wretr":          c.Int("Wretr"),
		"wredis":         c.Int("Wredis"),
		"status":         c.Str("Status"),
		"weight":         c.Int("Weight"),
		"act":            c.Int("Act"),
		"bck":            c.Int("Bck"),
		"chkfail":        c.Int("ChkFail"),
		"chkdown":        c.Int("ChkDown"),
		"lastchg":        c.Int("Lastchg"),
		"downtime":       c.Int("Downtime"),
		"qlimit":         c.Int("Qlimit"),
		"pid":            c.Int("Pid"),
		"iid":            c.Int("Iid"),
		"sid":            c.Int("Sid"),
		"throttle":       c.Int("Throttle"),
		"lbtot":          c.Int("Lbtot"),
		"tracked":        c.Int("Tracked"),
		"component_type": c.Int("Type"),

		"rate": s.Object{
			"value": c.Int("Rate"),
			"lim":   c.Int("RateLim"),
			"max":   c.Int("RateMax"),
		},

		"check": s.Object{
			"status":   c.Str("CheckStatus"),
			"code":     c.Int("CheckCode"),
			"duration": c.Int("CheckDuration"),
		},

		"hrsp": s.Object{
			"1xx":   c.Int("Hrsp1xx"),
			"2xx":   c.Int("Hrsp2xx"),
			"3xx":   c.Int("Hrsp3xx"),
			"4xx":   c.Int("Hrsp4xx"),
			"5xx":   c.Int("Hrsp5xx"),
			"other": c.Int("HrspOther"),
		},

		"hanafail": c.Int("Hanafail"),

		"req": s.Object{
			"rate": s.Object{
				"value": c.Int("ReqRate"),
				"max":   c.Int("ReqRateMax"),
			},
			"tot": c.Int("ReqTot"),
		},

		"cli_abrt": c.Int("CliAbrt"),
		"srv_abrt": c.Int("SrvAbrt"),

		"comp": s.Object{
			"in":  c.Int("CompIn"),
			"out": c.Int("CompOut"),
			"byp": c.Int("CompByp"),
			"rsp": c.Int("CompRsp"),
		},

		"last": s.Object{
			"sess": c.Int("LastSess"),
			"chk":  c.Str("LastChk"),
			"agt":  c.Str("LastAgt"),
		},

		"qtime": c.Int("Qtime"),
		"ctime": c.Int("Ctime"),
		"rtime": c.Int("Rtime"),
		"ttime": c.Int("Ttime"),
	}
)

// Map data to MapStr
func eventMapping(info []*haproxy.Stat) []common.MapStr {

	var events []common.MapStr

	for _, evt := range info {
		st := reflect.ValueOf(evt).Elem()
		typeOfT := st.Type()
		source := map[string]interface{}{}

		for i := 0; i < st.NumField(); i++ {
			f := st.Field(i)
			source[typeOfT.Field(i).Name] = f.Interface()

		}
		events = append(events, schema.Apply(source))
	}

	return events
}
