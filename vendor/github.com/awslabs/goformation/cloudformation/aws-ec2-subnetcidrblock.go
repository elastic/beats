package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSEC2SubnetCidrBlock AWS CloudFormation Resource (AWS::EC2::SubnetCidrBlock)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-ec2-subnetcidrblock.html
type AWSEC2SubnetCidrBlock struct {

	// Ipv6CidrBlock AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-ec2-subnetcidrblock.html#cfn-ec2-subnetcidrblock-ipv6cidrblock
	Ipv6CidrBlock string `json:"Ipv6CidrBlock,omitempty"`

	// SubnetId AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-ec2-subnetcidrblock.html#cfn-ec2-subnetcidrblock-subnetid
	SubnetId string `json:"SubnetId,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSEC2SubnetCidrBlock) AWSCloudFormationType() string {
	return "AWS::EC2::SubnetCidrBlock"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSEC2SubnetCidrBlock) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSEC2SubnetCidrBlock) MarshalJSON() ([]byte, error) {
	type Properties AWSEC2SubnetCidrBlock
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
func (r *AWSEC2SubnetCidrBlock) UnmarshalJSON(b []byte) error {
	type Properties AWSEC2SubnetCidrBlock
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
		*r = AWSEC2SubnetCidrBlock(*res.Properties)
	}

	return nil
}

// GetAllAWSEC2SubnetCidrBlockResources retrieves all AWSEC2SubnetCidrBlock items from an AWS CloudFormation template
func (t *Template) GetAllAWSEC2SubnetCidrBlockResources() map[string]AWSEC2SubnetCidrBlock {
	results := map[string]AWSEC2SubnetCidrBlock{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSEC2SubnetCidrBlock:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::EC2::SubnetCidrBlock" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSEC2SubnetCidrBlock
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

// GetAWSEC2SubnetCidrBlockWithName retrieves all AWSEC2SubnetCidrBlock items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSEC2SubnetCidrBlockWithName(name string) (AWSEC2SubnetCidrBlock, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSEC2SubnetCidrBlock:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::EC2::SubnetCidrBlock" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSEC2SubnetCidrBlock
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSEC2SubnetCidrBlock{}, errors.New("resource not found")
}
