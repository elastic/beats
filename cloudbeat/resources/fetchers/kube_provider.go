package fetchers

import (
	"github.com/elastic/beats/v7/cloudbeat/resources"
	"github.com/elastic/beats/v7/libbeat/common/kubernetes"
	"github.com/elastic/beats/v7/libbeat/logp"
)

func GetKubeData(watchers []kubernetes.Watcher) []resources.FetcherResult {
	ret := make([]resources.FetcherResult, 0)

	for _, watcher := range watchers {
		rs := watcher.Store().List()

		for _, r := range rs {
			nullifyManagedFields(r)
			resource, ok := r.(kubernetes.Resource)

			if !ok {
				logp.L().Errorf("Bad resource: %#v does not implement kubernetes.Resource", r)
				continue
			}

			err := addTypeInformationToKubeResource(resource)
			if err != nil {
				logp.L().Errorf("Bad resource: %w", err)
				continue
			} // See https://github.com/kubernetes/kubernetes/issues/3030

			ret = append(ret, resources.FetcherResult{
				Type:     KubeAPIType,
				Resource: resource,
			})
		}
	}

	return ret
}

// nullifyManagedFields ManagedFields field contains fields with dot that prevent from elasticsearch to index
// the events.
func nullifyManagedFields(resource interface{}) {
	switch val := resource.(type) {
	case *kubernetes.Pod:
		val.ManagedFields = nil
	case *kubernetes.Secret:
		val.ManagedFields = nil
	case *kubernetes.Role:
		val.ManagedFields = nil
	case *kubernetes.RoleBinding:
		val.ManagedFields = nil
	case *kubernetes.ClusterRole:
		val.ManagedFields = nil
	case *kubernetes.ClusterRoleBinding:
		val.ManagedFields = nil
	case *kubernetes.PodSecurityPolicy:
		val.ManagedFields = nil
	case *kubernetes.NetworkPolicy:
		val.ManagedFields = nil
	}
}
