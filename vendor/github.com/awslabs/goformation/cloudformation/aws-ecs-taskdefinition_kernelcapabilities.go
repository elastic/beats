package cloudformation

// AWSECSTaskDefinition_KernelCapabilities AWS CloudFormation Resource (AWS::ECS::TaskDefinition.KernelCapabilities)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ecs-taskdefinition-kernelcapabilities.html
type AWSECSTaskDefinition_KernelCapabilities struct {

	// Add AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ecs-taskdefinition-kernelcapabilities.html#cfn-ecs-taskdefinition-kernelcapabilities-add
	Add []string `json:"Add,omitempty"`

	// Drop AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ecs-taskdefinition-kernelcapabilities.html#cfn-ecs-taskdefinition-kernelcapabilities-drop
	Drop []string `json:"Drop,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSECSTaskDefinition_KernelCapabilities) AWSCloudFormationType() string {
	return "AWS::ECS::TaskDefinition.KernelCapabilities"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSECSTaskDefinition_KernelCapabilities) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
