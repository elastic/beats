package beater

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/elastic/beats/v7/libbeat/common/kubernetes"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
)

const (
	PodNameEnvar           = "POD_NAME"
	DefaultLeaderLeaseName = "elastic-agent-cluster-leader"
)

type LeaseInfo struct {
	ctx    context.Context
	client k8s.Interface
}

func NewLeaseInfo(ctx context.Context) (*LeaseInfo, error) {
	c, err := kubernetes.GetKubernetesClient("", kubernetes.KubeClientOptions{})
	if err != nil {
		return nil, err
	}

	return &LeaseInfo{ctx, c}, nil
}

func (l *LeaseInfo) IsLeader() (bool, error) {
	leases, err := l.client.CoordinationV1().Leases(kubeSystemNamespace).List(l.ctx, v1.ListOptions{})
	if err != nil {
		return false, err
	}

	for _, lease := range leases.Items {
		if lease.Name == DefaultLeaderLeaseName {
			podid := lastPart(*lease.Spec.HolderIdentity)

			if podid == l.currentPodID() {
				return true, nil
			}

			return false, nil
		}
	}

	return false, fmt.Errorf("could not find lease %v in Kube leases", DefaultLeaderLeaseName)
}

func (l *LeaseInfo) currentPodID() string {
	pod := os.Getenv(PodNameEnvar)

	return lastPart(pod)
}

func lastPart(s string) string {
	parts := strings.Split(s, "-")
	if len(parts) == 0 {
		return ""
	}

	return parts[len(parts)-1]
}
