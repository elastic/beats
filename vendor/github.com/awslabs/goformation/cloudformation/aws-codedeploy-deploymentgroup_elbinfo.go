package cloudformation

// AWSCodeDeployDeploymentGroup_ELBInfo AWS CloudFormation Resource (AWS::CodeDeploy::DeploymentGroup.ELBInfo)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codedeploy-deploymentgroup-elbinfo.html
type AWSCodeDeployDeploymentGroup_ELBInfo struct {

	// Name AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codedeploy-deploymentgroup-elbinfo.html#cfn-codedeploy-deploymentgroup-elbinfo-name
	Name string `json:"Name,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSCodeDeployDeploymentGroup_ELBInfo) AWSCloudFormationType() string {
	return "AWS::CodeDeploy::DeploymentGroup.ELBInfo"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSCodeDeployDeploymentGroup_ELBInfo) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
