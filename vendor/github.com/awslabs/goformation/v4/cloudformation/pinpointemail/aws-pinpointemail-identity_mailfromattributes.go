package pinpointemail

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Identity_MailFromAttributes AWS CloudFormation Resource (AWS::PinpointEmail::Identity.MailFromAttributes)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-pinpointemail-identity-mailfromattributes.html
type Identity_MailFromAttributes struct {

	// BehaviorOnMxFailure AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-pinpointemail-identity-mailfromattributes.html#cfn-pinpointemail-identity-mailfromattributes-behavioronmxfailure
	BehaviorOnMxFailure string `json:"BehaviorOnMxFailure,omitempty"`

	// MailFromDomain AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-pinpointemail-identity-mailfromattributes.html#cfn-pinpointemail-identity-mailfromattributes-mailfromdomain
	MailFromDomain string `json:"MailFromDomain,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Identity_MailFromAttributes) AWSCloudFormationType() string {
	return "AWS::PinpointEmail::Identity.MailFromAttributes"
}
