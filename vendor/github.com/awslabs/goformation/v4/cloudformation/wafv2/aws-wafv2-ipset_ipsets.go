package wafv2

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// IPSet_IPSets AWS CloudFormation Resource (AWS::WAFv2::IPSet.IPSets)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-ipset-ipsets.html
type IPSet_IPSets struct {

	// IPSets AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-ipset-ipsets.html#cfn-wafv2-ipset-ipsets-ipsets
	IPSets []IPSet_IPSetSummary `json:"IPSets,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *IPSet_IPSets) AWSCloudFormationType() string {
	return "AWS::WAFv2::IPSet.IPSets"
}
