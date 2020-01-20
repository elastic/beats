package serverless

import (
	"encoding/json"
	"sort"

	"github.com/awslabs/goformation/v4/cloudformation/utils"
)

// Api_Cors is a helper struct that can hold either a String or CorsConfiguration value
type Api_Cors struct {
	String *string

	CorsConfiguration *Api_CorsConfiguration
}

func (r Api_Cors) value() interface{} {
	ret := []interface{}{}

	if r.String != nil {
		ret = append(ret, r.String)
	}

	if r.CorsConfiguration != nil {
		ret = append(ret, *r.CorsConfiguration)
	}

	sort.Sort(utils.ByJSONLength(ret)) // Heuristic to select best attribute
	if len(ret) > 0 {
		return ret[0]
	}

	return nil
}

func (r Api_Cors) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.value())
}

// Hook into the marshaller
func (r *Api_Cors) UnmarshalJSON(b []byte) error {

	// Unmarshal into interface{} to check it's type
	var typecheck interface{}
	if err := json.Unmarshal(b, &typecheck); err != nil {
		return err
	}

	switch val := typecheck.(type) {

	case string:
		r.String = &val

	case map[string]interface{}:
		val = val // This ensures val is used to stop an error

		json.Unmarshal(b, &r.CorsConfiguration)

	case []interface{}:

	}

	return nil
}
