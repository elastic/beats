package wafv2

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// RegexPatternSet_RegexPatternSets AWS CloudFormation Resource (AWS::WAFv2::RegexPatternSet.RegexPatternSets)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-regexpatternset-regexpatternsets.html
type RegexPatternSet_RegexPatternSets struct {

	// RegexPatternSets AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-regexpatternset-regexpatternsets.html#cfn-wafv2-regexpatternset-regexpatternsets-regexpatternsets
	RegexPatternSets []RegexPatternSet_RegexPatternSetSummary `json:"RegexPatternSets,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *RegexPatternSet_RegexPatternSets) AWSCloudFormationType() string {
	return "AWS::WAFv2::RegexPatternSet.RegexPatternSets"
}
