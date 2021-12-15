package beater

import (
	"flag"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
	"testing"
)

const clusterName = "EKS-Elastic-agent-demo"

func TestAwsKubeFetcherGetECRInformation(t *testing.T) {

	kubeConfig := GetConfigPath()
	fetcher := NewAwsKubeFetcherFetcher(kubeConfig, clusterName)
	awsFetcher := fetcher.(*AwsKubeFetcher)
	results, err := awsFetcher.GetECRInformation()
	assert.Nil(t, err, "failed to fetch image scanning: %v", err)
	assert.NotEmpty(t, results)
}

func TestAwsKubeFetcherGetClusterInfo(t *testing.T) {

	kubeConfig := GetConfigPath()
	fetcher := NewAwsKubeFetcherFetcher(kubeConfig, clusterName)
	awsFetcher := fetcher.(*AwsKubeFetcher)
	results, err := awsFetcher.GetClusterInfo()
	assert.Nil(t, err, "failed to get cluster info: %v", err)
	assert.NotEmpty(t, results)
}

func TestAwsKubeFetcherGetLoadBalancerInformation(t *testing.T) {

	kubeConfig := GetConfigPath()
	fetcher := NewAwsKubeFetcherFetcher(kubeConfig, clusterName)
	awsFetcher := fetcher.(*AwsKubeFetcher)
	results, err := awsFetcher.GetLoadBalancerDescriptions()
	assert.Nil(t, err, "failed to get load balancer info: %v", err)
	assert.NotEmpty(t, results)
}

func TestAwsKubeFetcherGetNodeInformation(t *testing.T) {

	kubeConfig := GetConfigPath()
	fetcher := NewAwsKubeFetcherFetcher(kubeConfig, clusterName)
	awsFetcher := fetcher.(*AwsKubeFetcher)
	results, err := awsFetcher.GetNodeInformation()
	assert.Nil(t, err, "failed to get load balancer info: %v", err)
	assert.NotEmpty(t, results)
}


func GetConfigPath() string {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	return *kubeconfig
}
