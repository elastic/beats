package cloudformation

// AWSCertificateManagerCertificate_DomainValidationOption AWS CloudFormation Resource (AWS::CertificateManager::Certificate.DomainValidationOption)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-certificatemanager-certificate-domainvalidationoption.html
type AWSCertificateManagerCertificate_DomainValidationOption struct {

	// DomainName AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-certificatemanager-certificate-domainvalidationoption.html#cfn-certificatemanager-certificate-domainvalidationoptions-domainname
	DomainName string `json:"DomainName,omitempty"`

	// ValidationDomain AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-certificatemanager-certificate-domainvalidationoption.html#cfn-certificatemanager-certificate-domainvalidationoption-validationdomain
	ValidationDomain string `json:"ValidationDomain,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSCertificateManagerCertificate_DomainValidationOption) AWSCloudFormationType() string {
	return "AWS::CertificateManager::Certificate.DomainValidationOption"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSCertificateManagerCertificate_DomainValidationOption) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
