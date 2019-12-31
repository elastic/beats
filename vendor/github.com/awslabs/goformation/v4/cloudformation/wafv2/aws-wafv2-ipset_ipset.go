package wafv2

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// IPSet_IPSet AWS CloudFormation Resource (AWS::WAFv2::IPSet.IPSet)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-ipset-ipset.html
type IPSet_IPSet struct {

	// ARN AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-ipset-ipset.html#cfn-wafv2-ipset-ipset-arn
	ARN string `json:"ARN,omitempty"`

	// Addresses AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-ipset-ipset.html#cfn-wafv2-ipset-ipset-addresses
	Addresses *IPSet_IPAddresses `json:"Addresses,omitempty"`

	// Description AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-ipset-ipset.html#cfn-wafv2-ipset-ipset-description
	Description string `json:"Description,omitempty"`

	// IPAddressVersion AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-ipset-ipset.html#cfn-wafv2-ipset-ipset-ipaddressversion
	IPAddressVersion string `json:"IPAddressVersion,omitempty"`

	// Id AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-ipset-ipset.html#cfn-wafv2-ipset-ipset-id
	Id string `json:"Id,omitempty"`

	// Name AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-ipset-ipset.html#cfn-wafv2-ipset-ipset-name
	Name string `json:"Name,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *IPSet_IPSet) AWSCloudFormationType() string {
	return "AWS::WAFv2::IPSet.IPSet"
}
