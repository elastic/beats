package jsontransform

import (
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// WriteJSONKeys writes the json keys to the given event based on the overwriteKeys option
func WriteJSONKeys(event *beat.Event, keys map[string]interface{}, overwriteKeys bool) {
	if !overwriteKeys {
		for k, v := range keys {
			if _, exists := event.Fields[k]; !exists && k != "@timestamp" && k != "@metadata" {
				event.Fields[k] = v
			}
		}
		return
	}

	for k, v := range keys {
		switch k {
		case "@timestamp":
			vstr, ok := v.(string)
			if !ok {
				logp.Err("JSON: Won't overwrite @timestamp because value is not string")
				event.Fields["error"] = createJSONError("@timestamp not overwritten (not string)")
				continue
			}

			// @timestamp must be of format RFC3339
			ts, err := time.Parse(time.RFC3339, vstr)
			if err != nil {
				logp.Err("JSON: Won't overwrite @timestamp because of parsing error: %v", err)
				event.Fields["error"] = createJSONError(fmt.Sprintf("@timestamp not overwritten (parse error on %s)", vstr))
				continue
			}
			event.Timestamp = ts

		case "@metadata":
			switch m := v.(type) {
			case map[string]string:
				for meta, value := range m {
					event.Meta[meta] = value
				}

			case map[string]interface{}:
				event.Meta.Update(common.MapStr(m))

			default:
				event.Fields["error"] = createJSONError("failed to update @metadata")
			}

		case "type":
			vstr, ok := v.(string)
			if !ok {
				logp.Err("JSON: Won't overwrite type because value is not string")
				event.Fields["error"] = createJSONError("type not overwritten (not string)")
				continue
			}
			if len(vstr) == 0 || vstr[0] == '_' {
				logp.Err("JSON: Won't overwrite type because value is empty or starts with an underscore")
				event.Fields["error"] = createJSONError(fmt.Sprintf("type not overwritten (invalid value [%s])", vstr))
				continue
			}
			event.Fields[k] = vstr

		default:
			event.Fields[k] = v
		}
	}
}

func createJSONError(message string) common.MapStr {
	return common.MapStr{"message": message, "type": "json"}
}
