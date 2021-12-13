package beater

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/elastic/beats/v7/libbeat/common/kubernetes"
	"github.com/elastic/beats/v7/libbeat/logp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	"log"
	"strings"
	"time"
)

type AwsKubeFetcher struct {
	cfg         aws.Config
	ecrFetcher  ECRDataFetcher
	kubeClient  k8s.Interface
	eks         EKSProvider
	clusterName string
	elb         ElbProvider
}

func NewAwsKubeFetcherFetcher(kubeConfigPath string, clusterName string) Fetcher {

	if kubeConfigPath == "" {
		kubeConfigPath = kubernetes.GetKubeConfigEnvironmentVariable()
	}

	kubernetesClient, err := kubernetes.GetKubernetesClient(kubeConfigPath, kubernetes.KubeClientOptions{})
	if err != nil {
		log.Fatal(err)
	}

	cfg, err := config.LoadDefaultConfig(context.TODO())

	ecr := ECRDataFetcher{}
	eks := EKSProvider{}
	elb := ElbProvider{}

	return &AwsKubeFetcher{
		cfg:         cfg,
		ecrFetcher:  ecr,
		kubeClient:  kubernetesClient,
		eks:         eks,
		elb:         elb,
		clusterName: clusterName,
	}
}

func (f AwsKubeFetcher) Fetch() ([]interface{}, error) {

	//Get Images for ECR
	results := make([]interface{}, 0)

	repositories, err := f.GetECRInformation()
	results = append(results, repositories)

	data, err := f.GetClusterInfo()
	results = append(results, data)

	lbData, err := f.GetLoadBalancerInformation()
	results = append(results, lbData)

	return results, err
}

// 2.1.1 Enable audit Logs (Manual)
// 5.3.1 - Ensure Kubernetes Secrets are encrypted using Customer Master Keys (CMKs) managed in AWS KMS (Automated)
// 5.4.1 - Restrict Access to the Control Plane Endpoint (Manual)
// 5.4.2 - Ensure clusters are created with Private Endpoint Enabled and Public Access Disabled (Manual)
func (f AwsKubeFetcher) GetClusterInfo() (*eks.DescribeClusterOutput, error) {

	// https://github.com/kubernetes/client-go/issues/530
	// Probably we will need to ask for the cluster name via user input

	// TODO - Need to use all namespaces with leader election
	ctx2, cancel2 := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel2()

	result, err := f.eks.DescribeCluster(f.cfg, ctx2, f.clusterName)
	if err != nil {
		logp.Err("Failed to get cluster description  - %+v", err)
	}
	return result, err
}

// EKS benchmark 5.1.1 -  Ensure Image Vulnerability Scanning using Amazon ECR image scanning or a third party provider (Manual)
func (f AwsKubeFetcher) GetECRInformation() ([]types.Repository, error) {

	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()

	// TODO - Need to use all namespaces with leader election
	podsList, err := f.kubeClient.CoreV1().Pods("default").List(ctx, metav1.ListOptions{})
	if err != nil {
		logp.Err("Failed to get pods  - %+v", err)
		return nil, err
	}

	repo := make([]string, 0)
	for _, pod := range podsList.Items {

		for _, container := range pod.Spec.Containers {

			// Takes only aws images
			if strings.Contains(container.Image, "amazonaws") {

				// TODO - Have to refactor or to use the scanning results
				repositoryName := strings.Split(container.Image, "/")[1]
				repo = append(repo, repositoryName)
			}
		}
	}

	ctx2, cancel2 := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel2()

	repositories, err := f.ecrFetcher.DescribeAllRepositories(f.cfg, ctx2, repo)
	return repositories, err
}

// EKS benchmark 5.4.5 -  Encrypt traffic to HTTPS load balancers with TLS certificates (Manual)
func (f AwsKubeFetcher) GetLoadBalancerInformation() (*elasticloadbalancing.DescribeLoadBalancersOutput, error) {

	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()

	// TODO - Need to use all namespaces with leader election
	services, err := f.kubeClient.CoreV1().Services("default").List(ctx, metav1.ListOptions{})
	loadBalancers := make([]string, 0)
	for _, service := range services.Items {

		for _, ingress := range service.Status.LoadBalancer.Ingress {
			if strings.Contains(ingress.Hostname, "amazonaws.com") {
				// TODO - Needs to be refactored
				lbName := strings.Split(ingress.Hostname, "-")[0]
				loadBalancers = append(loadBalancers, lbName)
			}
		}
		log.Printf("bla bla %v", service.Name)
	}
	if err != nil {
		logp.Err("Failed to get pods  - %+v", err)
		return nil, err
	}

	ctx2, cancel2 := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel2()

	result, err := f.elb.DescribeLoadBalancer(f.cfg, ctx2, loadBalancers)
	return result, err
}

// EKS benchmark 5.4.3 Ensure clusters are created with Private Nodes (Manual)
func (f AwsKubeFetcher) GetNodeInformation() ([]interface{}, error) {

	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()

	// TODO - leader election
	nodeList, err := f.kubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	nodesInfo := make([]interface{}, 0)
	for _, node := range nodeList.Items {

		nodesInfo = append(nodesInfo, node)
	}
	if err != nil {
		logp.Err("Failed to get pods  - %+v", err)
		return nil, err
	}

	return nodesInfo, err
}

func (f AwsKubeFetcher) Stop() {

}
