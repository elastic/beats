package wafv2

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// RuleGroup_RuleGroupSummary AWS CloudFormation Resource (AWS::WAFv2::RuleGroup.RuleGroupSummary)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-rulegroupsummary.html
type RuleGroup_RuleGroupSummary struct {

	// ARN AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-rulegroupsummary.html#cfn-wafv2-rulegroup-rulegroupsummary-arn
	ARN string `json:"ARN,omitempty"`

	// Description AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-rulegroupsummary.html#cfn-wafv2-rulegroup-rulegroupsummary-description
	Description string `json:"Description,omitempty"`

	// Id AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-rulegroupsummary.html#cfn-wafv2-rulegroup-rulegroupsummary-id
	Id string `json:"Id,omitempty"`

	// LockToken AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-rulegroupsummary.html#cfn-wafv2-rulegroup-rulegroupsummary-locktoken
	LockToken string `json:"LockToken,omitempty"`

	// Name AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-rulegroupsummary.html#cfn-wafv2-rulegroup-rulegroupsummary-name
	Name string `json:"Name,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *RuleGroup_RuleGroupSummary) AWSCloudFormationType() string {
	return "AWS::WAFv2::RuleGroup.RuleGroupSummary"
}
