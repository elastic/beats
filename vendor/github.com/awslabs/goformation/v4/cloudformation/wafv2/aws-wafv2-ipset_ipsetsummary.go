package wafv2

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// IPSet_IPSetSummary AWS CloudFormation Resource (AWS::WAFv2::IPSet.IPSetSummary)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-ipset-ipsetsummary.html
type IPSet_IPSetSummary struct {

	// ARN AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-ipset-ipsetsummary.html#cfn-wafv2-ipset-ipsetsummary-arn
	ARN string `json:"ARN,omitempty"`

	// Description AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-ipset-ipsetsummary.html#cfn-wafv2-ipset-ipsetsummary-description
	Description string `json:"Description,omitempty"`

	// Id AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-ipset-ipsetsummary.html#cfn-wafv2-ipset-ipsetsummary-id
	Id string `json:"Id,omitempty"`

	// LockToken AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-ipset-ipsetsummary.html#cfn-wafv2-ipset-ipsetsummary-locktoken
	LockToken string `json:"LockToken,omitempty"`

	// Name AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-ipset-ipsetsummary.html#cfn-wafv2-ipset-ipsetsummary-name
	Name string `json:"Name,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *IPSet_IPSetSummary) AWSCloudFormationType() string {
	return "AWS::WAFv2::IPSet.IPSetSummary"
}
