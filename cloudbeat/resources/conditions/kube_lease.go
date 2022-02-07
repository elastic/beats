package conditions

import (
	"context"
	"fmt"
	"os"
	"strings"

	"k8s.io/client-go/kubernetes"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	PodNameEnvar           = "POD_NAME"
	DefaultLeaderLeaseName = "elastic-agent-cluster-leader"
)

type leaseProvider struct {
	ctx    context.Context
	client kubernetes.Interface
}

func NewLeaderLeaseProvider(ctx context.Context, client kubernetes.Interface) LeaderLeaseProvider {
	return &leaseProvider{ctx, client}
}

func (l *leaseProvider) IsLeader() (bool, error) {
	leases, err := l.client.CoordinationV1().Leases("kube-system").List(l.ctx, v1.ListOptions{})
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

func (l *leaseProvider) currentPodID() string {
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
