package ssm

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// MaintenanceWindowTask_LoggingInfo AWS CloudFormation Resource (AWS::SSM::MaintenanceWindowTask.LoggingInfo)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-maintenancewindowtask-logginginfo.html
type MaintenanceWindowTask_LoggingInfo struct {

	// Region AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-maintenancewindowtask-logginginfo.html#cfn-ssm-maintenancewindowtask-logginginfo-region
	Region string `json:"Region,omitempty"`

	// S3Bucket AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-maintenancewindowtask-logginginfo.html#cfn-ssm-maintenancewindowtask-logginginfo-s3bucket
	S3Bucket string `json:"S3Bucket,omitempty"`

	// S3Prefix AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-maintenancewindowtask-logginginfo.html#cfn-ssm-maintenancewindowtask-logginginfo-s3prefix
	S3Prefix string `json:"S3Prefix,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *MaintenanceWindowTask_LoggingInfo) AWSCloudFormationType() string {
	return "AWS::SSM::MaintenanceWindowTask.LoggingInfo"
}
