package cloudformation

// AWSSESTemplate_Template AWS CloudFormation Resource (AWS::SES::Template.Template)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ses-template-template.html
type AWSSESTemplate_Template struct {

	// HtmlPart AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ses-template-template.html#cfn-ses-template-template-htmlpart
	HtmlPart string `json:"HtmlPart,omitempty"`

	// SubjectPart AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ses-template-template.html#cfn-ses-template-template-subjectpart
	SubjectPart string `json:"SubjectPart,omitempty"`

	// TemplateName AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ses-template-template.html#cfn-ses-template-template-templatename
	TemplateName string `json:"TemplateName,omitempty"`

	// TextPart AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ses-template-template.html#cfn-ses-template-template-textpart
	TextPart string `json:"TextPart,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSSESTemplate_Template) AWSCloudFormationType() string {
	return "AWS::SES::Template.Template"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSSESTemplate_Template) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
