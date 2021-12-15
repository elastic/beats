package beater

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/elastic/beats/v7/libbeat/logp"
	"time"
)

type IamProvider struct {
}

func (f IamProvider) GetIamRolePermissions(cfg aws.Config, ctx context.Context, roleName string) (interface{}, error) {

	results := make([]interface{}, 0)
	policiesIdentifiers, err := f.getAllRolePolicies(cfg, ctx, roleName)
	if err != nil {
		logp.Err("Failed to list role %s policies - %+v", roleName, err)
		return nil, err
	}

	svc := iam.New(cfg)
	for _, policyId := range policiesIdentifiers {

		input := &iam.GetRolePolicyInput{
			PolicyName: policyId.PolicyName,
			RoleName:   &roleName,
		}
		req := svc.GetRolePolicyRequest(input)
		policy, err := req.Send(ctx)
		if err != nil {
			logp.Err("Failed to get policy %s - %+v", *policyId.PolicyName, err)
			continue
		}
		results = append(results, policy)
	}

	return results, nil
}

func (f IamProvider) getAllRolePolicies(cfg aws.Config, ctx context.Context, roleName string) ([]iam.AttachedPolicy, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()

	svc := iam.New(cfg)
	input := &iam.ListAttachedRolePoliciesInput{
		RoleName: &roleName,
	}

	req := svc.ListAttachedRolePoliciesRequest(input)
	allPolicies, err := req.Send(ctx)
	if err != nil {
		logp.Err("Failed to list role %s policies - %+v", roleName, err)
		return nil, err
	}

	return allPolicies.AttachedPolicies, err
}
