package cloudformation

// AWSEC2VPNConnection_VpnTunnelOptionsSpecification AWS CloudFormation Resource (AWS::EC2::VPNConnection.VpnTunnelOptionsSpecification)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-vpnconnection-vpntunneloptionsspecification.html
type AWSEC2VPNConnection_VpnTunnelOptionsSpecification struct {

	// PreSharedKey AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-vpnconnection-vpntunneloptionsspecification.html#cfn-ec2-vpnconnection-vpntunneloptionsspecification-presharedkey
	PreSharedKey string `json:"PreSharedKey,omitempty"`

	// TunnelInsideCidr AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-vpnconnection-vpntunneloptionsspecification.html#cfn-ec2-vpnconnection-vpntunneloptionsspecification-tunnelinsidecidr
	TunnelInsideCidr string `json:"TunnelInsideCidr,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSEC2VPNConnection_VpnTunnelOptionsSpecification) AWSCloudFormationType() string {
	return "AWS::EC2::VPNConnection.VpnTunnelOptionsSpecification"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSEC2VPNConnection_VpnTunnelOptionsSpecification) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
