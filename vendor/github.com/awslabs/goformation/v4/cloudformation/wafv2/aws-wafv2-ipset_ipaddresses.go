package wafv2

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// IPSet_IPAddresses AWS CloudFormation Resource (AWS::WAFv2::IPSet.IPAddresses)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-ipset-ipaddresses.html
type IPSet_IPAddresses struct {

	// IPAddresses AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-ipset-ipaddresses.html#cfn-wafv2-ipset-ipaddresses-ipaddresses
	IPAddresses []string `json:"IPAddresses,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *IPSet_IPAddresses) AWSCloudFormationType() string {
	return "AWS::WAFv2::IPSet.IPAddresses"
}
