package ses

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// ReceiptRule_Action AWS CloudFormation Resource (AWS::SES::ReceiptRule.Action)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ses-receiptrule-action.html
type ReceiptRule_Action struct {

	// AddHeaderAction AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ses-receiptrule-action.html#cfn-ses-receiptrule-action-addheaderaction
	AddHeaderAction *ReceiptRule_AddHeaderAction `json:"AddHeaderAction,omitempty"`

	// BounceAction AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ses-receiptrule-action.html#cfn-ses-receiptrule-action-bounceaction
	BounceAction *ReceiptRule_BounceAction `json:"BounceAction,omitempty"`

	// LambdaAction AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ses-receiptrule-action.html#cfn-ses-receiptrule-action-lambdaaction
	LambdaAction *ReceiptRule_LambdaAction `json:"LambdaAction,omitempty"`

	// S3Action AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ses-receiptrule-action.html#cfn-ses-receiptrule-action-s3action
	S3Action *ReceiptRule_S3Action `json:"S3Action,omitempty"`

	// SNSAction AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ses-receiptrule-action.html#cfn-ses-receiptrule-action-snsaction
	SNSAction *ReceiptRule_SNSAction `json:"SNSAction,omitempty"`

	// StopAction AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ses-receiptrule-action.html#cfn-ses-receiptrule-action-stopaction
	StopAction *ReceiptRule_StopAction `json:"StopAction,omitempty"`

	// WorkmailAction AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ses-receiptrule-action.html#cfn-ses-receiptrule-action-workmailaction
	WorkmailAction *ReceiptRule_WorkmailAction `json:"WorkmailAction,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *ReceiptRule_Action) AWSCloudFormationType() string {
	return "AWS::SES::ReceiptRule.Action"
}
