package fetchers

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/elastic/beats/v7/libbeat/logp"
)

type IAMProvider struct {
	client *iam.Client
}

func NewIAMProvider(cfg aws.Config) *IAMProvider {
	svc := iam.New(cfg)
	return &IAMProvider{
		client: svc,
	}
}

func (provider IAMProvider) GetIAMRolePermissions(ctx context.Context, roleName string) (interface{}, error) {
	results := make([]interface{}, 0)
	policiesIdentifiers, err := provider.getAllRolePolicies(ctx, roleName)
	if err != nil {
		logp.Err("Failed to list role %s policies - %+v", roleName, err)
		return nil, err
	}

	for _, policyId := range policiesIdentifiers {
		input := &iam.GetRolePolicyInput{
			PolicyName: policyId.PolicyName,
			RoleName:   &roleName,
		}
		req := provider.client.GetRolePolicyRequest(input)
		policy, err := req.Send(ctx)
		if err != nil {
			logp.Err("Failed to get policy %s - %+v", *policyId.PolicyName, err)
			continue
		}
		results = append(results, policy)
	}

	return results, nil
}

func (provider IAMProvider) getAllRolePolicies(ctx context.Context, roleName string) ([]iam.AttachedPolicy, error) {
	input := &iam.ListAttachedRolePoliciesInput{
		RoleName: &roleName,
	}
	req := provider.client.ListAttachedRolePoliciesRequest(input)
	allPolicies, err := req.Send(ctx)
	if err != nil {
		logp.Err("Failed to list role %s policies - %+v", roleName, err)
		return nil, err
	}

	return allPolicies.AttachedPolicies, err
}
