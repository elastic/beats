package serverless

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Function_IdentitySAMPT AWS CloudFormation Resource (AWS::Serverless::Function.IdentitySAMPT)
// See: https://github.com/awslabs/serverless-application-model/blob/master/docs/policy_templates.rst
type Function_IdentitySAMPT struct {

	// IdentityName AWS CloudFormation Property
	// Required: true
	// See: https://github.com/awslabs/serverless-application-model/blob/master/docs/policy_templates.rst
	IdentityName string `json:"IdentityName,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Function_IdentitySAMPT) AWSCloudFormationType() string {
	return "AWS::Serverless::Function.IdentitySAMPT"
}
