package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSServiceCatalogPortfolio AWS CloudFormation Resource (AWS::ServiceCatalog::Portfolio)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-servicecatalog-portfolio.html
type AWSServiceCatalogPortfolio struct {

	// AcceptLanguage AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-servicecatalog-portfolio.html#cfn-servicecatalog-portfolio-acceptlanguage
	AcceptLanguage string `json:"AcceptLanguage,omitempty"`

	// Description AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-servicecatalog-portfolio.html#cfn-servicecatalog-portfolio-description
	Description string `json:"Description,omitempty"`

	// DisplayName AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-servicecatalog-portfolio.html#cfn-servicecatalog-portfolio-displayname
	DisplayName string `json:"DisplayName,omitempty"`

	// ProviderName AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-servicecatalog-portfolio.html#cfn-servicecatalog-portfolio-providername
	ProviderName string `json:"ProviderName,omitempty"`

	// Tags AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-servicecatalog-portfolio.html#cfn-servicecatalog-portfolio-tags
	Tags []Tag `json:"Tags,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSServiceCatalogPortfolio) AWSCloudFormationType() string {
	return "AWS::ServiceCatalog::Portfolio"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSServiceCatalogPortfolio) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSServiceCatalogPortfolio) MarshalJSON() ([]byte, error) {
	type Properties AWSServiceCatalogPortfolio
	return json.Marshal(&struct {
		Type           string
		Properties     Properties
		DeletionPolicy DeletionPolicy `json:"DeletionPolicy,omitempty"`
	}{
		Type:           r.AWSCloudFormationType(),
		Properties:     (Properties)(r),
		DeletionPolicy: r._deletionPolicy,
	})
}

// UnmarshalJSON is a custom JSON unmarshalling hook that strips the outer
// AWS CloudFormation resource object, and just keeps the 'Properties' field.
func (r *AWSServiceCatalogPortfolio) UnmarshalJSON(b []byte) error {
	type Properties AWSServiceCatalogPortfolio
	res := &struct {
		Type       string
		Properties *Properties
	}{}
	if err := json.Unmarshal(b, &res); err != nil {
		fmt.Printf("ERROR: %s\n", err)
		return err
	}

	// If the resource has no Properties set, it could be nil
	if res.Properties != nil {
		*r = AWSServiceCatalogPortfolio(*res.Properties)
	}

	return nil
}

// GetAllAWSServiceCatalogPortfolioResources retrieves all AWSServiceCatalogPortfolio items from an AWS CloudFormation template
func (t *Template) GetAllAWSServiceCatalogPortfolioResources() map[string]AWSServiceCatalogPortfolio {
	results := map[string]AWSServiceCatalogPortfolio{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSServiceCatalogPortfolio:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::ServiceCatalog::Portfolio" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSServiceCatalogPortfolio
						if err := json.Unmarshal(b, &result); err == nil {
							results[name] = result
						}
					}
				}
			}
		}
	}
	return results
}

// GetAWSServiceCatalogPortfolioWithName retrieves all AWSServiceCatalogPortfolio items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSServiceCatalogPortfolioWithName(name string) (AWSServiceCatalogPortfolio, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSServiceCatalogPortfolio:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::ServiceCatalog::Portfolio" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSServiceCatalogPortfolio
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSServiceCatalogPortfolio{}, errors.New("resource not found")
}
