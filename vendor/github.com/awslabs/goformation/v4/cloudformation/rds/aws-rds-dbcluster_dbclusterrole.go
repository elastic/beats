package rds

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// DBCluster_DBClusterRole AWS CloudFormation Resource (AWS::RDS::DBCluster.DBClusterRole)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-rds-dbcluster-dbclusterrole.html
type DBCluster_DBClusterRole struct {

	// FeatureName AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-rds-dbcluster-dbclusterrole.html#cfn-rds-dbcluster-dbclusterrole-featurename
	FeatureName string `json:"FeatureName,omitempty"`

	// RoleArn AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-rds-dbcluster-dbclusterrole.html#cfn-rds-dbcluster-dbclusterrole-rolearn
	RoleArn string `json:"RoleArn,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *DBCluster_DBClusterRole) AWSCloudFormationType() string {
	return "AWS::RDS::DBCluster.DBClusterRole"
}
