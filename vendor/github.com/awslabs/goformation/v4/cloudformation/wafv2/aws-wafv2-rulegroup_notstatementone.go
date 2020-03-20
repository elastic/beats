package wafv2

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// RuleGroup_NotStatementOne AWS CloudFormation Resource (AWS::WAFv2::RuleGroup.NotStatementOne)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-notstatementone.html
type RuleGroup_NotStatementOne struct {

	// Statement AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-notstatementone.html#cfn-wafv2-rulegroup-notstatementone-statement
	Statement *RuleGroup_StatementTwo `json:"Statement,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *RuleGroup_NotStatementOne) AWSCloudFormationType() string {
	return "AWS::WAFv2::RuleGroup.NotStatementOne"
}
