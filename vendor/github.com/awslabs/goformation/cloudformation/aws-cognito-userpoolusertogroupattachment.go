package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSCognitoUserPoolUserToGroupAttachment AWS CloudFormation Resource (AWS::Cognito::UserPoolUserToGroupAttachment)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-cognito-userpoolusertogroupattachment.html
type AWSCognitoUserPoolUserToGroupAttachment struct {

	// GroupName AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-cognito-userpoolusertogroupattachment.html#cfn-cognito-userpoolusertogroupattachment-groupname
	GroupName string `json:"GroupName,omitempty"`

	// UserPoolId AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-cognito-userpoolusertogroupattachment.html#cfn-cognito-userpoolusertogroupattachment-userpoolid
	UserPoolId string `json:"UserPoolId,omitempty"`

	// Username AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-cognito-userpoolusertogroupattachment.html#cfn-cognito-userpoolusertogroupattachment-username
	Username string `json:"Username,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSCognitoUserPoolUserToGroupAttachment) AWSCloudFormationType() string {
	return "AWS::Cognito::UserPoolUserToGroupAttachment"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSCognitoUserPoolUserToGroupAttachment) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSCognitoUserPoolUserToGroupAttachment) MarshalJSON() ([]byte, error) {
	type Properties AWSCognitoUserPoolUserToGroupAttachment
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
func (r *AWSCognitoUserPoolUserToGroupAttachment) UnmarshalJSON(b []byte) error {
	type Properties AWSCognitoUserPoolUserToGroupAttachment
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
		*r = AWSCognitoUserPoolUserToGroupAttachment(*res.Properties)
	}

	return nil
}

// GetAllAWSCognitoUserPoolUserToGroupAttachmentResources retrieves all AWSCognitoUserPoolUserToGroupAttachment items from an AWS CloudFormation template
func (t *Template) GetAllAWSCognitoUserPoolUserToGroupAttachmentResources() map[string]AWSCognitoUserPoolUserToGroupAttachment {
	results := map[string]AWSCognitoUserPoolUserToGroupAttachment{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSCognitoUserPoolUserToGroupAttachment:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::Cognito::UserPoolUserToGroupAttachment" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSCognitoUserPoolUserToGroupAttachment
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

// GetAWSCognitoUserPoolUserToGroupAttachmentWithName retrieves all AWSCognitoUserPoolUserToGroupAttachment items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSCognitoUserPoolUserToGroupAttachmentWithName(name string) (AWSCognitoUserPoolUserToGroupAttachment, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSCognitoUserPoolUserToGroupAttachment:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::Cognito::UserPoolUserToGroupAttachment" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSCognitoUserPoolUserToGroupAttachment
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSCognitoUserPoolUserToGroupAttachment{}, errors.New("resource not found")
}
