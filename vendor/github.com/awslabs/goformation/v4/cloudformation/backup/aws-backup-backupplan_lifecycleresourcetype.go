package backup

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// BackupPlan_LifecycleResourceType AWS CloudFormation Resource (AWS::Backup::BackupPlan.LifecycleResourceType)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-backup-backupplan-lifecycleresourcetype.html
type BackupPlan_LifecycleResourceType struct {

	// DeleteAfterDays AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-backup-backupplan-lifecycleresourcetype.html#cfn-backup-backupplan-lifecycleresourcetype-deleteafterdays
	DeleteAfterDays float64 `json:"DeleteAfterDays,omitempty"`

	// MoveToColdStorageAfterDays AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-backup-backupplan-lifecycleresourcetype.html#cfn-backup-backupplan-lifecycleresourcetype-movetocoldstorageafterdays
	MoveToColdStorageAfterDays float64 `json:"MoveToColdStorageAfterDays,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *BackupPlan_LifecycleResourceType) AWSCloudFormationType() string {
	return "AWS::Backup::BackupPlan.LifecycleResourceType"
}
