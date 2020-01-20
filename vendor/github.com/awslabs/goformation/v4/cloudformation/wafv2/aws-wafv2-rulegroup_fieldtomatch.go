package wafv2

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// RuleGroup_FieldToMatch AWS CloudFormation Resource (AWS::WAFv2::RuleGroup.FieldToMatch)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-fieldtomatch.html
type RuleGroup_FieldToMatch struct {

	// AllQueryArguments AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-fieldtomatch.html#cfn-wafv2-rulegroup-fieldtomatch-allqueryarguments
	AllQueryArguments *RuleGroup_AllQueryArguments `json:"AllQueryArguments,omitempty"`

	// Body AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-fieldtomatch.html#cfn-wafv2-rulegroup-fieldtomatch-body
	Body *RuleGroup_Body `json:"Body,omitempty"`

	// Method AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-fieldtomatch.html#cfn-wafv2-rulegroup-fieldtomatch-method
	Method *RuleGroup_Method `json:"Method,omitempty"`

	// QueryString AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-fieldtomatch.html#cfn-wafv2-rulegroup-fieldtomatch-querystring
	QueryString *RuleGroup_QueryString `json:"QueryString,omitempty"`

	// SingleHeader AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-fieldtomatch.html#cfn-wafv2-rulegroup-fieldtomatch-singleheader
	SingleHeader *RuleGroup_SingleHeader `json:"SingleHeader,omitempty"`

	// SingleQueryArgument AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-fieldtomatch.html#cfn-wafv2-rulegroup-fieldtomatch-singlequeryargument
	SingleQueryArgument *RuleGroup_SingleQueryArgument `json:"SingleQueryArgument,omitempty"`

	// UriPath AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-fieldtomatch.html#cfn-wafv2-rulegroup-fieldtomatch-uripath
	UriPath *RuleGroup_UriPath `json:"UriPath,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *RuleGroup_FieldToMatch) AWSCloudFormationType() string {
	return "AWS::WAFv2::RuleGroup.FieldToMatch"
}
