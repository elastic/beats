package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSCloudFormationWaitCondition AWS CloudFormation Resource (AWS::CloudFormation::WaitCondition)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-waitcondition.html
type AWSCloudFormationWaitCondition struct {

	// Count AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-waitcondition.html#cfn-waitcondition-count
	Count int `json:"Count,omitempty"`

	// Handle AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-waitcondition.html#cfn-waitcondition-handle
	Handle string `json:"Handle,omitempty"`

	// Timeout AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-waitcondition.html#cfn-waitcondition-timeout
	Timeout string `json:"Timeout,omitempty"`

	// _creationPolicy represents a CloudFormation CreationPolicy
	_creationPolicy *CreationPolicy

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSCloudFormationWaitCondition) AWSCloudFormationType() string {
	return "AWS::CloudFormation::WaitCondition"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSCloudFormationWaitCondition) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// SetCreationPolicy applies an AWS CloudFormation CreationPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-creationpolicy.html
func (r *AWSCloudFormationWaitCondition) SetCreationPolicy(policy *CreationPolicy) {
	r._creationPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSCloudFormationWaitCondition) MarshalJSON() ([]byte, error) {
	type Properties AWSCloudFormationWaitCondition
	return json.Marshal(&struct {
		Type           string
		Properties     Properties
		DeletionPolicy DeletionPolicy `json:"DeletionPolicy,omitempty"`

		CreationPolicy *CreationPolicy `json:"CreationPolicy,omitempty"`
	}{
		Type:           r.AWSCloudFormationType(),
		Properties:     (Properties)(r),
		DeletionPolicy: r._deletionPolicy,

		CreationPolicy: r._creationPolicy,
	})
}

// UnmarshalJSON is a custom JSON unmarshalling hook that strips the outer
// AWS CloudFormation resource object, and just keeps the 'Properties' field.
func (r *AWSCloudFormationWaitCondition) UnmarshalJSON(b []byte) error {
	type Properties AWSCloudFormationWaitCondition
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
		*r = AWSCloudFormationWaitCondition(*res.Properties)
	}

	return nil
}

// GetAllAWSCloudFormationWaitConditionResources retrieves all AWSCloudFormationWaitCondition items from an AWS CloudFormation template
func (t *Template) GetAllAWSCloudFormationWaitConditionResources() map[string]AWSCloudFormationWaitCondition {
	results := map[string]AWSCloudFormationWaitCondition{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSCloudFormationWaitCondition:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::CloudFormation::WaitCondition" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSCloudFormationWaitCondition
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

// GetAWSCloudFormationWaitConditionWithName retrieves all AWSCloudFormationWaitCondition items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSCloudFormationWaitConditionWithName(name string) (AWSCloudFormationWaitCondition, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSCloudFormationWaitCondition:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::CloudFormation::WaitCondition" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSCloudFormationWaitCondition
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSCloudFormationWaitCondition{}, errors.New("resource not found")
}
