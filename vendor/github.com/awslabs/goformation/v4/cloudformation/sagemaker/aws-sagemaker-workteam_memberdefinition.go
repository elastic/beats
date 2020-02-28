package sagemaker

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Workteam_MemberDefinition AWS CloudFormation Resource (AWS::SageMaker::Workteam.MemberDefinition)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-sagemaker-workteam-memberdefinition.html
type Workteam_MemberDefinition struct {

	// CognitoMemberDefinition AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-sagemaker-workteam-memberdefinition.html#cfn-sagemaker-workteam-memberdefinition-cognitomemberdefinition
	CognitoMemberDefinition *Workteam_CognitoMemberDefinition `json:"CognitoMemberDefinition,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Workteam_MemberDefinition) AWSCloudFormationType() string {
	return "AWS::SageMaker::Workteam.MemberDefinition"
}
