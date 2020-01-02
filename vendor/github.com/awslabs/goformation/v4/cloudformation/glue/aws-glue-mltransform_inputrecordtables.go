package glue

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// MLTransform_InputRecordTables AWS CloudFormation Resource (AWS::Glue::MLTransform.InputRecordTables)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-mltransform-inputrecordtables.html
type MLTransform_InputRecordTables struct {

	// GlueTables AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-mltransform-inputrecordtables.html#cfn-glue-mltransform-inputrecordtables-gluetables
	GlueTables []MLTransform_GlueTables `json:"GlueTables,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *MLTransform_InputRecordTables) AWSCloudFormationType() string {
	return "AWS::Glue::MLTransform.InputRecordTables"
}
