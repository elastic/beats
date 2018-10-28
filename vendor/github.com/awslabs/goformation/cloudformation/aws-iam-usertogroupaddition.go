package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSIAMUserToGroupAddition AWS CloudFormation Resource (AWS::IAM::UserToGroupAddition)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iam-addusertogroup.html
type AWSIAMUserToGroupAddition struct {

	// GroupName AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iam-addusertogroup.html#cfn-iam-addusertogroup-groupname
	GroupName string `json:"GroupName,omitempty"`

	// Users AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iam-addusertogroup.html#cfn-iam-addusertogroup-users
	Users []string `json:"Users,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSIAMUserToGroupAddition) AWSCloudFormationType() string {
	return "AWS::IAM::UserToGroupAddition"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSIAMUserToGroupAddition) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSIAMUserToGroupAddition) MarshalJSON() ([]byte, error) {
	type Properties AWSIAMUserToGroupAddition
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
func (r *AWSIAMUserToGroupAddition) UnmarshalJSON(b []byte) error {
	type Properties AWSIAMUserToGroupAddition
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
		*r = AWSIAMUserToGroupAddition(*res.Properties)
	}

	return nil
}

// GetAllAWSIAMUserToGroupAdditionResources retrieves all AWSIAMUserToGroupAddition items from an AWS CloudFormation template
func (t *Template) GetAllAWSIAMUserToGroupAdditionResources() map[string]AWSIAMUserToGroupAddition {
	results := map[string]AWSIAMUserToGroupAddition{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSIAMUserToGroupAddition:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::IAM::UserToGroupAddition" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSIAMUserToGroupAddition
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

// GetAWSIAMUserToGroupAdditionWithName retrieves all AWSIAMUserToGroupAddition items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSIAMUserToGroupAdditionWithName(name string) (AWSIAMUserToGroupAddition, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSIAMUserToGroupAddition:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::IAM::UserToGroupAddition" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSIAMUserToGroupAddition
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSIAMUserToGroupAddition{}, errors.New("resource not found")
}
