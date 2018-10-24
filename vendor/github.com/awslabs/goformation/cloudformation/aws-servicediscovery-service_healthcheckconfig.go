package cloudformation

// AWSServiceDiscoveryService_HealthCheckConfig AWS CloudFormation Resource (AWS::ServiceDiscovery::Service.HealthCheckConfig)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-servicediscovery-service-healthcheckconfig.html
type AWSServiceDiscoveryService_HealthCheckConfig struct {

	// FailureThreshold AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-servicediscovery-service-healthcheckconfig.html#cfn-servicediscovery-service-healthcheckconfig-failurethreshold
	FailureThreshold float64 `json:"FailureThreshold,omitempty"`

	// ResourcePath AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-servicediscovery-service-healthcheckconfig.html#cfn-servicediscovery-service-healthcheckconfig-resourcepath
	ResourcePath string `json:"ResourcePath,omitempty"`

	// Type AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-servicediscovery-service-healthcheckconfig.html#cfn-servicediscovery-service-healthcheckconfig-type
	Type string `json:"Type,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSServiceDiscoveryService_HealthCheckConfig) AWSCloudFormationType() string {
	return "AWS::ServiceDiscovery::Service.HealthCheckConfig"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSServiceDiscoveryService_HealthCheckConfig) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
