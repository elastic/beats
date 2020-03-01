package backup

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// BackupVault_NotificationObjectType AWS CloudFormation Resource (AWS::Backup::BackupVault.NotificationObjectType)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-backup-backupvault-notificationobjecttype.html
type BackupVault_NotificationObjectType struct {

	// BackupVaultEvents AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-backup-backupvault-notificationobjecttype.html#cfn-backup-backupvault-notificationobjecttype-backupvaultevents
	BackupVaultEvents []string `json:"BackupVaultEvents,omitempty"`

	// SNSTopicArn AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-backup-backupvault-notificationobjecttype.html#cfn-backup-backupvault-notificationobjecttype-snstopicarn
	SNSTopicArn string `json:"SNSTopicArn,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *BackupVault_NotificationObjectType) AWSCloudFormationType() string {
	return "AWS::Backup::BackupVault.NotificationObjectType"
}
