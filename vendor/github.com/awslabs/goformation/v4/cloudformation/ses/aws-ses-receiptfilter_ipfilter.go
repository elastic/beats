package ses

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// ReceiptFilter_IpFilter AWS CloudFormation Resource (AWS::SES::ReceiptFilter.IpFilter)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ses-receiptfilter-ipfilter.html
type ReceiptFilter_IpFilter struct {

	// Cidr AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ses-receiptfilter-ipfilter.html#cfn-ses-receiptfilter-ipfilter-cidr
	Cidr string `json:"Cidr,omitempty"`

	// Policy AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ses-receiptfilter-ipfilter.html#cfn-ses-receiptfilter-ipfilter-policy
	Policy string `json:"Policy,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *ReceiptFilter_IpFilter) AWSCloudFormationType() string {
	return "AWS::SES::ReceiptFilter.IpFilter"
}
