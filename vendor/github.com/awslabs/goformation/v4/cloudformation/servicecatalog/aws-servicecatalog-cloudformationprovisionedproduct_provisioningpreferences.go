package servicecatalog

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// CloudFormationProvisionedProduct_ProvisioningPreferences AWS CloudFormation Resource (AWS::ServiceCatalog::CloudFormationProvisionedProduct.ProvisioningPreferences)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-servicecatalog-cloudformationprovisionedproduct-provisioningpreferences.html
type CloudFormationProvisionedProduct_ProvisioningPreferences struct {

	// StackSetAccounts AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-servicecatalog-cloudformationprovisionedproduct-provisioningpreferences.html#cfn-servicecatalog-cloudformationprovisionedproduct-provisioningpreferences-stacksetaccounts
	StackSetAccounts []string `json:"StackSetAccounts,omitempty"`

	// StackSetFailureToleranceCount AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-servicecatalog-cloudformationprovisionedproduct-provisioningpreferences.html#cfn-servicecatalog-cloudformationprovisionedproduct-provisioningpreferences-stacksetfailuretolerancecount
	StackSetFailureToleranceCount int `json:"StackSetFailureToleranceCount,omitempty"`

	// StackSetFailureTolerancePercentage AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-servicecatalog-cloudformationprovisionedproduct-provisioningpreferences.html#cfn-servicecatalog-cloudformationprovisionedproduct-provisioningpreferences-stacksetfailuretolerancepercentage
	StackSetFailureTolerancePercentage int `json:"StackSetFailureTolerancePercentage,omitempty"`

	// StackSetMaxConcurrencyCount AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-servicecatalog-cloudformationprovisionedproduct-provisioningpreferences.html#cfn-servicecatalog-cloudformationprovisionedproduct-provisioningpreferences-stacksetmaxconcurrencycount
	StackSetMaxConcurrencyCount int `json:"StackSetMaxConcurrencyCount,omitempty"`

	// StackSetMaxConcurrencyPercentage AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-servicecatalog-cloudformationprovisionedproduct-provisioningpreferences.html#cfn-servicecatalog-cloudformationprovisionedproduct-provisioningpreferences-stacksetmaxconcurrencypercentage
	StackSetMaxConcurrencyPercentage int `json:"StackSetMaxConcurrencyPercentage,omitempty"`

	// StackSetOperationType AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-servicecatalog-cloudformationprovisionedproduct-provisioningpreferences.html#cfn-servicecatalog-cloudformationprovisionedproduct-provisioningpreferences-stacksetoperationtype
	StackSetOperationType string `json:"StackSetOperationType,omitempty"`

	// StackSetRegions AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-servicecatalog-cloudformationprovisionedproduct-provisioningpreferences.html#cfn-servicecatalog-cloudformationprovisionedproduct-provisioningpreferences-stacksetregions
	StackSetRegions []string `json:"StackSetRegions,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *CloudFormationProvisionedProduct_ProvisioningPreferences) AWSCloudFormationType() string {
	return "AWS::ServiceCatalog::CloudFormationProvisionedProduct.ProvisioningPreferences"
}
