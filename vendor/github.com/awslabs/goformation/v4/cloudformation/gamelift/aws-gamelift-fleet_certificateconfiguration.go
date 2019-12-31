package gamelift

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Fleet_CertificateConfiguration AWS CloudFormation Resource (AWS::GameLift::Fleet.CertificateConfiguration)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-gamelift-fleet-certificateconfiguration.html
type Fleet_CertificateConfiguration struct {

	// CertificateType AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-gamelift-fleet-certificateconfiguration.html#cfn-gamelift-fleet-certificateconfiguration-certificatetype
	CertificateType string `json:"CertificateType,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Fleet_CertificateConfiguration) AWSCloudFormationType() string {
	return "AWS::GameLift::Fleet.CertificateConfiguration"
}
