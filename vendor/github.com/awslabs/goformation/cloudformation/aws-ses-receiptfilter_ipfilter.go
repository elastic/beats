package cloudformation

// AWSSESReceiptFilter_IpFilter AWS CloudFormation Resource (AWS::SES::ReceiptFilter.IpFilter)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ses-receiptfilter-ipfilter.html
type AWSSESReceiptFilter_IpFilter struct {

	// Cidr AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ses-receiptfilter-ipfilter.html#cfn-ses-receiptfilter-ipfilter-cidr
	Cidr string `json:"Cidr,omitempty"`

	// Policy AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ses-receiptfilter-ipfilter.html#cfn-ses-receiptfilter-ipfilter-policy
	Policy string `json:"Policy,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSSESReceiptFilter_IpFilter) AWSCloudFormationType() string {
	return "AWS::SES::ReceiptFilter.IpFilter"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSSESReceiptFilter_IpFilter) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
