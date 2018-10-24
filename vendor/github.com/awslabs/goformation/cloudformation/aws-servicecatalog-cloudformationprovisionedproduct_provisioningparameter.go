package cloudformation

// AWSServiceCatalogCloudFormationProvisionedProduct_ProvisioningParameter AWS CloudFormation Resource (AWS::ServiceCatalog::CloudFormationProvisionedProduct.ProvisioningParameter)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-servicecatalog-cloudformationprovisionedproduct-provisioningparameter.html
type AWSServiceCatalogCloudFormationProvisionedProduct_ProvisioningParameter struct {

	// Key AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-servicecatalog-cloudformationprovisionedproduct-provisioningparameter.html#cfn-servicecatalog-cloudformationprovisionedproduct-provisioningparameter-key
	Key string `json:"Key,omitempty"`

	// Value AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-servicecatalog-cloudformationprovisionedproduct-provisioningparameter.html#cfn-servicecatalog-cloudformationprovisionedproduct-provisioningparameter-value
	Value string `json:"Value,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSServiceCatalogCloudFormationProvisionedProduct_ProvisioningParameter) AWSCloudFormationType() string {
	return "AWS::ServiceCatalog::CloudFormationProvisionedProduct.ProvisioningParameter"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSServiceCatalogCloudFormationProvisionedProduct_ProvisioningParameter) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
