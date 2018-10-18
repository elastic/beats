package cloudformation

// AWSElasticLoadBalancingV2ListenerCertificate_Certificate AWS CloudFormation Resource (AWS::ElasticLoadBalancingV2::ListenerCertificate.Certificate)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticloadbalancingv2-listener-certificates.html
type AWSElasticLoadBalancingV2ListenerCertificate_Certificate struct {

	// CertificateArn AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticloadbalancingv2-listener-certificates.html#cfn-elasticloadbalancingv2-listener-certificates-certificatearn
	CertificateArn string `json:"CertificateArn,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSElasticLoadBalancingV2ListenerCertificate_Certificate) AWSCloudFormationType() string {
	return "AWS::ElasticLoadBalancingV2::ListenerCertificate.Certificate"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSElasticLoadBalancingV2ListenerCertificate_Certificate) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
