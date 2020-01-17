package glue

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// MLTransform_TransformParameters AWS CloudFormation Resource (AWS::Glue::MLTransform.TransformParameters)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-mltransform-transformparameters.html
type MLTransform_TransformParameters struct {

	// FindMatchesParameters AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-mltransform-transformparameters.html#cfn-glue-mltransform-transformparameters-findmatchesparameters
	FindMatchesParameters *MLTransform_FindMatchesParameters `json:"FindMatchesParameters,omitempty"`

	// TransformType AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-mltransform-transformparameters.html#cfn-glue-mltransform-transformparameters-transformtype
	TransformType string `json:"TransformType,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *MLTransform_TransformParameters) AWSCloudFormationType() string {
	return "AWS::Glue::MLTransform.TransformParameters"
}
