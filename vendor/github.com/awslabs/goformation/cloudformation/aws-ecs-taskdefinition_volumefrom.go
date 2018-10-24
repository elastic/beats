package cloudformation

// AWSECSTaskDefinition_VolumeFrom AWS CloudFormation Resource (AWS::ECS::TaskDefinition.VolumeFrom)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ecs-taskdefinition-containerdefinitions-volumesfrom.html
type AWSECSTaskDefinition_VolumeFrom struct {

	// ReadOnly AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ecs-taskdefinition-containerdefinitions-volumesfrom.html#cfn-ecs-taskdefinition-containerdefinition-volumesfrom-readonly
	ReadOnly bool `json:"ReadOnly,omitempty"`

	// SourceContainer AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ecs-taskdefinition-containerdefinitions-volumesfrom.html#cfn-ecs-taskdefinition-containerdefinition-volumesfrom-sourcecontainer
	SourceContainer string `json:"SourceContainer,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSECSTaskDefinition_VolumeFrom) AWSCloudFormationType() string {
	return "AWS::ECS::TaskDefinition.VolumeFrom"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSECSTaskDefinition_VolumeFrom) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
