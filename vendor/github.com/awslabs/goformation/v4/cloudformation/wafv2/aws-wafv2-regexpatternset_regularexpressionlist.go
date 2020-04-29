package wafv2

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// RegexPatternSet_RegularExpressionList AWS CloudFormation Resource (AWS::WAFv2::RegexPatternSet.RegularExpressionList)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-regexpatternset-regularexpressionlist.html
type RegexPatternSet_RegularExpressionList struct {

	// RegularExpressionList AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-regexpatternset-regularexpressionlist.html#cfn-wafv2-regexpatternset-regularexpressionlist-regularexpressionlist
	RegularExpressionList []RegexPatternSet_Regex `json:"RegularExpressionList,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *RegexPatternSet_RegularExpressionList) AWSCloudFormationType() string {
	return "AWS::WAFv2::RegexPatternSet.RegularExpressionList"
}
