package wafv2

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// WebACL_FieldToMatch AWS CloudFormation Resource (AWS::WAFv2::WebACL.FieldToMatch)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-webacl-fieldtomatch.html
type WebACL_FieldToMatch struct {

	// AllQueryArguments AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-webacl-fieldtomatch.html#cfn-wafv2-webacl-fieldtomatch-allqueryarguments
	AllQueryArguments *WebACL_AllQueryArguments `json:"AllQueryArguments,omitempty"`

	// Body AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-webacl-fieldtomatch.html#cfn-wafv2-webacl-fieldtomatch-body
	Body *WebACL_Body `json:"Body,omitempty"`

	// Method AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-webacl-fieldtomatch.html#cfn-wafv2-webacl-fieldtomatch-method
	Method *WebACL_Method `json:"Method,omitempty"`

	// QueryString AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-webacl-fieldtomatch.html#cfn-wafv2-webacl-fieldtomatch-querystring
	QueryString *WebACL_QueryString `json:"QueryString,omitempty"`

	// SingleHeader AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-webacl-fieldtomatch.html#cfn-wafv2-webacl-fieldtomatch-singleheader
	SingleHeader *WebACL_SingleHeader `json:"SingleHeader,omitempty"`

	// SingleQueryArgument AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-webacl-fieldtomatch.html#cfn-wafv2-webacl-fieldtomatch-singlequeryargument
	SingleQueryArgument *WebACL_SingleQueryArgument `json:"SingleQueryArgument,omitempty"`

	// UriPath AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-webacl-fieldtomatch.html#cfn-wafv2-webacl-fieldtomatch-uripath
	UriPath *WebACL_UriPath `json:"UriPath,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *WebACL_FieldToMatch) AWSCloudFormationType() string {
	return "AWS::WAFv2::WebACL.FieldToMatch"
}
