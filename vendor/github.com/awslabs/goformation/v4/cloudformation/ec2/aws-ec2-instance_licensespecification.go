package ec2

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Instance_LicenseSpecification AWS CloudFormation Resource (AWS::EC2::Instance.LicenseSpecification)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-instance-licensespecification.html
type Instance_LicenseSpecification struct {

	// LicenseConfigurationArn AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-instance-licensespecification.html#cfn-ec2-instance-licensespecification-licenseconfigurationarn
	LicenseConfigurationArn string `json:"LicenseConfigurationArn,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Instance_LicenseSpecification) AWSCloudFormationType() string {
	return "AWS::EC2::Instance.LicenseSpecification"
}
