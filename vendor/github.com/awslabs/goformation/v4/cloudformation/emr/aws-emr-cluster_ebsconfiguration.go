package emr

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Cluster_EbsConfiguration AWS CloudFormation Resource (AWS::EMR::Cluster.EbsConfiguration)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-cluster-ebsconfiguration.html
type Cluster_EbsConfiguration struct {

	// EbsBlockDeviceConfigs AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-cluster-ebsconfiguration.html#cfn-elasticmapreduce-cluster-ebsconfiguration-ebsblockdeviceconfigs
	EbsBlockDeviceConfigs []Cluster_EbsBlockDeviceConfig `json:"EbsBlockDeviceConfigs,omitempty"`

	// EbsOptimized AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-cluster-ebsconfiguration.html#cfn-elasticmapreduce-cluster-ebsconfiguration-ebsoptimized
	EbsOptimized bool `json:"EbsOptimized,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Cluster_EbsConfiguration) AWSCloudFormationType() string {
	return "AWS::EMR::Cluster.EbsConfiguration"
}
