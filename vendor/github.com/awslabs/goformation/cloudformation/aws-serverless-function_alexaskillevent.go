package cloudformation

// AWSServerlessFunction_AlexaSkillEvent AWS CloudFormation Resource (AWS::Serverless::Function.AlexaSkillEvent)
// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#alexaskill
type AWSServerlessFunction_AlexaSkillEvent struct {

	// Variables AWS CloudFormation Property
	// Required: false
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#alexaskill
	Variables map[string]string `json:"Variables,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSServerlessFunction_AlexaSkillEvent) AWSCloudFormationType() string {
	return "AWS::Serverless::Function.AlexaSkillEvent"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSServerlessFunction_AlexaSkillEvent) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
