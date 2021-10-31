package iotanalytics

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Dataset_VersioningConfiguration AWS CloudFormation Resource (AWS::IoTAnalytics::Dataset.VersioningConfiguration)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iotanalytics-dataset-versioningconfiguration.html
type Dataset_VersioningConfiguration struct {

	// MaxVersions AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iotanalytics-dataset-versioningconfiguration.html#cfn-iotanalytics-dataset-versioningconfiguration-maxversions
	MaxVersions int `json:"MaxVersions,omitempty"`

	// Unlimited AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iotanalytics-dataset-versioningconfiguration.html#cfn-iotanalytics-dataset-versioningconfiguration-unlimited
	Unlimited bool `json:"Unlimited,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Dataset_VersioningConfiguration) AWSCloudFormationType() string {
	return "AWS::IoTAnalytics::Dataset.VersioningConfiguration"
}
