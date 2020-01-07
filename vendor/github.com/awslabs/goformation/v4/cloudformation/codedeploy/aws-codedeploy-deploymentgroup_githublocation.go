package codedeploy

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// DeploymentGroup_GitHubLocation AWS CloudFormation Resource (AWS::CodeDeploy::DeploymentGroup.GitHubLocation)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codedeploy-deploymentgroup-deployment-revision-githublocation.html
type DeploymentGroup_GitHubLocation struct {

	// CommitId AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codedeploy-deploymentgroup-deployment-revision-githublocation.html#cfn-properties-codedeploy-deploymentgroup-deployment-revision-githublocation-commitid
	CommitId string `json:"CommitId,omitempty"`

	// Repository AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codedeploy-deploymentgroup-deployment-revision-githublocation.html#cfn-properties-codedeploy-deploymentgroup-deployment-revision-githublocation-repository
	Repository string `json:"Repository,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *DeploymentGroup_GitHubLocation) AWSCloudFormationType() string {
	return "AWS::CodeDeploy::DeploymentGroup.GitHubLocation"
}
