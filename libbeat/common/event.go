package common

import (
	"encoding/json"
	"time"

	"github.com/elastic/beats/libbeat/logp"
)

func ConvertToGenericEvent(v MapStr) MapStr {
	for key, value := range v {

		switch value.(type) {
		case int, int8, int16, int32, int64:
		case uint8, uint16, uint32, uint64:
		case float32, float64:
		case complex64, complex128:
		case bool:
		case uintptr:
		case string, *string:
		case Time, *Time:
		case time.Location, *time.Location:
		case MapStr:
			v[key] = ConvertToGenericEvent(value.(MapStr))
		case *MapStr:
			v[key] = ConvertToGenericEvent(*value.(*MapStr))
		default:

			// decode and encode JSON
			marshaled, err := json.Marshal(value)
			if err != nil {
				logp.Err("marshal err: %v", err)
				return nil
			}
			var v1 MapStr
			err = json.Unmarshal(marshaled, &v1)
			if err != nil {
				logp.Err("unmarshal err: %v, type %T", err, value)
				return nil
			}
			v[key] = v1
		}
	}
	return v
}
