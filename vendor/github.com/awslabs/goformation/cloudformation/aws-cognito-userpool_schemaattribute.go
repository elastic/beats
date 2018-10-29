package cloudformation

// AWSCognitoUserPool_SchemaAttribute AWS CloudFormation Resource (AWS::Cognito::UserPool.SchemaAttribute)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cognito-userpool-schemaattribute.html
type AWSCognitoUserPool_SchemaAttribute struct {

	// AttributeDataType AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cognito-userpool-schemaattribute.html#cfn-cognito-userpool-schemaattribute-attributedatatype
	AttributeDataType string `json:"AttributeDataType,omitempty"`

	// DeveloperOnlyAttribute AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cognito-userpool-schemaattribute.html#cfn-cognito-userpool-schemaattribute-developeronlyattribute
	DeveloperOnlyAttribute bool `json:"DeveloperOnlyAttribute,omitempty"`

	// Mutable AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cognito-userpool-schemaattribute.html#cfn-cognito-userpool-schemaattribute-mutable
	Mutable bool `json:"Mutable,omitempty"`

	// Name AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cognito-userpool-schemaattribute.html#cfn-cognito-userpool-schemaattribute-name
	Name string `json:"Name,omitempty"`

	// NumberAttributeConstraints AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cognito-userpool-schemaattribute.html#cfn-cognito-userpool-schemaattribute-numberattributeconstraints
	NumberAttributeConstraints *AWSCognitoUserPool_NumberAttributeConstraints `json:"NumberAttributeConstraints,omitempty"`

	// Required AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cognito-userpool-schemaattribute.html#cfn-cognito-userpool-schemaattribute-required
	Required bool `json:"Required,omitempty"`

	// StringAttributeConstraints AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cognito-userpool-schemaattribute.html#cfn-cognito-userpool-schemaattribute-stringattributeconstraints
	StringAttributeConstraints *AWSCognitoUserPool_StringAttributeConstraints `json:"StringAttributeConstraints,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSCognitoUserPool_SchemaAttribute) AWSCloudFormationType() string {
	return "AWS::Cognito::UserPool.SchemaAttribute"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSCognitoUserPool_SchemaAttribute) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
