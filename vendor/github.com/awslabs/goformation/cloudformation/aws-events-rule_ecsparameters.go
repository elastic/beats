package cloudformation

// AWSEventsRule_EcsParameters AWS CloudFormation Resource (AWS::Events::Rule.EcsParameters)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-events-rule-ecsparameters.html
type AWSEventsRule_EcsParameters struct {

	// TaskCount AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-events-rule-ecsparameters.html#cfn-events-rule-ecsparameters-taskcount
	TaskCount int `json:"TaskCount,omitempty"`

	// TaskDefinitionArn AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-events-rule-ecsparameters.html#cfn-events-rule-ecsparameters-taskdefinitionarn
	TaskDefinitionArn string `json:"TaskDefinitionArn,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSEventsRule_EcsParameters) AWSCloudFormationType() string {
	return "AWS::Events::Rule.EcsParameters"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSEventsRule_EcsParameters) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
