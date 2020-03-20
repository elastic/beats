package kinesisanalyticsv2

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Application_PropertyGroup AWS CloudFormation Resource (AWS::KinesisAnalyticsV2::Application.PropertyGroup)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisanalyticsv2-application-propertygroup.html
type Application_PropertyGroup struct {

	// PropertyGroupId AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisanalyticsv2-application-propertygroup.html#cfn-kinesisanalyticsv2-application-propertygroup-propertygroupid
	PropertyGroupId string `json:"PropertyGroupId,omitempty"`

	// PropertyMap AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisanalyticsv2-application-propertygroup.html#cfn-kinesisanalyticsv2-application-propertygroup-propertymap
	PropertyMap interface{} `json:"PropertyMap,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Application_PropertyGroup) AWSCloudFormationType() string {
	return "AWS::KinesisAnalyticsV2::Application.PropertyGroup"
}
