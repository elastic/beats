package wafv2

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// RegexPatternSet_RegexPatternSet AWS CloudFormation Resource (AWS::WAFv2::RegexPatternSet.RegexPatternSet)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-regexpatternset-regexpatternset.html
type RegexPatternSet_RegexPatternSet struct {

	// ARN AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-regexpatternset-regexpatternset.html#cfn-wafv2-regexpatternset-regexpatternset-arn
	ARN string `json:"ARN,omitempty"`

	// Description AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-regexpatternset-regexpatternset.html#cfn-wafv2-regexpatternset-regexpatternset-description
	Description string `json:"Description,omitempty"`

	// Id AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-regexpatternset-regexpatternset.html#cfn-wafv2-regexpatternset-regexpatternset-id
	Id string `json:"Id,omitempty"`

	// Name AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-regexpatternset-regexpatternset.html#cfn-wafv2-regexpatternset-regexpatternset-name
	Name string `json:"Name,omitempty"`

	// RegularExpressionList AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-regexpatternset-regexpatternset.html#cfn-wafv2-regexpatternset-regexpatternset-regularexpressionlist
	RegularExpressionList *RegexPatternSet_RegularExpressionList `json:"RegularExpressionList,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *RegexPatternSet_RegexPatternSet) AWSCloudFormationType() string {
	return "AWS::WAFv2::RegexPatternSet.RegexPatternSet"
}
