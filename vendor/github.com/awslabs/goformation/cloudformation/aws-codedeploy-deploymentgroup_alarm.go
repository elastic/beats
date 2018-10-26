package cloudformation

// AWSCodeDeployDeploymentGroup_Alarm AWS CloudFormation Resource (AWS::CodeDeploy::DeploymentGroup.Alarm)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codedeploy-deploymentgroup-alarm.html
type AWSCodeDeployDeploymentGroup_Alarm struct {

	// Name AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codedeploy-deploymentgroup-alarm.html#cfn-codedeploy-deploymentgroup-alarm-name
	Name string `json:"Name,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSCodeDeployDeploymentGroup_Alarm) AWSCloudFormationType() string {
	return "AWS::CodeDeploy::DeploymentGroup.Alarm"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSCodeDeployDeploymentGroup_Alarm) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
