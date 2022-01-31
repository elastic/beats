package fetchers

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/elastic/beats/v7/libbeat/logp"
)

type EKSProvider struct {
	client *eks.Client
}

func NewEksProvider(cfg aws.Config) *EKSProvider {
	svc := eks.New(cfg)
	return &EKSProvider{
		client: svc,
	}
}

func (provider EKSProvider) DescribeCluster(ctx context.Context, clusterName string) (*eks.DescribeClusterResponse, error) {
	input := &eks.DescribeClusterInput{
		Name: &clusterName,
	}
	req := provider.client.DescribeClusterRequest(input)
	response, err := req.Send(ctx)
	if err != nil {
		logp.Err("Failed to describe cluster %s from eks , error - %+v", clusterName, err)
		return nil, err
	}

	return response, err
}
