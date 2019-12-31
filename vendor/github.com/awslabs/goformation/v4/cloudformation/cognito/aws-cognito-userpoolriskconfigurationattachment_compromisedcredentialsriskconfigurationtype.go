package cognito

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// UserPoolRiskConfigurationAttachment_CompromisedCredentialsRiskConfigurationType AWS CloudFormation Resource (AWS::Cognito::UserPoolRiskConfigurationAttachment.CompromisedCredentialsRiskConfigurationType)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cognito-userpoolriskconfigurationattachment-compromisedcredentialsriskconfigurationtype.html
type UserPoolRiskConfigurationAttachment_CompromisedCredentialsRiskConfigurationType struct {

	// Actions AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cognito-userpoolriskconfigurationattachment-compromisedcredentialsriskconfigurationtype.html#cfn-cognito-userpoolriskconfigurationattachment-compromisedcredentialsriskconfigurationtype-actions
	Actions *UserPoolRiskConfigurationAttachment_CompromisedCredentialsActionsType `json:"Actions,omitempty"`

	// EventFilter AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cognito-userpoolriskconfigurationattachment-compromisedcredentialsriskconfigurationtype.html#cfn-cognito-userpoolriskconfigurationattachment-compromisedcredentialsriskconfigurationtype-eventfilter
	EventFilter []string `json:"EventFilter,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *UserPoolRiskConfigurationAttachment_CompromisedCredentialsRiskConfigurationType) AWSCloudFormationType() string {
	return "AWS::Cognito::UserPoolRiskConfigurationAttachment.CompromisedCredentialsRiskConfigurationType"
}
