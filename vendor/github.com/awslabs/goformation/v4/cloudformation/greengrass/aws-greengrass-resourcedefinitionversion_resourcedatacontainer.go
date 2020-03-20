package greengrass

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// ResourceDefinitionVersion_ResourceDataContainer AWS CloudFormation Resource (AWS::Greengrass::ResourceDefinitionVersion.ResourceDataContainer)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-resourcedefinitionversion-resourcedatacontainer.html
type ResourceDefinitionVersion_ResourceDataContainer struct {

	// LocalDeviceResourceData AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-resourcedefinitionversion-resourcedatacontainer.html#cfn-greengrass-resourcedefinitionversion-resourcedatacontainer-localdeviceresourcedata
	LocalDeviceResourceData *ResourceDefinitionVersion_LocalDeviceResourceData `json:"LocalDeviceResourceData,omitempty"`

	// LocalVolumeResourceData AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-resourcedefinitionversion-resourcedatacontainer.html#cfn-greengrass-resourcedefinitionversion-resourcedatacontainer-localvolumeresourcedata
	LocalVolumeResourceData *ResourceDefinitionVersion_LocalVolumeResourceData `json:"LocalVolumeResourceData,omitempty"`

	// S3MachineLearningModelResourceData AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-resourcedefinitionversion-resourcedatacontainer.html#cfn-greengrass-resourcedefinitionversion-resourcedatacontainer-s3machinelearningmodelresourcedata
	S3MachineLearningModelResourceData *ResourceDefinitionVersion_S3MachineLearningModelResourceData `json:"S3MachineLearningModelResourceData,omitempty"`

	// SageMakerMachineLearningModelResourceData AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-resourcedefinitionversion-resourcedatacontainer.html#cfn-greengrass-resourcedefinitionversion-resourcedatacontainer-sagemakermachinelearningmodelresourcedata
	SageMakerMachineLearningModelResourceData *ResourceDefinitionVersion_SageMakerMachineLearningModelResourceData `json:"SageMakerMachineLearningModelResourceData,omitempty"`

	// SecretsManagerSecretResourceData AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-resourcedefinitionversion-resourcedatacontainer.html#cfn-greengrass-resourcedefinitionversion-resourcedatacontainer-secretsmanagersecretresourcedata
	SecretsManagerSecretResourceData *ResourceDefinitionVersion_SecretsManagerSecretResourceData `json:"SecretsManagerSecretResourceData,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *ResourceDefinitionVersion_ResourceDataContainer) AWSCloudFormationType() string {
	return "AWS::Greengrass::ResourceDefinitionVersion.ResourceDataContainer"
}
