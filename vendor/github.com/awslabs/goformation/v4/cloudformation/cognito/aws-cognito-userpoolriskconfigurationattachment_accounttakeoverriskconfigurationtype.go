package cognito

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// UserPoolRiskConfigurationAttachment_AccountTakeoverRiskConfigurationType AWS CloudFormation Resource (AWS::Cognito::UserPoolRiskConfigurationAttachment.AccountTakeoverRiskConfigurationType)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cognito-userpoolriskconfigurationattachment-accounttakeoverriskconfigurationtype.html
type UserPoolRiskConfigurationAttachment_AccountTakeoverRiskConfigurationType struct {

	// Actions AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cognito-userpoolriskconfigurationattachment-accounttakeoverriskconfigurationtype.html#cfn-cognito-userpoolriskconfigurationattachment-accounttakeoverriskconfigurationtype-actions
	Actions *UserPoolRiskConfigurationAttachment_AccountTakeoverActionsType `json:"Actions,omitempty"`

	// NotifyConfiguration AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cognito-userpoolriskconfigurationattachment-accounttakeoverriskconfigurationtype.html#cfn-cognito-userpoolriskconfigurationattachment-accounttakeoverriskconfigurationtype-notifyconfiguration
	NotifyConfiguration *UserPoolRiskConfigurationAttachment_NotifyConfigurationType `json:"NotifyConfiguration,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *UserPoolRiskConfigurationAttachment_AccountTakeoverRiskConfigurationType) AWSCloudFormationType() string {
	return "AWS::Cognito::UserPoolRiskConfigurationAttachment.AccountTakeoverRiskConfigurationType"
}
