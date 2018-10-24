package cloudformation

// AWSEventsRule_InputTransformer AWS CloudFormation Resource (AWS::Events::Rule.InputTransformer)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-events-rule-inputtransformer.html
type AWSEventsRule_InputTransformer struct {

	// InputPathsMap AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-events-rule-inputtransformer.html#cfn-events-rule-inputtransformer-inputpathsmap
	InputPathsMap map[string]string `json:"InputPathsMap,omitempty"`

	// InputTemplate AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-events-rule-inputtransformer.html#cfn-events-rule-inputtransformer-inputtemplate
	InputTemplate string `json:"InputTemplate,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSEventsRule_InputTransformer) AWSCloudFormationType() string {
	return "AWS::Events::Rule.InputTransformer"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSEventsRule_InputTransformer) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
