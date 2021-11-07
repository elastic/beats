package wafv2

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// WebACL_WebACL AWS CloudFormation Resource (AWS::WAFv2::WebACL.WebACL)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-webacl-webacl.html
type WebACL_WebACL struct {

	// ARN AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-webacl-webacl.html#cfn-wafv2-webacl-webacl-arn
	ARN string `json:"ARN,omitempty"`

	// Capacity AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-webacl-webacl.html#cfn-wafv2-webacl-webacl-capacity
	Capacity int `json:"Capacity,omitempty"`

	// DefaultAction AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-webacl-webacl.html#cfn-wafv2-webacl-webacl-defaultaction
	DefaultAction *WebACL_DefaultAction `json:"DefaultAction,omitempty"`

	// Description AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-webacl-webacl.html#cfn-wafv2-webacl-webacl-description
	Description string `json:"Description,omitempty"`

	// Id AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-webacl-webacl.html#cfn-wafv2-webacl-webacl-id
	Id string `json:"Id,omitempty"`

	// Name AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-webacl-webacl.html#cfn-wafv2-webacl-webacl-name
	Name string `json:"Name,omitempty"`

	// Rules AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-webacl-webacl.html#cfn-wafv2-webacl-webacl-rules
	Rules *WebACL_Rules `json:"Rules,omitempty"`

	// VisibilityConfig AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-webacl-webacl.html#cfn-wafv2-webacl-webacl-visibilityconfig
	VisibilityConfig *WebACL_VisibilityConfig `json:"VisibilityConfig,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *WebACL_WebACL) AWSCloudFormationType() string {
	return "AWS::WAFv2::WebACL.WebACL"
}
