package codebuild

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Project_GitSubmodulesConfig AWS CloudFormation Resource (AWS::CodeBuild::Project.GitSubmodulesConfig)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codebuild-project-gitsubmodulesconfig.html
type Project_GitSubmodulesConfig struct {

	// FetchSubmodules AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codebuild-project-gitsubmodulesconfig.html#cfn-codebuild-project-gitsubmodulesconfig-fetchsubmodules
	FetchSubmodules bool `json:"FetchSubmodules"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Project_GitSubmodulesConfig) AWSCloudFormationType() string {
	return "AWS::CodeBuild::Project.GitSubmodulesConfig"
}
