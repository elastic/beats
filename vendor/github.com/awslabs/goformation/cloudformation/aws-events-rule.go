package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSEventsRule AWS CloudFormation Resource (AWS::Events::Rule)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-events-rule.html
type AWSEventsRule struct {

	// Description AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-events-rule.html#cfn-events-rule-description
	Description string `json:"Description,omitempty"`

	// EventPattern AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-events-rule.html#cfn-events-rule-eventpattern
	EventPattern interface{} `json:"EventPattern,omitempty"`

	// Name AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-events-rule.html#cfn-events-rule-name
	Name string `json:"Name,omitempty"`

	// RoleArn AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-events-rule.html#cfn-events-rule-rolearn
	RoleArn string `json:"RoleArn,omitempty"`

	// ScheduleExpression AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-events-rule.html#cfn-events-rule-scheduleexpression
	ScheduleExpression string `json:"ScheduleExpression,omitempty"`

	// State AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-events-rule.html#cfn-events-rule-state
	State string `json:"State,omitempty"`

	// Targets AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-events-rule.html#cfn-events-rule-targets
	Targets []AWSEventsRule_Target `json:"Targets,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSEventsRule) AWSCloudFormationType() string {
	return "AWS::Events::Rule"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSEventsRule) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSEventsRule) MarshalJSON() ([]byte, error) {
	type Properties AWSEventsRule
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
func (r *AWSEventsRule) UnmarshalJSON(b []byte) error {
	type Properties AWSEventsRule
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
		*r = AWSEventsRule(*res.Properties)
	}

	return nil
}

// GetAllAWSEventsRuleResources retrieves all AWSEventsRule items from an AWS CloudFormation template
func (t *Template) GetAllAWSEventsRuleResources() map[string]AWSEventsRule {
	results := map[string]AWSEventsRule{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSEventsRule:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::Events::Rule" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSEventsRule
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

// GetAWSEventsRuleWithName retrieves all AWSEventsRule items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSEventsRuleWithName(name string) (AWSEventsRule, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSEventsRule:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::Events::Rule" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSEventsRule
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSEventsRule{}, errors.New("resource not found")
}
