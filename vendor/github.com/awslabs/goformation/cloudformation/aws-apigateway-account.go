package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSApiGatewayAccount AWS CloudFormation Resource (AWS::ApiGateway::Account)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-apigateway-account.html
type AWSApiGatewayAccount struct {

	// CloudWatchRoleArn AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-apigateway-account.html#cfn-apigateway-account-cloudwatchrolearn
	CloudWatchRoleArn string `json:"CloudWatchRoleArn,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSApiGatewayAccount) AWSCloudFormationType() string {
	return "AWS::ApiGateway::Account"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSApiGatewayAccount) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSApiGatewayAccount) MarshalJSON() ([]byte, error) {
	type Properties AWSApiGatewayAccount
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
func (r *AWSApiGatewayAccount) UnmarshalJSON(b []byte) error {
	type Properties AWSApiGatewayAccount
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
		*r = AWSApiGatewayAccount(*res.Properties)
	}

	return nil
}

// GetAllAWSApiGatewayAccountResources retrieves all AWSApiGatewayAccount items from an AWS CloudFormation template
func (t *Template) GetAllAWSApiGatewayAccountResources() map[string]AWSApiGatewayAccount {
	results := map[string]AWSApiGatewayAccount{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSApiGatewayAccount:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::ApiGateway::Account" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSApiGatewayAccount
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

// GetAWSApiGatewayAccountWithName retrieves all AWSApiGatewayAccount items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSApiGatewayAccountWithName(name string) (AWSApiGatewayAccount, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSApiGatewayAccount:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::ApiGateway::Account" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSApiGatewayAccount
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSApiGatewayAccount{}, errors.New("resource not found")
}
