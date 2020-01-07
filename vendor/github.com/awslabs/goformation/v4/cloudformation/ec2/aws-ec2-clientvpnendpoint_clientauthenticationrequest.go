package ec2

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// ClientVpnEndpoint_ClientAuthenticationRequest AWS CloudFormation Resource (AWS::EC2::ClientVpnEndpoint.ClientAuthenticationRequest)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-clientvpnendpoint-clientauthenticationrequest.html
type ClientVpnEndpoint_ClientAuthenticationRequest struct {

	// ActiveDirectory AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-clientvpnendpoint-clientauthenticationrequest.html#cfn-ec2-clientvpnendpoint-clientauthenticationrequest-activedirectory
	ActiveDirectory *ClientVpnEndpoint_DirectoryServiceAuthenticationRequest `json:"ActiveDirectory,omitempty"`

	// MutualAuthentication AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-clientvpnendpoint-clientauthenticationrequest.html#cfn-ec2-clientvpnendpoint-clientauthenticationrequest-mutualauthentication
	MutualAuthentication *ClientVpnEndpoint_CertificateAuthenticationRequest `json:"MutualAuthentication,omitempty"`

	// Type AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-clientvpnendpoint-clientauthenticationrequest.html#cfn-ec2-clientvpnendpoint-clientauthenticationrequest-type
	Type string `json:"Type,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *ClientVpnEndpoint_ClientAuthenticationRequest) AWSCloudFormationType() string {
	return "AWS::EC2::ClientVpnEndpoint.ClientAuthenticationRequest"
}
