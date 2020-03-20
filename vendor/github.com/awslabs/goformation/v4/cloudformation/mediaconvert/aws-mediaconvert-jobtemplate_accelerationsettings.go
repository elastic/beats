package mediaconvert

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// JobTemplate_AccelerationSettings AWS CloudFormation Resource (AWS::MediaConvert::JobTemplate.AccelerationSettings)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-mediaconvert-jobtemplate-accelerationsettings.html
type JobTemplate_AccelerationSettings struct {

	// Mode AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-mediaconvert-jobtemplate-accelerationsettings.html#cfn-mediaconvert-jobtemplate-accelerationsettings-mode
	Mode string `json:"Mode,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *JobTemplate_AccelerationSettings) AWSCloudFormationType() string {
	return "AWS::MediaConvert::JobTemplate.AccelerationSettings"
}
