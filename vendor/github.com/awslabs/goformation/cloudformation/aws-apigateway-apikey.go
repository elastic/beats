package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSApiGatewayApiKey AWS CloudFormation Resource (AWS::ApiGateway::ApiKey)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-apigateway-apikey.html
type AWSApiGatewayApiKey struct {

	// CustomerId AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-apigateway-apikey.html#cfn-apigateway-apikey-customerid
	CustomerId string `json:"CustomerId,omitempty"`

	// Description AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-apigateway-apikey.html#cfn-apigateway-apikey-description
	Description string `json:"Description,omitempty"`

	// Enabled AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-apigateway-apikey.html#cfn-apigateway-apikey-enabled
	Enabled bool `json:"Enabled,omitempty"`

	// GenerateDistinctId AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-apigateway-apikey.html#cfn-apigateway-apikey-generatedistinctid
	GenerateDistinctId bool `json:"GenerateDistinctId,omitempty"`

	// Name AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-apigateway-apikey.html#cfn-apigateway-apikey-name
	Name string `json:"Name,omitempty"`

	// StageKeys AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-apigateway-apikey.html#cfn-apigateway-apikey-stagekeys
	StageKeys []AWSApiGatewayApiKey_StageKey `json:"StageKeys,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSApiGatewayApiKey) AWSCloudFormationType() string {
	return "AWS::ApiGateway::ApiKey"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSApiGatewayApiKey) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSApiGatewayApiKey) MarshalJSON() ([]byte, error) {
	type Properties AWSApiGatewayApiKey
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
func (r *AWSApiGatewayApiKey) UnmarshalJSON(b []byte) error {
	type Properties AWSApiGatewayApiKey
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
		*r = AWSApiGatewayApiKey(*res.Properties)
	}

	return nil
}

// GetAllAWSApiGatewayApiKeyResources retrieves all AWSApiGatewayApiKey items from an AWS CloudFormation template
func (t *Template) GetAllAWSApiGatewayApiKeyResources() map[string]AWSApiGatewayApiKey {
	results := map[string]AWSApiGatewayApiKey{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSApiGatewayApiKey:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::ApiGateway::ApiKey" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSApiGatewayApiKey
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

// GetAWSApiGatewayApiKeyWithName retrieves all AWSApiGatewayApiKey items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSApiGatewayApiKeyWithName(name string) (AWSApiGatewayApiKey, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSApiGatewayApiKey:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::ApiGateway::ApiKey" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSApiGatewayApiKey
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSApiGatewayApiKey{}, errors.New("resource not found")
}
