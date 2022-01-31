package add_cluster_id

import (
	"context"
	"github.com/elastic/beats/v7/libbeat/common/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ClusterHelper struct {
	clusterId string
}

func newClusterHelper() (*ClusterHelper, error) {
	clusterId, err := getClusterIdFromClient()
	if err != nil {
		return nil, err
	}
	return &ClusterHelper{clusterId: clusterId}, nil
}

func (c ClusterHelper) ClusterId() string {
	return c.clusterId
}

func getClusterIdFromClient() (string, error) {
	client, err := kubernetes.GetKubernetesClient("", kubernetes.KubeClientOptions{})
	if err != nil {
		return "", err
	}
	n, err := client.CoreV1().Namespaces().Get(context.Background(), "kube-system", metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return string(n.ObjectMeta.UID), nil
}
