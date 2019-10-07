package cloudformation

import (
	"encoding/json"

	"github.com/awslabs/goformation/intrinsics"
	"github.com/sanathkr/yaml"
)

// Template represents an AWS CloudFormation template
// see: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/template-anatomy.html
type Template struct {
	AWSTemplateFormatVersion string                 `json:"AWSTemplateFormatVersion,omitempty"`
	Transform                *Transform             `json:"Transform,omitempty"`
	Description              string                 `json:"Description,omitempty"`
	Metadata                 map[string]interface{} `json:"Metadata,omitempty"`
	Parameters               map[string]interface{} `json:"Parameters,omitempty"`
	Mappings                 map[string]interface{} `json:"Mappings,omitempty"`
	Conditions               map[string]interface{} `json:"Conditions,omitempty"`
	Resources                map[string]interface{} `json:"Resources,omitempty"`
	Outputs                  map[string]interface{} `json:"Outputs,omitempty"`
}

type Transform struct {
	String *string

	StringArray *[]string
}

func (t Transform) value() interface{} {
	if t.String != nil {
		return t.String
	}

	if t.StringArray != nil {
		return t.StringArray
	}

	return nil
}

func (t *Transform) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.value())
}

func (t *Transform) UnmarshalJSON(b []byte) error {
	var typecheck interface{}
	if err := json.Unmarshal(b, &typecheck); err != nil {
		return err
	}

	switch val := typecheck.(type) {

	case string:
		t.String = &val

	case []string:
		t.StringArray = &val
	}

	return nil
}

// NewTemplate creates a new AWS CloudFormation template struct
func NewTemplate() *Template {
	return &Template{
		AWSTemplateFormatVersion: "2010-09-09",
		Description:              "",
		Metadata:                 map[string]interface{}{},
		Parameters:               map[string]interface{}{},
		Mappings:                 map[string]interface{}{},
		Conditions:               map[string]interface{}{},
		Resources:                map[string]interface{}{},
		Outputs:                  map[string]interface{}{},
	}
}

// JSON converts an AWS CloudFormation template object to JSON
func (t *Template) JSON() ([]byte, error) {

	j, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return nil, err
	}

	return intrinsics.ProcessJSON(j, nil)

}

// YAML converts an AWS CloudFormation template object to YAML
func (t *Template) YAML() ([]byte, error) {

	j, err := t.JSON()
	if err != nil {
		return nil, err
	}

	return yaml.JSONToYAML(j)

}
