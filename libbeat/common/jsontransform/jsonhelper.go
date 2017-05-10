package jsontransform

import (
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// WriteJSONKeys writes the json keys to the given event based on the overwriteKeys option
func WriteJSONKeys(event common.MapStr, keys map[string]interface{}, overwriteKeys bool) {
	for k, v := range keys {
		if overwriteKeys {
			if k == "@timestamp" {
				vstr, ok := v.(string)
				if !ok {
					logp.Err("JSON: Won't overwrite @timestamp because value is not string")
					event["error"] = createJSONError("@timestamp not overwritten (not string)")
					continue
				}

				// @timestamp must be of format RFC3339
				ts, err := time.Parse(time.RFC3339, vstr)
				if err != nil {
					logp.Err("JSON: Won't overwrite @timestamp because of parsing error: %v", err)
					event["error"] = createJSONError(fmt.Sprintf("@timestamp not overwritten (parse error on %s)", vstr))
					continue
				}
				event[k] = common.Time(ts)
			} else if k == "type" {
				vstr, ok := v.(string)
				if !ok {
					logp.Err("JSON: Won't overwrite type because value is not string")
					event["error"] = createJSONError("type not overwritten (not string)")
					continue
				}
				if len(vstr) == 0 || vstr[0] == '_' {
					logp.Err("JSON: Won't overwrite type because value is empty or starts with an underscore")
					event["error"] = createJSONError(fmt.Sprintf("type not overwritten (invalid value [%s])", vstr))
					continue
				}
				event[k] = vstr
			} else {
				event[k] = v
			}
		} else if _, exists := event[k]; !exists {
			event[k] = v
		}

	}
}

func createJSONError(message string) common.MapStr {
	return common.MapStr{"message": message, "type": "json"}
}
