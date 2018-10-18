package cloudformation

// AWSEventsRule_RunCommandParameters AWS CloudFormation Resource (AWS::Events::Rule.RunCommandParameters)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-events-rule-runcommandparameters.html
type AWSEventsRule_RunCommandParameters struct {

	// RunCommandTargets AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-events-rule-runcommandparameters.html#cfn-events-rule-runcommandparameters-runcommandtargets
	RunCommandTargets []AWSEventsRule_RunCommandTarget `json:"RunCommandTargets,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSEventsRule_RunCommandParameters) AWSCloudFormationType() string {
	return "AWS::Events::Rule.RunCommandParameters"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSEventsRule_RunCommandParameters) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
