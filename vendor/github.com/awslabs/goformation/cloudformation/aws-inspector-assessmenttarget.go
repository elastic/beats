package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSInspectorAssessmentTarget AWS CloudFormation Resource (AWS::Inspector::AssessmentTarget)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-inspector-assessmenttarget.html
type AWSInspectorAssessmentTarget struct {

	// AssessmentTargetName AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-inspector-assessmenttarget.html#cfn-inspector-assessmenttarget-assessmenttargetname
	AssessmentTargetName string `json:"AssessmentTargetName,omitempty"`

	// ResourceGroupArn AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-inspector-assessmenttarget.html#cfn-inspector-assessmenttarget-resourcegrouparn
	ResourceGroupArn string `json:"ResourceGroupArn,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSInspectorAssessmentTarget) AWSCloudFormationType() string {
	return "AWS::Inspector::AssessmentTarget"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSInspectorAssessmentTarget) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSInspectorAssessmentTarget) MarshalJSON() ([]byte, error) {
	type Properties AWSInspectorAssessmentTarget
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
func (r *AWSInspectorAssessmentTarget) UnmarshalJSON(b []byte) error {
	type Properties AWSInspectorAssessmentTarget
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
		*r = AWSInspectorAssessmentTarget(*res.Properties)
	}

	return nil
}

// GetAllAWSInspectorAssessmentTargetResources retrieves all AWSInspectorAssessmentTarget items from an AWS CloudFormation template
func (t *Template) GetAllAWSInspectorAssessmentTargetResources() map[string]AWSInspectorAssessmentTarget {
	results := map[string]AWSInspectorAssessmentTarget{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSInspectorAssessmentTarget:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::Inspector::AssessmentTarget" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSInspectorAssessmentTarget
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

// GetAWSInspectorAssessmentTargetWithName retrieves all AWSInspectorAssessmentTarget items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSInspectorAssessmentTargetWithName(name string) (AWSInspectorAssessmentTarget, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSInspectorAssessmentTarget:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::Inspector::AssessmentTarget" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSInspectorAssessmentTarget
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSInspectorAssessmentTarget{}, errors.New("resource not found")
}
