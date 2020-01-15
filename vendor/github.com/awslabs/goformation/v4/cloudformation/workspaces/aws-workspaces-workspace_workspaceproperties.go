package workspaces

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Workspace_WorkspaceProperties AWS CloudFormation Resource (AWS::WorkSpaces::Workspace.WorkspaceProperties)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-workspaces-workspace-workspaceproperties.html
type Workspace_WorkspaceProperties struct {

	// ComputeTypeName AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-workspaces-workspace-workspaceproperties.html#cfn-workspaces-workspace-workspaceproperties-computetypename
	ComputeTypeName string `json:"ComputeTypeName,omitempty"`

	// RootVolumeSizeGib AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-workspaces-workspace-workspaceproperties.html#cfn-workspaces-workspace-workspaceproperties-rootvolumesizegib
	RootVolumeSizeGib int `json:"RootVolumeSizeGib,omitempty"`

	// RunningMode AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-workspaces-workspace-workspaceproperties.html#cfn-workspaces-workspace-workspaceproperties-runningmode
	RunningMode string `json:"RunningMode,omitempty"`

	// RunningModeAutoStopTimeoutInMinutes AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-workspaces-workspace-workspaceproperties.html#cfn-workspaces-workspace-workspaceproperties-runningmodeautostoptimeoutinminutes
	RunningModeAutoStopTimeoutInMinutes int `json:"RunningModeAutoStopTimeoutInMinutes,omitempty"`

	// UserVolumeSizeGib AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-workspaces-workspace-workspaceproperties.html#cfn-workspaces-workspace-workspaceproperties-uservolumesizegib
	UserVolumeSizeGib int `json:"UserVolumeSizeGib,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Workspace_WorkspaceProperties) AWSCloudFormationType() string {
	return "AWS::WorkSpaces::Workspace.WorkspaceProperties"
}
