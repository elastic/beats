package lakeformation

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// DataLakeSettings_Admins AWS CloudFormation Resource (AWS::LakeFormation::DataLakeSettings.Admins)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-lakeformation-datalakesettings-admins.html
type DataLakeSettings_Admins struct {

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *DataLakeSettings_Admins) AWSCloudFormationType() string {
	return "AWS::LakeFormation::DataLakeSettings.Admins"
}
