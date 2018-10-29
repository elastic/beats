package cloudformation

// AWSBatchJobDefinition_Volumes AWS CloudFormation Resource (AWS::Batch::JobDefinition.Volumes)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-batch-jobdefinition-volumes.html
type AWSBatchJobDefinition_Volumes struct {

	// Host AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-batch-jobdefinition-volumes.html#cfn-batch-jobdefinition-volumes-host
	Host *AWSBatchJobDefinition_VolumesHost `json:"Host,omitempty"`

	// Name AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-batch-jobdefinition-volumes.html#cfn-batch-jobdefinition-volumes-name
	Name string `json:"Name,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSBatchJobDefinition_Volumes) AWSCloudFormationType() string {
	return "AWS::Batch::JobDefinition.Volumes"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSBatchJobDefinition_Volumes) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
