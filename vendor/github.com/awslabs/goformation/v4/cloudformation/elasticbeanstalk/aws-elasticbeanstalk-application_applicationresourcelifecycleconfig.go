package elasticbeanstalk

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Application_ApplicationResourceLifecycleConfig AWS CloudFormation Resource (AWS::ElasticBeanstalk::Application.ApplicationResourceLifecycleConfig)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticbeanstalk-application-applicationresourcelifecycleconfig.html
type Application_ApplicationResourceLifecycleConfig struct {

	// ServiceRole AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticbeanstalk-application-applicationresourcelifecycleconfig.html#cfn-elasticbeanstalk-application-applicationresourcelifecycleconfig-servicerole
	ServiceRole string `json:"ServiceRole,omitempty"`

	// VersionLifecycleConfig AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticbeanstalk-application-applicationresourcelifecycleconfig.html#cfn-elasticbeanstalk-application-applicationresourcelifecycleconfig-versionlifecycleconfig
	VersionLifecycleConfig *Application_ApplicationVersionLifecycleConfig `json:"VersionLifecycleConfig,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Application_ApplicationResourceLifecycleConfig) AWSCloudFormationType() string {
	return "AWS::ElasticBeanstalk::Application.ApplicationResourceLifecycleConfig"
}
