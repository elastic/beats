package transfer

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Server_EndpointDetails AWS CloudFormation Resource (AWS::Transfer::Server.EndpointDetails)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-transfer-server-endpointdetails.html
type Server_EndpointDetails struct {

	// VpcEndpointId AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-transfer-server-endpointdetails.html#cfn-transfer-server-endpointdetails-vpcendpointid
	VpcEndpointId string `json:"VpcEndpointId,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Server_EndpointDetails) AWSCloudFormationType() string {
	return "AWS::Transfer::Server.EndpointDetails"
}
