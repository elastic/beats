package stat

import (
	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/metricbeat/schema"
	c "github.com/elastic/beats/metricbeat/schema/mapstrstr"
	"strings"
)

var (
	schema = s.Schema{
		"pxname":         c.Str("pxname"),
		"svname":         c.Str("svname"),
		"qcur":           c.Int("qcur"),
		"qmax":           c.Int("qmax"),
		"scur":           c.Int("scur"),
		"smax":           c.Int("smax"),
		"slim":           c.Int("slim"),
		"stot":           c.Int("stot"),
		"bin":            c.Int("bin"),
		"bout":           c.Int("bout"),
		"dreq":           c.Int("dreq"),
		"dresp":          c.Int("dresp"),
		"ereq":           c.Int("ereq"),
		"econ":           c.Int("econ"),
		"eresp":          c.Int("eresp"),
		"wretr":          c.Int("wretr"),
		"wredis":         c.Int("wredis"),
		"status":         c.Str("status"),
		"weight":         c.Int("weight"),
		"act":            c.Int("act"),
		"bck":            c.Int("bck"),
		"chkfail":        c.Int("chkfail"),
		"chkdown":        c.Int("chkdown"),
		"lastchg":        c.Int("lastchg"),
		"downtime":       c.Int("downtime"),
		"qlimit":         c.Int("qlimit"),
		"pid":            c.Int("pid"),
		"iid":            c.Int("iid"),
		"sid":            c.Int("sid"),
		"throttle":       c.Int("throttle"),
		"lbtot":          c.Int("lbtot"),
		"tracked":        c.Int("tracked"),
		"type":           c.Int("type"),
		"rate":           c.Int("rate"),
		"rate_lim":       c.Int("rate_lim"),
		"rate_max":       c.Int("rate_max"),
		"check_status":   c.Str("check_status"),
		"check_code":     c.Int("check_code"),
		"check_duration": c.Int("check_duration"),
		"hrsp_1xx":       c.Int("hrsp_1xx"),
		"hrsp_2xx":       c.Int("hrsp_2xx"),
		"hrsp_3xx":       c.Int("hrsp_3xx"),
		"hrsp_4xx":       c.Int("hrsp_4xx"),
		"hrsp_5xx":       c.Int("hrsp_5xx"),
		"hrsp_other":     c.Int("hrsp_other"),
		"hanafail":       c.Int("hanafail"),
		"req_rate":       c.Int("req_rate"),
		"req_rate_max":   c.Int("req_rate_max"),
		"req_tot":        c.Int("req_tot"),
		"cli_abrt":       c.Int("cli_abrt"),
		"srv_abrt":       c.Int("srv_abrt"),
		"comp_in":        c.Int("comp_in"),
		"comp_out":       c.Int("comp_out"),
		"comp_byp":       c.Int("comp_byp"),
		"comp_rsp":       c.Int("comp_rsp"),
		"lastsess":       c.Int("lastsess"),
		"last_chk":       c.Str("last_chk"),
		"last_agt":       c.Int("last_agt"),
		"qtime":          c.Int("qtime"),
		"ctime":          c.Int("ctime"),
		"rtime":          c.Int("rtime"),
		"ttime":          c.Int("ttime"),
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
func eventMapping(info []map[string]string) []common.MapStr {

	var events []common.MapStr

	source := map[string]interface{}{}

	for _, evt := range info {
		source = map[string]interface{}{}
		for key, val := range evt {
			source[key] = val
		}
		events = append(events, schema.Apply(source))
	}

	return events
}
