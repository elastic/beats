package cloudformation

// AWSS3Bucket_WebsiteConfiguration AWS CloudFormation Resource (AWS::S3::Bucket.WebsiteConfiguration)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-websiteconfiguration.html
type AWSS3Bucket_WebsiteConfiguration struct {

	// ErrorDocument AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-websiteconfiguration.html#cfn-s3-websiteconfiguration-errordocument
	ErrorDocument string `json:"ErrorDocument,omitempty"`

	// IndexDocument AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-websiteconfiguration.html#cfn-s3-websiteconfiguration-indexdocument
	IndexDocument string `json:"IndexDocument,omitempty"`

	// RedirectAllRequestsTo AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-websiteconfiguration.html#cfn-s3-websiteconfiguration-redirectallrequeststo
	RedirectAllRequestsTo *AWSS3Bucket_RedirectAllRequestsTo `json:"RedirectAllRequestsTo,omitempty"`

	// RoutingRules AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-websiteconfiguration.html#cfn-s3-websiteconfiguration-routingrules
	RoutingRules []AWSS3Bucket_RoutingRule `json:"RoutingRules,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSS3Bucket_WebsiteConfiguration) AWSCloudFormationType() string {
	return "AWS::S3::Bucket.WebsiteConfiguration"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSS3Bucket_WebsiteConfiguration) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
