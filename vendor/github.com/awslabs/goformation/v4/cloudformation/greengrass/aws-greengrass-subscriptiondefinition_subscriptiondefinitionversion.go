package greengrass

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// SubscriptionDefinition_SubscriptionDefinitionVersion AWS CloudFormation Resource (AWS::Greengrass::SubscriptionDefinition.SubscriptionDefinitionVersion)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-subscriptiondefinition-subscriptiondefinitionversion.html
type SubscriptionDefinition_SubscriptionDefinitionVersion struct {

	// Subscriptions AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-subscriptiondefinition-subscriptiondefinitionversion.html#cfn-greengrass-subscriptiondefinition-subscriptiondefinitionversion-subscriptions
	Subscriptions []SubscriptionDefinition_Subscription `json:"Subscriptions,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *SubscriptionDefinition_SubscriptionDefinitionVersion) AWSCloudFormationType() string {
	return "AWS::Greengrass::SubscriptionDefinition.SubscriptionDefinitionVersion"
}
