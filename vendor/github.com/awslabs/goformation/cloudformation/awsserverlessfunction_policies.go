package cloudformation

import (
	"encoding/json"

	"reflect"

	"github.com/mitchellh/mapstructure"
)

// AWSServerlessFunction_Policies is a helper struct that can hold either a String, String, IAMPolicyDocument, or IAMPolicyDocument value
type AWSServerlessFunction_Policies struct {
	String *string

	StringArray *[]string

	IAMPolicyDocument *AWSServerlessFunction_IAMPolicyDocument

	IAMPolicyDocumentArray *[]AWSServerlessFunction_IAMPolicyDocument
}

func (r AWSServerlessFunction_Policies) value() interface{} {

	if r.String != nil {
		return r.String
	}

	if r.StringArray != nil {
		return r.StringArray
	}

	if r.IAMPolicyDocument != nil && !reflect.DeepEqual(r.IAMPolicyDocument, &AWSServerlessFunction_IAMPolicyDocument{}) {
		return r.IAMPolicyDocument
	}

	if r.IAMPolicyDocument != nil {
		return r.IAMPolicyDocument
	}

	if r.IAMPolicyDocumentArray != nil {
		return r.IAMPolicyDocumentArray
	}

	return nil

}

func (r AWSServerlessFunction_Policies) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.value())
}

// Hook into the marshaller
func (r *AWSServerlessFunction_Policies) UnmarshalJSON(b []byte) error {

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

		mapstructure.Decode(val, &r.IAMPolicyDocument)

	case []interface{}:

		mapstructure.Decode(val, &r.StringArray)

		mapstructure.Decode(val, &r.IAMPolicyDocumentArray)

	}

	return nil
}
