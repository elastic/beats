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
		"qcur":           c.Int("Qcur", s.Optional),
		"qmax":           c.Int("Qmax", s.Optional),
		"scur":           c.Int("Scur"),
		"smax":           c.Int("Smax"),
		"slim":           c.Int("Slim", s.Optional),
		"stot":           c.Int("Stot"),
		"bin":            c.Int("Bin"),
		"bout":           c.Int("Bout"),
		"dreq":           c.Int("Dreq", s.Optional),
		"dresp":          c.Int("Dresp"),
		"ereq":           c.Int("Ereq", s.Optional),
		"econ":           c.Int("Econ", s.Optional),
		"eresp":          c.Int("Eresp", s.Optional),
		"wretr":          c.Int("Wretr", s.Optional),
		"wredis":         c.Int("Wredis", s.Optional),
		"status":         c.Str("Status"),
		"weight":         c.Int("Weight", s.Optional),
		"act":            c.Int("Act", s.Optional),
		"bck":            c.Int("Bck", s.Optional),
		"chkfail":        c.Int("ChkFail", s.Optional),
		"chkdown":        c.Int("ChkDown", s.Optional),
		"lastchg":        c.Int("Lastchg", s.Optional),
		"downtime":       c.Int("Downtime", s.Optional),
		"qlimit":         c.Int("Qlimit", s.Optional),
		"pid":            c.Int("Pid"),
		"iid":            c.Int("Iid"),
		"sid":            c.Int("Sid"),
		"throttle":       c.Int("Throttle", s.Optional),
		"lbtot":          c.Int("Lbtot", s.Optional),
		"tracked":        c.Int("Tracked", s.Optional),
		"component_type": c.Int("Type"),

		"rate": s.Object{
			"value": c.Int("Rate", s.Optional),
			"lim":   c.Int("RateLim", s.Optional),
			"max":   c.Int("RateMax", s.Optional),
		},

		"check": s.Object{
			"status":   c.Str("CheckStatus"),
			"code":     c.Int("CheckCode", s.Optional),
			"duration": c.Int("CheckDuration", s.Optional),
		},

		"hrsp": s.Object{
			"1xx":   c.Int("Hrsp1xx"),
			"2xx":   c.Int("Hrsp2xx"),
			"3xx":   c.Int("Hrsp3xx"),
			"4xx":   c.Int("Hrsp4xx"),
			"5xx":   c.Int("Hrsp5xx"),
			"other": c.Int("HrspOther"),
		},

		"hanafail": c.Int("Hanafail", s.Optional),

		"req": s.Object{
			"rate": s.Object{
				"value": c.Int("ReqRate", s.Optional),
				"max":   c.Int("ReqRateMax", s.Optional),
			},
			"tot": c.Int("ReqTot", s.Optional),
		},

		"cli_abrt": c.Int("CliAbrt", s.Optional),
		"srv_abrt": c.Int("SrvAbrt", s.Optional),

		"comp": s.Object{
			"in":  c.Int("CompIn", s.Optional),
			"out": c.Int("CompOut", s.Optional),
			"byp": c.Int("CompByp", s.Optional),
			"rsp": c.Int("CompRsp", s.Optional),
		},

		"last": s.Object{
			"sess": c.Int("LastSess", s.Optional),
			"chk":  c.Str("LastChk"),
			"agt":  c.Str("LastAgt"),
		},

		"qtime": c.Int("Qtime", s.Optional),
		"ctime": c.Int("Ctime", s.Optional),
		"rtime": c.Int("Rtime", s.Optional),
		"ttime": c.Int("Ttime", s.Optional),
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
