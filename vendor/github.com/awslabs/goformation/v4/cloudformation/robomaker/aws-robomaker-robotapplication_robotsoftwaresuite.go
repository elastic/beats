package robomaker

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// RobotApplication_RobotSoftwareSuite AWS CloudFormation Resource (AWS::RoboMaker::RobotApplication.RobotSoftwareSuite)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-robomaker-robotapplication-robotsoftwaresuite.html
type RobotApplication_RobotSoftwareSuite struct {

	// Name AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-robomaker-robotapplication-robotsoftwaresuite.html#cfn-robomaker-robotapplication-robotsoftwaresuite-name
	Name string `json:"Name,omitempty"`

	// Version AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-robomaker-robotapplication-robotsoftwaresuite.html#cfn-robomaker-robotapplication-robotsoftwaresuite-version
	Version string `json:"Version,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *RobotApplication_RobotSoftwareSuite) AWSCloudFormationType() string {
	return "AWS::RoboMaker::RobotApplication.RobotSoftwareSuite"
}
