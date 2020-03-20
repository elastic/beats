package msk

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Cluster_StorageInfo AWS CloudFormation Resource (AWS::MSK::Cluster.StorageInfo)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-msk-cluster-storageinfo.html
type Cluster_StorageInfo struct {

	// EBSStorageInfo AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-msk-cluster-storageinfo.html#cfn-msk-cluster-storageinfo-ebsstorageinfo
	EBSStorageInfo *Cluster_EBSStorageInfo `json:"EBSStorageInfo,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Cluster_StorageInfo) AWSCloudFormationType() string {
	return "AWS::MSK::Cluster.StorageInfo"
}
