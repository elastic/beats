package cloudformation

// AWSCodeDeployDeploymentGroup_RevisionLocation AWS CloudFormation Resource (AWS::CodeDeploy::DeploymentGroup.RevisionLocation)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codedeploy-deploymentgroup-deployment-revision.html
type AWSCodeDeployDeploymentGroup_RevisionLocation struct {

	// GitHubLocation AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codedeploy-deploymentgroup-deployment-revision.html#cfn-properties-codedeploy-deploymentgroup-deployment-revision-githublocation
	GitHubLocation *AWSCodeDeployDeploymentGroup_GitHubLocation `json:"GitHubLocation,omitempty"`

	// RevisionType AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codedeploy-deploymentgroup-deployment-revision.html#cfn-properties-codedeploy-deploymentgroup-deployment-revision-revisiontype
	RevisionType string `json:"RevisionType,omitempty"`

	// S3Location AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codedeploy-deploymentgroup-deployment-revision.html#cfn-properties-codedeploy-deploymentgroup-deployment-revision-s3location
	S3Location *AWSCodeDeployDeploymentGroup_S3Location `json:"S3Location,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSCodeDeployDeploymentGroup_RevisionLocation) AWSCloudFormationType() string {
	return "AWS::CodeDeploy::DeploymentGroup.RevisionLocation"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSCodeDeployDeploymentGroup_RevisionLocation) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
