package wafv2

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// WebACL_AllQueryArguments AWS CloudFormation Resource (AWS::WAFv2::WebACL.AllQueryArguments)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-webacl-allqueryarguments.html
type WebACL_AllQueryArguments struct {

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *WebACL_AllQueryArguments) AWSCloudFormationType() string {
	return "AWS::WAFv2::WebACL.AllQueryArguments"
}
