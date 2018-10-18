package cloudformation

// AWSECSService_PlacementConstraint AWS CloudFormation Resource (AWS::ECS::Service.PlacementConstraint)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ecs-service-placementconstraint.html
type AWSECSService_PlacementConstraint struct {

	// Expression AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ecs-service-placementconstraint.html#cfn-ecs-service-placementconstraint-expression
	Expression string `json:"Expression,omitempty"`

	// Type AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ecs-service-placementconstraint.html#cfn-ecs-service-placementconstraint-type
	Type string `json:"Type,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSECSService_PlacementConstraint) AWSCloudFormationType() string {
	return "AWS::ECS::Service.PlacementConstraint"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSECSService_PlacementConstraint) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
