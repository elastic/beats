package cloudformation

// AWSElasticLoadBalancingLoadBalancer_Listeners AWS CloudFormation Resource (AWS::ElasticLoadBalancing::LoadBalancer.Listeners)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-elb-listener.html
type AWSElasticLoadBalancingLoadBalancer_Listeners struct {

	// InstancePort AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-elb-listener.html#cfn-ec2-elb-listener-instanceport
	InstancePort string `json:"InstancePort,omitempty"`

	// InstanceProtocol AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-elb-listener.html#cfn-ec2-elb-listener-instanceprotocol
	InstanceProtocol string `json:"InstanceProtocol,omitempty"`

	// LoadBalancerPort AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-elb-listener.html#cfn-ec2-elb-listener-loadbalancerport
	LoadBalancerPort string `json:"LoadBalancerPort,omitempty"`

	// PolicyNames AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-elb-listener.html#cfn-ec2-elb-listener-policynames
	PolicyNames []string `json:"PolicyNames,omitempty"`

	// Protocol AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-elb-listener.html#cfn-ec2-elb-listener-protocol
	Protocol string `json:"Protocol,omitempty"`

	// SSLCertificateId AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-elb-listener.html#cfn-ec2-elb-listener-sslcertificateid
	SSLCertificateId string `json:"SSLCertificateId,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSElasticLoadBalancingLoadBalancer_Listeners) AWSCloudFormationType() string {
	return "AWS::ElasticLoadBalancing::LoadBalancer.Listeners"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSElasticLoadBalancingLoadBalancer_Listeners) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
