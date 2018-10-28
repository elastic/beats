package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSWAFXssMatchSet AWS CloudFormation Resource (AWS::WAF::XssMatchSet)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-waf-xssmatchset.html
type AWSWAFXssMatchSet struct {

	// Name AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-waf-xssmatchset.html#cfn-waf-xssmatchset-name
	Name string `json:"Name,omitempty"`

	// XssMatchTuples AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-waf-xssmatchset.html#cfn-waf-xssmatchset-xssmatchtuples
	XssMatchTuples []AWSWAFXssMatchSet_XssMatchTuple `json:"XssMatchTuples,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSWAFXssMatchSet) AWSCloudFormationType() string {
	return "AWS::WAF::XssMatchSet"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSWAFXssMatchSet) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSWAFXssMatchSet) MarshalJSON() ([]byte, error) {
	type Properties AWSWAFXssMatchSet
	return json.Marshal(&struct {
		Type           string
		Properties     Properties
		DeletionPolicy DeletionPolicy `json:"DeletionPolicy,omitempty"`
	}{
		Type:           r.AWSCloudFormationType(),
		Properties:     (Properties)(r),
		DeletionPolicy: r._deletionPolicy,
	})
}

// UnmarshalJSON is a custom JSON unmarshalling hook that strips the outer
// AWS CloudFormation resource object, and just keeps the 'Properties' field.
func (r *AWSWAFXssMatchSet) UnmarshalJSON(b []byte) error {
	type Properties AWSWAFXssMatchSet
	res := &struct {
		Type       string
		Properties *Properties
	}{}
	if err := json.Unmarshal(b, &res); err != nil {
		fmt.Printf("ERROR: %s\n", err)
		return err
	}

	// If the resource has no Properties set, it could be nil
	if res.Properties != nil {
		*r = AWSWAFXssMatchSet(*res.Properties)
	}

	return nil
}

// GetAllAWSWAFXssMatchSetResources retrieves all AWSWAFXssMatchSet items from an AWS CloudFormation template
func (t *Template) GetAllAWSWAFXssMatchSetResources() map[string]AWSWAFXssMatchSet {
	results := map[string]AWSWAFXssMatchSet{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSWAFXssMatchSet:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::WAF::XssMatchSet" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSWAFXssMatchSet
						if err := json.Unmarshal(b, &result); err == nil {
							results[name] = result
						}
					}
				}
			}
		}
	}
	return results
}

// GetAWSWAFXssMatchSetWithName retrieves all AWSWAFXssMatchSet items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSWAFXssMatchSetWithName(name string) (AWSWAFXssMatchSet, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSWAFXssMatchSet:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::WAF::XssMatchSet" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSWAFXssMatchSet
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSWAFXssMatchSet{}, errors.New("resource not found")
}
