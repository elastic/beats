package beater

import (
	"context"
	"fmt"
	"k8s.io/client-go/kubernetes"
	"os"
	"strings"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	PodNameEnvar           = "POD_NAME"
	DefaultLeaderLeaseName = "elastic-agent-cluster-leader"
)

type LeaseInfo struct {
	ctx    context.Context
	client kubernetes.Interface
}

func NewLeaseInfo(ctx context.Context, client kubernetes.Interface) (*LeaseInfo, error) {
	return &LeaseInfo{ctx, client}, nil
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
