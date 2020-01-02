package sagemaker

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Workteam_CognitoMemberDefinition AWS CloudFormation Resource (AWS::SageMaker::Workteam.CognitoMemberDefinition)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-sagemaker-workteam-cognitomemberdefinition.html
type Workteam_CognitoMemberDefinition struct {

	// CognitoClientId AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-sagemaker-workteam-cognitomemberdefinition.html#cfn-sagemaker-workteam-cognitomemberdefinition-cognitoclientid
	CognitoClientId string `json:"CognitoClientId,omitempty"`

	// CognitoUserGroup AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-sagemaker-workteam-cognitomemberdefinition.html#cfn-sagemaker-workteam-cognitomemberdefinition-cognitousergroup
	CognitoUserGroup string `json:"CognitoUserGroup,omitempty"`

	// CognitoUserPool AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-sagemaker-workteam-cognitomemberdefinition.html#cfn-sagemaker-workteam-cognitomemberdefinition-cognitouserpool
	CognitoUserPool string `json:"CognitoUserPool,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Workteam_CognitoMemberDefinition) AWSCloudFormationType() string {
	return "AWS::SageMaker::Workteam.CognitoMemberDefinition"
}
