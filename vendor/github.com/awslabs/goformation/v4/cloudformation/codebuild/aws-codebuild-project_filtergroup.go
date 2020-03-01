package codebuild

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Project_FilterGroup AWS CloudFormation Resource (AWS::CodeBuild::Project.FilterGroup)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codebuild-project-filtergroup.html
type Project_FilterGroup struct {

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Project_FilterGroup) AWSCloudFormationType() string {
	return "AWS::CodeBuild::Project.FilterGroup"
}
