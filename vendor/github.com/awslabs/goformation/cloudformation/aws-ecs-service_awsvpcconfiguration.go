package cloudformation

// AWSECSService_AwsVpcConfiguration AWS CloudFormation Resource (AWS::ECS::Service.AwsVpcConfiguration)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ecs-service-awsvpcconfiguration.html
type AWSECSService_AwsVpcConfiguration struct {

	// AssignPublicIp AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ecs-service-awsvpcconfiguration.html#cfn-ecs-service-awsvpcconfiguration-assignpublicip
	AssignPublicIp string `json:"AssignPublicIp,omitempty"`

	// SecurityGroups AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ecs-service-awsvpcconfiguration.html#cfn-ecs-service-awsvpcconfiguration-securitygroups
	SecurityGroups []string `json:"SecurityGroups,omitempty"`

	// Subnets AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ecs-service-awsvpcconfiguration.html#cfn-ecs-service-awsvpcconfiguration-subnets
	Subnets []string `json:"Subnets,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSECSService_AwsVpcConfiguration) AWSCloudFormationType() string {
	return "AWS::ECS::Service.AwsVpcConfiguration"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSECSService_AwsVpcConfiguration) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
