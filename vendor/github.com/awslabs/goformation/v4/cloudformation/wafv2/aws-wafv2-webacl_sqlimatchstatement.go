package wafv2

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// WebACL_SqliMatchStatement AWS CloudFormation Resource (AWS::WAFv2::WebACL.SqliMatchStatement)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-webacl-sqlimatchstatement.html
type WebACL_SqliMatchStatement struct {

	// FieldToMatch AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-webacl-sqlimatchstatement.html#cfn-wafv2-webacl-sqlimatchstatement-fieldtomatch
	FieldToMatch *WebACL_FieldToMatch `json:"FieldToMatch,omitempty"`

	// TextTransformations AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-webacl-sqlimatchstatement.html#cfn-wafv2-webacl-sqlimatchstatement-texttransformations
	TextTransformations *WebACL_TextTransformations `json:"TextTransformations,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *WebACL_SqliMatchStatement) AWSCloudFormationType() string {
	return "AWS::WAFv2::WebACL.SqliMatchStatement"
}
