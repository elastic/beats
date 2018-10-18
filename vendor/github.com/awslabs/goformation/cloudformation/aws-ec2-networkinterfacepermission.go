package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSEC2NetworkInterfacePermission AWS CloudFormation Resource (AWS::EC2::NetworkInterfacePermission)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-ec2-networkinterfacepermission.html
type AWSEC2NetworkInterfacePermission struct {

	// AwsAccountId AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-ec2-networkinterfacepermission.html#cfn-ec2-networkinterfacepermission-awsaccountid
	AwsAccountId string `json:"AwsAccountId,omitempty"`

	// NetworkInterfaceId AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-ec2-networkinterfacepermission.html#cfn-ec2-networkinterfacepermission-networkinterfaceid
	NetworkInterfaceId string `json:"NetworkInterfaceId,omitempty"`

	// Permission AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-ec2-networkinterfacepermission.html#cfn-ec2-networkinterfacepermission-permission
	Permission string `json:"Permission,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSEC2NetworkInterfacePermission) AWSCloudFormationType() string {
	return "AWS::EC2::NetworkInterfacePermission"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSEC2NetworkInterfacePermission) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSEC2NetworkInterfacePermission) MarshalJSON() ([]byte, error) {
	type Properties AWSEC2NetworkInterfacePermission
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
func (r *AWSEC2NetworkInterfacePermission) UnmarshalJSON(b []byte) error {
	type Properties AWSEC2NetworkInterfacePermission
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
		*r = AWSEC2NetworkInterfacePermission(*res.Properties)
	}

	return nil
}

// GetAllAWSEC2NetworkInterfacePermissionResources retrieves all AWSEC2NetworkInterfacePermission items from an AWS CloudFormation template
func (t *Template) GetAllAWSEC2NetworkInterfacePermissionResources() map[string]AWSEC2NetworkInterfacePermission {
	results := map[string]AWSEC2NetworkInterfacePermission{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSEC2NetworkInterfacePermission:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::EC2::NetworkInterfacePermission" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSEC2NetworkInterfacePermission
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

// GetAWSEC2NetworkInterfacePermissionWithName retrieves all AWSEC2NetworkInterfacePermission items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSEC2NetworkInterfacePermissionWithName(name string) (AWSEC2NetworkInterfacePermission, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSEC2NetworkInterfacePermission:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::EC2::NetworkInterfacePermission" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSEC2NetworkInterfacePermission
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSEC2NetworkInterfacePermission{}, errors.New("resource not found")
}
