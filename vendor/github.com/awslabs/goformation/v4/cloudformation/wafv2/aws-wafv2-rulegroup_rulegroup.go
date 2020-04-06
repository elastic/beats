package wafv2

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// RuleGroup_RuleGroup AWS CloudFormation Resource (AWS::WAFv2::RuleGroup.RuleGroup)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-rulegroup.html
type RuleGroup_RuleGroup struct {

	// ARN AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-rulegroup.html#cfn-wafv2-rulegroup-rulegroup-arn
	ARN string `json:"ARN,omitempty"`

	// Capacity AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-rulegroup.html#cfn-wafv2-rulegroup-rulegroup-capacity
	Capacity int `json:"Capacity,omitempty"`

	// Description AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-rulegroup.html#cfn-wafv2-rulegroup-rulegroup-description
	Description string `json:"Description,omitempty"`

	// Id AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-rulegroup.html#cfn-wafv2-rulegroup-rulegroup-id
	Id string `json:"Id,omitempty"`

	// Name AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-rulegroup.html#cfn-wafv2-rulegroup-rulegroup-name
	Name string `json:"Name,omitempty"`

	// Rules AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-rulegroup.html#cfn-wafv2-rulegroup-rulegroup-rules
	Rules *RuleGroup_Rules `json:"Rules,omitempty"`

	// VisibilityConfig AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-rulegroup.html#cfn-wafv2-rulegroup-rulegroup-visibilityconfig
	VisibilityConfig *RuleGroup_VisibilityConfig `json:"VisibilityConfig,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *RuleGroup_RuleGroup) AWSCloudFormationType() string {
	return "AWS::WAFv2::RuleGroup.RuleGroup"
}
