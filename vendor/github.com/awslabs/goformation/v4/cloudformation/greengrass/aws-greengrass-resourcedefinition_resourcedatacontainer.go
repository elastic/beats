package greengrass

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// ResourceDefinition_ResourceDataContainer AWS CloudFormation Resource (AWS::Greengrass::ResourceDefinition.ResourceDataContainer)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-resourcedefinition-resourcedatacontainer.html
type ResourceDefinition_ResourceDataContainer struct {

	// LocalDeviceResourceData AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-resourcedefinition-resourcedatacontainer.html#cfn-greengrass-resourcedefinition-resourcedatacontainer-localdeviceresourcedata
	LocalDeviceResourceData *ResourceDefinition_LocalDeviceResourceData `json:"LocalDeviceResourceData,omitempty"`

	// LocalVolumeResourceData AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-resourcedefinition-resourcedatacontainer.html#cfn-greengrass-resourcedefinition-resourcedatacontainer-localvolumeresourcedata
	LocalVolumeResourceData *ResourceDefinition_LocalVolumeResourceData `json:"LocalVolumeResourceData,omitempty"`

	// S3MachineLearningModelResourceData AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-resourcedefinition-resourcedatacontainer.html#cfn-greengrass-resourcedefinition-resourcedatacontainer-s3machinelearningmodelresourcedata
	S3MachineLearningModelResourceData *ResourceDefinition_S3MachineLearningModelResourceData `json:"S3MachineLearningModelResourceData,omitempty"`

	// SageMakerMachineLearningModelResourceData AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-resourcedefinition-resourcedatacontainer.html#cfn-greengrass-resourcedefinition-resourcedatacontainer-sagemakermachinelearningmodelresourcedata
	SageMakerMachineLearningModelResourceData *ResourceDefinition_SageMakerMachineLearningModelResourceData `json:"SageMakerMachineLearningModelResourceData,omitempty"`

	// SecretsManagerSecretResourceData AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-resourcedefinition-resourcedatacontainer.html#cfn-greengrass-resourcedefinition-resourcedatacontainer-secretsmanagersecretresourcedata
	SecretsManagerSecretResourceData *ResourceDefinition_SecretsManagerSecretResourceData `json:"SecretsManagerSecretResourceData,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *ResourceDefinition_ResourceDataContainer) AWSCloudFormationType() string {
	return "AWS::Greengrass::ResourceDefinition.ResourceDataContainer"
}
