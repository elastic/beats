package cloudformation

// AWSGlueConnection_ConnectionInput AWS CloudFormation Resource (AWS::Glue::Connection.ConnectionInput)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-connection-connectioninput.html
type AWSGlueConnection_ConnectionInput struct {

	// ConnectionProperties AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-connection-connectioninput.html#cfn-glue-connection-connectioninput-connectionproperties
	ConnectionProperties interface{} `json:"ConnectionProperties,omitempty"`

	// ConnectionType AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-connection-connectioninput.html#cfn-glue-connection-connectioninput-connectiontype
	ConnectionType string `json:"ConnectionType,omitempty"`

	// Description AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-connection-connectioninput.html#cfn-glue-connection-connectioninput-description
	Description string `json:"Description,omitempty"`

	// MatchCriteria AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-connection-connectioninput.html#cfn-glue-connection-connectioninput-matchcriteria
	MatchCriteria []string `json:"MatchCriteria,omitempty"`

	// Name AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-connection-connectioninput.html#cfn-glue-connection-connectioninput-name
	Name string `json:"Name,omitempty"`

	// PhysicalConnectionRequirements AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-connection-connectioninput.html#cfn-glue-connection-connectioninput-physicalconnectionrequirements
	PhysicalConnectionRequirements *AWSGlueConnection_PhysicalConnectionRequirements `json:"PhysicalConnectionRequirements,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSGlueConnection_ConnectionInput) AWSCloudFormationType() string {
	return "AWS::Glue::Connection.ConnectionInput"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSGlueConnection_ConnectionInput) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
