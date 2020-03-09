package emr

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// InstanceFleetConfig_InstanceFleetProvisioningSpecifications AWS CloudFormation Resource (AWS::EMR::InstanceFleetConfig.InstanceFleetProvisioningSpecifications)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-instancefleetconfig-instancefleetprovisioningspecifications.html
type InstanceFleetConfig_InstanceFleetProvisioningSpecifications struct {

	// SpotSpecification AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-instancefleetconfig-instancefleetprovisioningspecifications.html#cfn-elasticmapreduce-instancefleetconfig-instancefleetprovisioningspecifications-spotspecification
	SpotSpecification *InstanceFleetConfig_SpotProvisioningSpecification `json:"SpotSpecification,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *InstanceFleetConfig_InstanceFleetProvisioningSpecifications) AWSCloudFormationType() string {
	return "AWS::EMR::InstanceFleetConfig.InstanceFleetProvisioningSpecifications"
}
