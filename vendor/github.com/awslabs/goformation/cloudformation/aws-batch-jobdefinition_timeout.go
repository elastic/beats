package cloudformation

// AWSBatchJobDefinition_Timeout AWS CloudFormation Resource (AWS::Batch::JobDefinition.Timeout)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-batch-jobdefinition-timeout.html
type AWSBatchJobDefinition_Timeout struct {

	// AttemptDurationSeconds AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-batch-jobdefinition-timeout.html#cfn-batch-jobdefinition-timeout-attemptdurationseconds
	AttemptDurationSeconds int `json:"AttemptDurationSeconds,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSBatchJobDefinition_Timeout) AWSCloudFormationType() string {
	return "AWS::Batch::JobDefinition.Timeout"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSBatchJobDefinition_Timeout) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
