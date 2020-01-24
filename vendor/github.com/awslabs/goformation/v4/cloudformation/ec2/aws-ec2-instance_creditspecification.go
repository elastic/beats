package ec2

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Instance_CreditSpecification AWS CloudFormation Resource (AWS::EC2::Instance.CreditSpecification)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-instance-creditspecification.html
type Instance_CreditSpecification struct {

	// CPUCredits AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-instance-creditspecification.html#cfn-ec2-instance-creditspecification-cpucredits
	CPUCredits string `json:"CPUCredits,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Instance_CreditSpecification) AWSCloudFormationType() string {
	return "AWS::EC2::Instance.CreditSpecification"
}
