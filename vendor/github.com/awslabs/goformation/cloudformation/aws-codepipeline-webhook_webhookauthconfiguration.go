package cloudformation

// AWSCodePipelineWebhook_WebhookAuthConfiguration AWS CloudFormation Resource (AWS::CodePipeline::Webhook.WebhookAuthConfiguration)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codepipeline-webhook-webhookauthconfiguration.html
type AWSCodePipelineWebhook_WebhookAuthConfiguration struct {

	// AllowedIPRange AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codepipeline-webhook-webhookauthconfiguration.html#cfn-codepipeline-webhook-webhookauthconfiguration-allowediprange
	AllowedIPRange string `json:"AllowedIPRange,omitempty"`

	// SecretToken AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codepipeline-webhook-webhookauthconfiguration.html#cfn-codepipeline-webhook-webhookauthconfiguration-secrettoken
	SecretToken string `json:"SecretToken,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSCodePipelineWebhook_WebhookAuthConfiguration) AWSCloudFormationType() string {
	return "AWS::CodePipeline::Webhook.WebhookAuthConfiguration"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSCodePipelineWebhook_WebhookAuthConfiguration) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
