package emr

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// InstanceFleetConfig_SpotProvisioningSpecification AWS CloudFormation Resource (AWS::EMR::InstanceFleetConfig.SpotProvisioningSpecification)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-instancefleetconfig-spotprovisioningspecification.html
type InstanceFleetConfig_SpotProvisioningSpecification struct {

	// BlockDurationMinutes AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-instancefleetconfig-spotprovisioningspecification.html#cfn-elasticmapreduce-instancefleetconfig-spotprovisioningspecification-blockdurationminutes
	BlockDurationMinutes int `json:"BlockDurationMinutes,omitempty"`

	// TimeoutAction AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-instancefleetconfig-spotprovisioningspecification.html#cfn-elasticmapreduce-instancefleetconfig-spotprovisioningspecification-timeoutaction
	TimeoutAction string `json:"TimeoutAction,omitempty"`

	// TimeoutDurationMinutes AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-instancefleetconfig-spotprovisioningspecification.html#cfn-elasticmapreduce-instancefleetconfig-spotprovisioningspecification-timeoutdurationminutes
	TimeoutDurationMinutes int `json:"TimeoutDurationMinutes"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *InstanceFleetConfig_SpotProvisioningSpecification) AWSCloudFormationType() string {
	return "AWS::EMR::InstanceFleetConfig.SpotProvisioningSpecification"
}
