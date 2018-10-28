package cloudformation

// AWSElasticBeanstalkApplication_ApplicationResourceLifecycleConfig AWS CloudFormation Resource (AWS::ElasticBeanstalk::Application.ApplicationResourceLifecycleConfig)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticbeanstalk-application-applicationresourcelifecycleconfig.html
type AWSElasticBeanstalkApplication_ApplicationResourceLifecycleConfig struct {

	// ServiceRole AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticbeanstalk-application-applicationresourcelifecycleconfig.html#cfn-elasticbeanstalk-application-applicationresourcelifecycleconfig-servicerole
	ServiceRole string `json:"ServiceRole,omitempty"`

	// VersionLifecycleConfig AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticbeanstalk-application-applicationresourcelifecycleconfig.html#cfn-elasticbeanstalk-application-applicationresourcelifecycleconfig-versionlifecycleconfig
	VersionLifecycleConfig *AWSElasticBeanstalkApplication_ApplicationVersionLifecycleConfig `json:"VersionLifecycleConfig,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSElasticBeanstalkApplication_ApplicationResourceLifecycleConfig) AWSCloudFormationType() string {
	return "AWS::ElasticBeanstalk::Application.ApplicationResourceLifecycleConfig"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSElasticBeanstalkApplication_ApplicationResourceLifecycleConfig) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
