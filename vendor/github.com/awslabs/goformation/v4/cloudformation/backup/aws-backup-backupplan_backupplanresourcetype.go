package backup

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// BackupPlan_BackupPlanResourceType AWS CloudFormation Resource (AWS::Backup::BackupPlan.BackupPlanResourceType)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-backup-backupplan-backupplanresourcetype.html
type BackupPlan_BackupPlanResourceType struct {

	// BackupPlanName AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-backup-backupplan-backupplanresourcetype.html#cfn-backup-backupplan-backupplanresourcetype-backupplanname
	BackupPlanName string `json:"BackupPlanName,omitempty"`

	// BackupPlanRule AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-backup-backupplan-backupplanresourcetype.html#cfn-backup-backupplan-backupplanresourcetype-backupplanrule
	BackupPlanRule []BackupPlan_BackupRuleResourceType `json:"BackupPlanRule,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *BackupPlan_BackupPlanResourceType) AWSCloudFormationType() string {
	return "AWS::Backup::BackupPlan.BackupPlanResourceType"
}
