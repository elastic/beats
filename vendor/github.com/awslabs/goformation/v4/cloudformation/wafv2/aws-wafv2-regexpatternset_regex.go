package wafv2

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// RegexPatternSet_Regex AWS CloudFormation Resource (AWS::WAFv2::RegexPatternSet.Regex)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-regexpatternset-regex.html
type RegexPatternSet_Regex struct {

	// RegexString AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-regexpatternset-regex.html#cfn-wafv2-regexpatternset-regex-regexstring
	RegexString string `json:"RegexString,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *RegexPatternSet_Regex) AWSCloudFormationType() string {
	return "AWS::WAFv2::RegexPatternSet.Regex"
}
