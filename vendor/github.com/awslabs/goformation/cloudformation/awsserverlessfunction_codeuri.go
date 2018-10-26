package cloudformation

import (
	"encoding/json"

	"reflect"

	"github.com/mitchellh/mapstructure"
)

// AWSServerlessFunction_CodeUri is a helper struct that can hold either a String or S3Location value
type AWSServerlessFunction_CodeUri struct {
	String *string

	S3Location *AWSServerlessFunction_S3Location
}

func (r AWSServerlessFunction_CodeUri) value() interface{} {

	if r.String != nil {
		return r.String
	}

	if r.S3Location != nil && !reflect.DeepEqual(r.S3Location, &AWSServerlessFunction_S3Location{}) {
		return r.S3Location
	}

	if r.S3Location != nil {
		return r.S3Location
	}

	return nil

}

func (r AWSServerlessFunction_CodeUri) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.value())
}

// Hook into the marshaller
func (r *AWSServerlessFunction_CodeUri) UnmarshalJSON(b []byte) error {

	// Unmarshal into interface{} to check it's type
	var typecheck interface{}
	if err := json.Unmarshal(b, &typecheck); err != nil {
		return err
	}

	switch val := typecheck.(type) {

	case string:
		r.String = &val

	case map[string]interface{}:

		mapstructure.Decode(val, &r.S3Location)

	case []interface{}:

	}

	return nil
}
