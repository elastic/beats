package budgets

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Budget_NotificationWithSubscribers AWS CloudFormation Resource (AWS::Budgets::Budget.NotificationWithSubscribers)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-budgets-budget-notificationwithsubscribers.html
type Budget_NotificationWithSubscribers struct {

	// Notification AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-budgets-budget-notificationwithsubscribers.html#cfn-budgets-budget-notificationwithsubscribers-notification
	Notification *Budget_Notification `json:"Notification,omitempty"`

	// Subscribers AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-budgets-budget-notificationwithsubscribers.html#cfn-budgets-budget-notificationwithsubscribers-subscribers
	Subscribers []Budget_Subscriber `json:"Subscribers,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Budget_NotificationWithSubscribers) AWSCloudFormationType() string {
	return "AWS::Budgets::Budget.NotificationWithSubscribers"
}
