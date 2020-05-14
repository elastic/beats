package greengrass

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Group_GroupVersion AWS CloudFormation Resource (AWS::Greengrass::Group.GroupVersion)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-group-groupversion.html
type Group_GroupVersion struct {

	// ConnectorDefinitionVersionArn AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-group-groupversion.html#cfn-greengrass-group-groupversion-connectordefinitionversionarn
	ConnectorDefinitionVersionArn string `json:"ConnectorDefinitionVersionArn,omitempty"`

	// CoreDefinitionVersionArn AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-group-groupversion.html#cfn-greengrass-group-groupversion-coredefinitionversionarn
	CoreDefinitionVersionArn string `json:"CoreDefinitionVersionArn,omitempty"`

	// DeviceDefinitionVersionArn AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-group-groupversion.html#cfn-greengrass-group-groupversion-devicedefinitionversionarn
	DeviceDefinitionVersionArn string `json:"DeviceDefinitionVersionArn,omitempty"`

	// FunctionDefinitionVersionArn AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-group-groupversion.html#cfn-greengrass-group-groupversion-functiondefinitionversionarn
	FunctionDefinitionVersionArn string `json:"FunctionDefinitionVersionArn,omitempty"`

	// LoggerDefinitionVersionArn AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-group-groupversion.html#cfn-greengrass-group-groupversion-loggerdefinitionversionarn
	LoggerDefinitionVersionArn string `json:"LoggerDefinitionVersionArn,omitempty"`

	// ResourceDefinitionVersionArn AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-group-groupversion.html#cfn-greengrass-group-groupversion-resourcedefinitionversionarn
	ResourceDefinitionVersionArn string `json:"ResourceDefinitionVersionArn,omitempty"`

	// SubscriptionDefinitionVersionArn AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-group-groupversion.html#cfn-greengrass-group-groupversion-subscriptiondefinitionversionarn
	SubscriptionDefinitionVersionArn string `json:"SubscriptionDefinitionVersionArn,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Group_GroupVersion) AWSCloudFormationType() string {
	return "AWS::Greengrass::Group.GroupVersion"
}
