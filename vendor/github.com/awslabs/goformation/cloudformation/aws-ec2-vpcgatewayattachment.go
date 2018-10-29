package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSEC2VPCGatewayAttachment AWS CloudFormation Resource (AWS::EC2::VPCGatewayAttachment)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-ec2-vpc-gateway-attachment.html
type AWSEC2VPCGatewayAttachment struct {

	// InternetGatewayId AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-ec2-vpc-gateway-attachment.html#cfn-ec2-vpcgatewayattachment-internetgatewayid
	InternetGatewayId string `json:"InternetGatewayId,omitempty"`

	// VpcId AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-ec2-vpc-gateway-attachment.html#cfn-ec2-vpcgatewayattachment-vpcid
	VpcId string `json:"VpcId,omitempty"`

	// VpnGatewayId AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-ec2-vpc-gateway-attachment.html#cfn-ec2-vpcgatewayattachment-vpngatewayid
	VpnGatewayId string `json:"VpnGatewayId,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSEC2VPCGatewayAttachment) AWSCloudFormationType() string {
	return "AWS::EC2::VPCGatewayAttachment"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSEC2VPCGatewayAttachment) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSEC2VPCGatewayAttachment) MarshalJSON() ([]byte, error) {
	type Properties AWSEC2VPCGatewayAttachment
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
func (r *AWSEC2VPCGatewayAttachment) UnmarshalJSON(b []byte) error {
	type Properties AWSEC2VPCGatewayAttachment
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
		*r = AWSEC2VPCGatewayAttachment(*res.Properties)
	}

	return nil
}

// GetAllAWSEC2VPCGatewayAttachmentResources retrieves all AWSEC2VPCGatewayAttachment items from an AWS CloudFormation template
func (t *Template) GetAllAWSEC2VPCGatewayAttachmentResources() map[string]AWSEC2VPCGatewayAttachment {
	results := map[string]AWSEC2VPCGatewayAttachment{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSEC2VPCGatewayAttachment:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::EC2::VPCGatewayAttachment" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSEC2VPCGatewayAttachment
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

// GetAWSEC2VPCGatewayAttachmentWithName retrieves all AWSEC2VPCGatewayAttachment items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSEC2VPCGatewayAttachmentWithName(name string) (AWSEC2VPCGatewayAttachment, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSEC2VPCGatewayAttachment:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::EC2::VPCGatewayAttachment" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSEC2VPCGatewayAttachment
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSEC2VPCGatewayAttachment{}, errors.New("resource not found")
}
