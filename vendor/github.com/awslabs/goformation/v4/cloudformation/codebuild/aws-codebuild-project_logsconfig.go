package codebuild

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Project_LogsConfig AWS CloudFormation Resource (AWS::CodeBuild::Project.LogsConfig)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codebuild-project-logsconfig.html
type Project_LogsConfig struct {

	// CloudWatchLogs AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codebuild-project-logsconfig.html#cfn-codebuild-project-logsconfig-cloudwatchlogs
	CloudWatchLogs *Project_CloudWatchLogsConfig `json:"CloudWatchLogs,omitempty"`

	// S3Logs AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codebuild-project-logsconfig.html#cfn-codebuild-project-logsconfig-s3logs
	S3Logs *Project_S3LogsConfig `json:"S3Logs,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Project_LogsConfig) AWSCloudFormationType() string {
	return "AWS::CodeBuild::Project.LogsConfig"
}
