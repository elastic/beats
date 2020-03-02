package ec2

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// EC2Fleet_FleetLaunchTemplateConfigRequest AWS CloudFormation Resource (AWS::EC2::EC2Fleet.FleetLaunchTemplateConfigRequest)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-ec2fleet-fleetlaunchtemplateconfigrequest.html
type EC2Fleet_FleetLaunchTemplateConfigRequest struct {

	// LaunchTemplateSpecification AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-ec2fleet-fleetlaunchtemplateconfigrequest.html#cfn-ec2-ec2fleet-fleetlaunchtemplateconfigrequest-launchtemplatespecification
	LaunchTemplateSpecification *EC2Fleet_FleetLaunchTemplateSpecificationRequest `json:"LaunchTemplateSpecification,omitempty"`

	// Overrides AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-ec2fleet-fleetlaunchtemplateconfigrequest.html#cfn-ec2-ec2fleet-fleetlaunchtemplateconfigrequest-overrides
	Overrides []EC2Fleet_FleetLaunchTemplateOverridesRequest `json:"Overrides,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *EC2Fleet_FleetLaunchTemplateConfigRequest) AWSCloudFormationType() string {
	return "AWS::EC2::EC2Fleet.FleetLaunchTemplateConfigRequest"
}
