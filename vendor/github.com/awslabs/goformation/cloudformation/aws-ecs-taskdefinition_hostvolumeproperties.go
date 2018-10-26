package cloudformation

// AWSECSTaskDefinition_HostVolumeProperties AWS CloudFormation Resource (AWS::ECS::TaskDefinition.HostVolumeProperties)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ecs-taskdefinition-volumes-host.html
type AWSECSTaskDefinition_HostVolumeProperties struct {

	// SourcePath AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ecs-taskdefinition-volumes-host.html#cfn-ecs-taskdefinition-volumes-host-sourcepath
	SourcePath string `json:"SourcePath,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSECSTaskDefinition_HostVolumeProperties) AWSCloudFormationType() string {
	return "AWS::ECS::TaskDefinition.HostVolumeProperties"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSECSTaskDefinition_HostVolumeProperties) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
