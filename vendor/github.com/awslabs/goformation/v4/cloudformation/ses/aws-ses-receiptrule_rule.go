package ses

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// ReceiptRule_Rule AWS CloudFormation Resource (AWS::SES::ReceiptRule.Rule)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ses-receiptrule-rule.html
type ReceiptRule_Rule struct {

	// Actions AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ses-receiptrule-rule.html#cfn-ses-receiptrule-rule-actions
	Actions []ReceiptRule_Action `json:"Actions,omitempty"`

	// Enabled AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ses-receiptrule-rule.html#cfn-ses-receiptrule-rule-enabled
	Enabled bool `json:"Enabled,omitempty"`

	// Name AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ses-receiptrule-rule.html#cfn-ses-receiptrule-rule-name
	Name string `json:"Name,omitempty"`

	// Recipients AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ses-receiptrule-rule.html#cfn-ses-receiptrule-rule-recipients
	Recipients []string `json:"Recipients,omitempty"`

	// ScanEnabled AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ses-receiptrule-rule.html#cfn-ses-receiptrule-rule-scanenabled
	ScanEnabled bool `json:"ScanEnabled,omitempty"`

	// TlsPolicy AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ses-receiptrule-rule.html#cfn-ses-receiptrule-rule-tlspolicy
	TlsPolicy string `json:"TlsPolicy,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *ReceiptRule_Rule) AWSCloudFormationType() string {
	return "AWS::SES::ReceiptRule.Rule"
}
