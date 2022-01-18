// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

import "time"

// Todo add consts as configurable vars from integration package - add datastream struct
const DefaultNamespace = "default"

const datastreamPrefix = "logs-k8s_cis.result-"

type Config struct {
	KubeConfig string        `config:"kube_config"`
	Period     time.Duration `config:"period"`
	Files      []string      `config:"files"`
}

var DefaultConfig = Config{
	Period: 10 * time.Second,
	Files: []string{
		"/hostfs/etc/kubernetes/scheduler.conf",
		"/hostfs/etc/kubernetes/controller-manager.conf",
		"/hostfs/etc/kubernetes/admin.conf",
		"/hostfs/etc/kubernetes/kubelet.conf",
		"/hostfs/etc/kubernetes/manifests/etcd.yaml",
		"/hostfs/etc/kubernetes/manifests/kube-apiserver.yaml",
		"/hostfs/etc/kubernetes/manifests/kube-controller-manager.yaml",
		"/hostfs/etc/kubernetes/manifests/kube-scheduler.yaml",
		"/hostfs/etc/systemd/system/kubelet.service.d/10-kubeadm.conf",
		"/hostfs/var/lib/kubelet/config.yaml",
		"/hostfs/var/lib/etcd/**",
		"/hostfs/etc/kubernetes/pki/**",
	},

}

// function to generate the datastream value
func Datastream(namespace string) string {
	if namespace == "" {
		namespace = DefaultNamespace
	}
	return datastreamPrefix + namespace
}
