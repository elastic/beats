package wafv2

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// RegexPatternSet_RegexPatternSetSummary AWS CloudFormation Resource (AWS::WAFv2::RegexPatternSet.RegexPatternSetSummary)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-regexpatternset-regexpatternsetsummary.html
type RegexPatternSet_RegexPatternSetSummary struct {

	// ARN AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-regexpatternset-regexpatternsetsummary.html#cfn-wafv2-regexpatternset-regexpatternsetsummary-arn
	ARN string `json:"ARN,omitempty"`

	// Description AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-regexpatternset-regexpatternsetsummary.html#cfn-wafv2-regexpatternset-regexpatternsetsummary-description
	Description string `json:"Description,omitempty"`

	// Id AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-regexpatternset-regexpatternsetsummary.html#cfn-wafv2-regexpatternset-regexpatternsetsummary-id
	Id string `json:"Id,omitempty"`

	// LockToken AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-regexpatternset-regexpatternsetsummary.html#cfn-wafv2-regexpatternset-regexpatternsetsummary-locktoken
	LockToken string `json:"LockToken,omitempty"`

	// Name AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-regexpatternset-regexpatternsetsummary.html#cfn-wafv2-regexpatternset-regexpatternsetsummary-name
	Name string `json:"Name,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *RegexPatternSet_RegexPatternSetSummary) AWSCloudFormationType() string {
	return "AWS::WAFv2::RegexPatternSet.RegexPatternSetSummary"
}
