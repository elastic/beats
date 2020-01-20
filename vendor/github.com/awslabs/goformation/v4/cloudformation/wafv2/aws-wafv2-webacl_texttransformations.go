package wafv2

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// WebACL_TextTransformations AWS CloudFormation Resource (AWS::WAFv2::WebACL.TextTransformations)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-webacl-texttransformations.html
type WebACL_TextTransformations struct {

	// TextTransformations AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-webacl-texttransformations.html#cfn-wafv2-webacl-texttransformations-texttransformations
	TextTransformations []WebACL_TextTransformation `json:"TextTransformations,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *WebACL_TextTransformations) AWSCloudFormationType() string {
	return "AWS::WAFv2::WebACL.TextTransformations"
}
