package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSSESReceiptRuleSet AWS CloudFormation Resource (AWS::SES::ReceiptRuleSet)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-ses-receiptruleset.html
type AWSSESReceiptRuleSet struct {

	// RuleSetName AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-ses-receiptruleset.html#cfn-ses-receiptruleset-rulesetname
	RuleSetName string `json:"RuleSetName,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSSESReceiptRuleSet) AWSCloudFormationType() string {
	return "AWS::SES::ReceiptRuleSet"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSSESReceiptRuleSet) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSSESReceiptRuleSet) MarshalJSON() ([]byte, error) {
	type Properties AWSSESReceiptRuleSet
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
func (r *AWSSESReceiptRuleSet) UnmarshalJSON(b []byte) error {
	type Properties AWSSESReceiptRuleSet
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
		*r = AWSSESReceiptRuleSet(*res.Properties)
	}

	return nil
}

// GetAllAWSSESReceiptRuleSetResources retrieves all AWSSESReceiptRuleSet items from an AWS CloudFormation template
func (t *Template) GetAllAWSSESReceiptRuleSetResources() map[string]AWSSESReceiptRuleSet {
	results := map[string]AWSSESReceiptRuleSet{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSSESReceiptRuleSet:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::SES::ReceiptRuleSet" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSSESReceiptRuleSet
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

// GetAWSSESReceiptRuleSetWithName retrieves all AWSSESReceiptRuleSet items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSSESReceiptRuleSetWithName(name string) (AWSSESReceiptRuleSet, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSSESReceiptRuleSet:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::SES::ReceiptRuleSet" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSSESReceiptRuleSet
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSSESReceiptRuleSet{}, errors.New("resource not found")
}
