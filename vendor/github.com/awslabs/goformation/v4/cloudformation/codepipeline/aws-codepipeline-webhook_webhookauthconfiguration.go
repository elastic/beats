package codepipeline

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Webhook_WebhookAuthConfiguration AWS CloudFormation Resource (AWS::CodePipeline::Webhook.WebhookAuthConfiguration)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codepipeline-webhook-webhookauthconfiguration.html
type Webhook_WebhookAuthConfiguration struct {

	// AllowedIPRange AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codepipeline-webhook-webhookauthconfiguration.html#cfn-codepipeline-webhook-webhookauthconfiguration-allowediprange
	AllowedIPRange string `json:"AllowedIPRange,omitempty"`

	// SecretToken AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codepipeline-webhook-webhookauthconfiguration.html#cfn-codepipeline-webhook-webhookauthconfiguration-secrettoken
	SecretToken string `json:"SecretToken,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Webhook_WebhookAuthConfiguration) AWSCloudFormationType() string {
	return "AWS::CodePipeline::Webhook.WebhookAuthConfiguration"
}
