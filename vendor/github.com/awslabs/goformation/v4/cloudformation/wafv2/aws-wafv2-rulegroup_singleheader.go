package wafv2

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// RuleGroup_SingleHeader AWS CloudFormation Resource (AWS::WAFv2::RuleGroup.SingleHeader)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-singleheader.html
type RuleGroup_SingleHeader struct {

	// Name AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-singleheader.html#cfn-wafv2-rulegroup-singleheader-name
	Name string `json:"Name,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *RuleGroup_SingleHeader) AWSCloudFormationType() string {
	return "AWS::WAFv2::RuleGroup.SingleHeader"
}
