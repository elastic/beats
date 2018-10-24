package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSEC2EgressOnlyInternetGateway AWS CloudFormation Resource (AWS::EC2::EgressOnlyInternetGateway)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-ec2-egressonlyinternetgateway.html
type AWSEC2EgressOnlyInternetGateway struct {

	// VpcId AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-ec2-egressonlyinternetgateway.html#cfn-ec2-egressonlyinternetgateway-vpcid
	VpcId string `json:"VpcId,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSEC2EgressOnlyInternetGateway) AWSCloudFormationType() string {
	return "AWS::EC2::EgressOnlyInternetGateway"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSEC2EgressOnlyInternetGateway) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSEC2EgressOnlyInternetGateway) MarshalJSON() ([]byte, error) {
	type Properties AWSEC2EgressOnlyInternetGateway
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
func (r *AWSEC2EgressOnlyInternetGateway) UnmarshalJSON(b []byte) error {
	type Properties AWSEC2EgressOnlyInternetGateway
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
		*r = AWSEC2EgressOnlyInternetGateway(*res.Properties)
	}

	return nil
}

// GetAllAWSEC2EgressOnlyInternetGatewayResources retrieves all AWSEC2EgressOnlyInternetGateway items from an AWS CloudFormation template
func (t *Template) GetAllAWSEC2EgressOnlyInternetGatewayResources() map[string]AWSEC2EgressOnlyInternetGateway {
	results := map[string]AWSEC2EgressOnlyInternetGateway{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSEC2EgressOnlyInternetGateway:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::EC2::EgressOnlyInternetGateway" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSEC2EgressOnlyInternetGateway
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

// GetAWSEC2EgressOnlyInternetGatewayWithName retrieves all AWSEC2EgressOnlyInternetGateway items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSEC2EgressOnlyInternetGatewayWithName(name string) (AWSEC2EgressOnlyInternetGateway, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSEC2EgressOnlyInternetGateway:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::EC2::EgressOnlyInternetGateway" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSEC2EgressOnlyInternetGateway
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSEC2EgressOnlyInternetGateway{}, errors.New("resource not found")
}
