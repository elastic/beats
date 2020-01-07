package wafv2

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
	"github.com/awslabs/goformation/v4/cloudformation/tags"
)

// RegexPatternSet_TagList AWS CloudFormation Resource (AWS::WAFv2::RegexPatternSet.TagList)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-regexpatternset-taglist.html
type RegexPatternSet_TagList struct {

	// TagList AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-regexpatternset-taglist.html#cfn-wafv2-regexpatternset-taglist-taglist
	TagList []tags.Tag `json:"TagList,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *RegexPatternSet_TagList) AWSCloudFormationType() string {
	return "AWS::WAFv2::RegexPatternSet.TagList"
}
