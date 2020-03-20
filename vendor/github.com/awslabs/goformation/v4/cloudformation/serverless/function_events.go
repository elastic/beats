package serverless

import (
	"encoding/json"
	"sort"

	"github.com/awslabs/goformation/v4/cloudformation/utils"
)

// Function_Events is a helper struct that can hold either a String or String value
type Function_Events struct {
	String *string

	StringArray *[]string
}

func (r Function_Events) value() interface{} {
	ret := []interface{}{}

	if r.String != nil {
		ret = append(ret, r.String)
	}

	if r.StringArray != nil {
		ret = append(ret, r.StringArray)
	}

	sort.Sort(utils.ByJSONLength(ret)) // Heuristic to select best attribute
	if len(ret) > 0 {
		return ret[0]
	}

	return nil
}

func (r Function_Events) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.value())
}

// Hook into the marshaller
func (r *Function_Events) UnmarshalJSON(b []byte) error {

	// Unmarshal into interface{} to check it's type
	var typecheck interface{}
	if err := json.Unmarshal(b, &typecheck); err != nil {
		return err
	}

	switch val := typecheck.(type) {

	case string:
		r.String = &val

	case []string:
		r.StringArray = &val

	case map[string]interface{}:
		val = val // This ensures val is used to stop an error

	case []interface{}:

		json.Unmarshal(b, &r.StringArray)

	}

	return nil
}
