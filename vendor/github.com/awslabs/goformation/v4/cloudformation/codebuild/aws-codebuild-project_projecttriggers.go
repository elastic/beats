package codebuild

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Project_ProjectTriggers AWS CloudFormation Resource (AWS::CodeBuild::Project.ProjectTriggers)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codebuild-project-projecttriggers.html
type Project_ProjectTriggers struct {

	// FilterGroups AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codebuild-project-projecttriggers.html#cfn-codebuild-project-projecttriggers-filtergroups
	FilterGroups []Project_FilterGroup `json:"FilterGroups,omitempty"`

	// Webhook AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codebuild-project-projecttriggers.html#cfn-codebuild-project-projecttriggers-webhook
	Webhook bool `json:"Webhook,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Project_ProjectTriggers) AWSCloudFormationType() string {
	return "AWS::CodeBuild::Project.ProjectTriggers"
}
