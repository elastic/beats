package beater

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
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
	clusterName string
	cfg         aws.Config
	kubeClient  k8s.Interface
	ecrProvider ECRDataFetcher
	eks         EKSProvider
	elb         ELBProvider
}

func NewAwsKubeFetcherFetcher(kubeconfig string, clusterName string) Fetcher {

	kubernetesClient, err := kubernetes.GetKubernetesClient(kubeconfig, kubernetes.KubeClientOptions{})
	if err != nil {
		log.Fatal(err)
	}

	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		log.Fatal(err)
	}
	ecr := ECRDataFetcher{}
	eks := EKSProvider{}
	elb := ELBProvider{}

	return &AwsKubeFetcher{
		cfg:         cfg,
		ecrProvider: ecr,
		kubeClient:  kubernetesClient,
		eks:         eks,
		elb:         elb,
		clusterName: clusterName,
	}
}

func (f AwsKubeFetcher) Fetch() ([]interface{}, error) {

	results := make([]interface{}, 0)

	repositories, err := f.GetECRInformation()
	results = append(results, repositories)

	data, err := f.GetClusterInfo()
	results = append(results, data)

	lbData, err := f.GetLoadBalancerDescriptions()
	results = append(results, lbData)

	nodeData, err := f.GetNodeInformation()
	results = append(results, nodeData)

	return results, err
}

// 2.1.1 Enable audit Logs (Manual)
// 5.3.1 - Ensure Kubernetes Secrets are encrypted using Customer Master Keys (CMKs) managed in AWS KMS (Automated)
// 5.4.1 - Restrict Access to the Control Plane Endpoint (Manual)
// 5.4.2 - Ensure clusters are created with Private Endpoint Enabled and Public Access Disabled (Manual)
func (f AwsKubeFetcher) GetClusterInfo() (*eks.DescribeClusterResponse, error) {

	// https://github.com/kubernetes/client-go/issues/530
	// Currently we could not auto-detected the cluster name

	// TODO - leader election
	ctx2, cancel2 := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel2()

	result, err := f.eks.DescribeCluster(f.cfg, ctx2, f.clusterName)
	if err != nil {
		logp.Err("Failed to get cluster description  - %+v", err)
	}
	return result, err
}

// EKS benchmark 5.1.1 -  Ensure Image Vulnerability Scanning using Amazon ECR image scanning or a third party provider (Manual)
func (f AwsKubeFetcher) GetECRInformation() ([]ecr.Repository, error) {

	// TODO - Need to use leader election

	// TODO - Currently we do not know how to extract the ECR repository out of the image
	// When we do, we need to scan all the pods and gets their images
	// Otherwise it will get repositories that are not associated with this cluser
	ctx2, cancel := context.WithTimeout(context.TODO(), 150*time.Second)
	defer cancel()

	repositories, err := f.ecrProvider.DescribeRepositories(f.cfg, ctx2, nil)

	return repositories, err
}

// EKS benchmark 5.4.5 -  Encrypt traffic to HTTPS load balancers with TLS certificates (Manual)
func (f AwsKubeFetcher) GetLoadBalancerDescriptions() ([]elasticloadbalancing.LoadBalancerDescription, error) {

	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()

	// TODO - leader election
	services, err := f.kubeClient.CoreV1().Services("").List(ctx, metav1.ListOptions{})
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
		logp.Err("Failed to get all services  - %+v", err)
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
		logp.Err("Failed to get all nodes information  - %+v", err)
		return nil, err
	}

	return nodesInfo, err
}

func (f AwsKubeFetcher) Stop() {

}
