package cloudformation

// AWSGlueJob_JobCommand AWS CloudFormation Resource (AWS::Glue::Job.JobCommand)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-job-jobcommand.html
type AWSGlueJob_JobCommand struct {

	// Name AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-job-jobcommand.html#cfn-glue-job-jobcommand-name
	Name string `json:"Name,omitempty"`

	// ScriptLocation AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-job-jobcommand.html#cfn-glue-job-jobcommand-scriptlocation
	ScriptLocation string `json:"ScriptLocation,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSGlueJob_JobCommand) AWSCloudFormationType() string {
	return "AWS::Glue::Job.JobCommand"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSGlueJob_JobCommand) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
