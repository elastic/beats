package stat

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/module/haproxy"
	s "github.com/elastic/beats/metricbeat/schema"
	c "github.com/elastic/beats/metricbeat/schema/mapstrstr"
	"reflect"
	"strings"
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
		"type":           c.Int("Type"),
		"rate":           c.Int("Rate"),
		"rate_lim":       c.Int("RateLim"),
		"rate_max":       c.Int("RateMax"),
		"check_status":   c.Str("CheckStatus"),
		"check_code":     c.Int("CheckCode"),
		"check_duration": c.Int("CheckDuration"),
		"hrsp_1xx":       c.Int("Hrsp1xx"),
		"hrsp_2xx":       c.Int("Hrsp2xx"),
		"hrsp_3xx":       c.Int("Hrsp3xx"),
		"hrsp_4xx":       c.Int("Hrsp4xx"),
		"hrsp_5xx":       c.Int("Hrsp5xx"),
		"hrsp_other":     c.Int("HrspOther"),
		"hanafail":       c.Int("Hanafail"),
		"req_rate":       c.Int("ReqRate"),
		"req_rate_max":   c.Int("ReqRateMax"),
		"req_tot":        c.Int("ReqTot"),
		"cli_abrt":       c.Int("CliAbrt"),
		"srv_abrt":       c.Int("SrvAbrt"),
		"comp_in":        c.Int("CompIn"),
		"comp_out":       c.Int("CompOut"),
		"comp_byp":       c.Int("CompByp"),
		"comp_rsp":       c.Int("CompRsp"),
		"lastsess":       c.Int("LastSess"),
		"last_chk":       c.Str("LastChk"),
		"last_agt":       c.Int("LastAgt"),
		"qtime":          c.Int("Qtime"),
		"ctime":          c.Int("Ctime"),
		"rtime":          c.Int("Rtime"),
		"ttime":          c.Int("Ttime"),
	}
)

func parseResponse(data []byte) []map[string]string {

	var results []map[string]string

	str := string(data)
	fieldNames := []string{}

	for lnNum, ln := range strings.Split(str, "\n") {

		// If the line by any chance is empty, then skip it
		ln := strings.Trim(ln, " ")
		if ln == "" {
			continue
		}

		// Now split the line on each comma and if there isn
		ln = strings.Trim(ln, ",")
		parts := strings.Split(strings.Trim(ln, " "), ",")
		if len(parts) != 62 {
			continue
		}

		// For the first row, keep the column names and continue
		if lnNum == 0 {
			fieldNames = parts
			continue
		}

		res := map[string]string{}
		for i, v := range parts {
			res[fieldNames[i]] = v
		}

		results = append(results, res)

	}
	return results
}

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
