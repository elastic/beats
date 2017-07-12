package k8s

import (
	"context"
	"fmt"

	apiv1 "github.com/ericchiang/k8s/api/v1"
	appsv1alpha1 "github.com/ericchiang/k8s/apis/apps/v1alpha1"
	appsv1beta1 "github.com/ericchiang/k8s/apis/apps/v1beta1"
	authenticationv1 "github.com/ericchiang/k8s/apis/authentication/v1"
	authenticationv1beta1 "github.com/ericchiang/k8s/apis/authentication/v1beta1"
	authorizationv1 "github.com/ericchiang/k8s/apis/authorization/v1"
	authorizationv1beta1 "github.com/ericchiang/k8s/apis/authorization/v1beta1"
	autoscalingv1 "github.com/ericchiang/k8s/apis/autoscaling/v1"
	autoscalingv2alpha1 "github.com/ericchiang/k8s/apis/autoscaling/v2alpha1"
	batchv1 "github.com/ericchiang/k8s/apis/batch/v1"
	batchv2alpha1 "github.com/ericchiang/k8s/apis/batch/v2alpha1"
	certificatesv1alpha1 "github.com/ericchiang/k8s/apis/certificates/v1alpha1"
	certificatesv1beta1 "github.com/ericchiang/k8s/apis/certificates/v1beta1"
	extensionsv1beta1 "github.com/ericchiang/k8s/apis/extensions/v1beta1"
	imagepolicyv1alpha1 "github.com/ericchiang/k8s/apis/imagepolicy/v1alpha1"
	policyv1alpha1 "github.com/ericchiang/k8s/apis/policy/v1alpha1"
	policyv1beta1 "github.com/ericchiang/k8s/apis/policy/v1beta1"
	rbacv1alpha1 "github.com/ericchiang/k8s/apis/rbac/v1alpha1"
	rbacv1beta1 "github.com/ericchiang/k8s/apis/rbac/v1beta1"
	settingsv1alpha1 "github.com/ericchiang/k8s/apis/settings/v1alpha1"
	storagev1 "github.com/ericchiang/k8s/apis/storage/v1"
	storagev1beta1 "github.com/ericchiang/k8s/apis/storage/v1beta1"
	"github.com/ericchiang/k8s/watch/versioned"
	"github.com/golang/protobuf/proto"
)

// CoreV1 returns a client for interacting with the /v1 API group.
func (c *Client) CoreV1() *CoreV1 {
	return &CoreV1{c}
}

// CoreV1 is a client for interacting with the /v1 API group.
type CoreV1 struct {
	client *Client
}

func (c *CoreV1) CreateBinding(ctx context.Context, obj *apiv1.Binding) (*apiv1.Binding, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("", "v1", ns, "bindings", "")
	resp := new(apiv1.Binding)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) UpdateBinding(ctx context.Context, obj *apiv1.Binding) (*apiv1.Binding, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("", "v1", *md.Namespace, "bindings", *md.Name)
	resp := new(apiv1.Binding)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) DeleteBinding(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("", "v1", namespace, "bindings", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *CoreV1) GetBinding(ctx context.Context, name, namespace string) (*apiv1.Binding, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("", "v1", namespace, "bindings", name)
	resp := new(apiv1.Binding)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) CreateComponentStatus(ctx context.Context, obj *apiv1.ComponentStatus) (*apiv1.ComponentStatus, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("", "v1", ns, "componentstatuses", "")
	resp := new(apiv1.ComponentStatus)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) UpdateComponentStatus(ctx context.Context, obj *apiv1.ComponentStatus) (*apiv1.ComponentStatus, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("", "v1", *md.Namespace, "componentstatuses", *md.Name)
	resp := new(apiv1.ComponentStatus)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) DeleteComponentStatus(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("", "v1", AllNamespaces, "componentstatuses", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *CoreV1) GetComponentStatus(ctx context.Context, name string) (*apiv1.ComponentStatus, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("", "v1", AllNamespaces, "componentstatuses", name)
	resp := new(apiv1.ComponentStatus)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type CoreV1ComponentStatusWatcher struct {
	watcher *watcher
}

func (w *CoreV1ComponentStatusWatcher) Next() (*versioned.Event, *apiv1.ComponentStatus, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(apiv1.ComponentStatus)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *CoreV1ComponentStatusWatcher) Close() error {
	return w.watcher.Close()
}

func (c *CoreV1) WatchComponentStatuses(ctx context.Context, options ...Option) (*CoreV1ComponentStatusWatcher, error) {
	url := c.client.urlFor("", "v1", AllNamespaces, "componentstatuses", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &CoreV1ComponentStatusWatcher{watcher}, nil
}

func (c *CoreV1) ListComponentStatuses(ctx context.Context, options ...Option) (*apiv1.ComponentStatusList, error) {
	url := c.client.urlFor("", "v1", AllNamespaces, "componentstatuses", "", options...)
	resp := new(apiv1.ComponentStatusList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) CreateConfigMap(ctx context.Context, obj *apiv1.ConfigMap) (*apiv1.ConfigMap, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("", "v1", ns, "configmaps", "")
	resp := new(apiv1.ConfigMap)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) UpdateConfigMap(ctx context.Context, obj *apiv1.ConfigMap) (*apiv1.ConfigMap, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("", "v1", *md.Namespace, "configmaps", *md.Name)
	resp := new(apiv1.ConfigMap)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) DeleteConfigMap(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("", "v1", namespace, "configmaps", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *CoreV1) GetConfigMap(ctx context.Context, name, namespace string) (*apiv1.ConfigMap, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("", "v1", namespace, "configmaps", name)
	resp := new(apiv1.ConfigMap)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type CoreV1ConfigMapWatcher struct {
	watcher *watcher
}

func (w *CoreV1ConfigMapWatcher) Next() (*versioned.Event, *apiv1.ConfigMap, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(apiv1.ConfigMap)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *CoreV1ConfigMapWatcher) Close() error {
	return w.watcher.Close()
}

func (c *CoreV1) WatchConfigMaps(ctx context.Context, namespace string, options ...Option) (*CoreV1ConfigMapWatcher, error) {
	url := c.client.urlFor("", "v1", namespace, "configmaps", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &CoreV1ConfigMapWatcher{watcher}, nil
}

func (c *CoreV1) ListConfigMaps(ctx context.Context, namespace string, options ...Option) (*apiv1.ConfigMapList, error) {
	url := c.client.urlFor("", "v1", namespace, "configmaps", "", options...)
	resp := new(apiv1.ConfigMapList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) CreateEndpoints(ctx context.Context, obj *apiv1.Endpoints) (*apiv1.Endpoints, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("", "v1", ns, "endpoints", "")
	resp := new(apiv1.Endpoints)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) UpdateEndpoints(ctx context.Context, obj *apiv1.Endpoints) (*apiv1.Endpoints, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("", "v1", *md.Namespace, "endpoints", *md.Name)
	resp := new(apiv1.Endpoints)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) DeleteEndpoints(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("", "v1", namespace, "endpoints", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *CoreV1) GetEndpoints(ctx context.Context, name, namespace string) (*apiv1.Endpoints, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("", "v1", namespace, "endpoints", name)
	resp := new(apiv1.Endpoints)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type CoreV1EndpointsWatcher struct {
	watcher *watcher
}

func (w *CoreV1EndpointsWatcher) Next() (*versioned.Event, *apiv1.Endpoints, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(apiv1.Endpoints)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *CoreV1EndpointsWatcher) Close() error {
	return w.watcher.Close()
}

func (c *CoreV1) WatchEndpoints(ctx context.Context, namespace string, options ...Option) (*CoreV1EndpointsWatcher, error) {
	url := c.client.urlFor("", "v1", namespace, "endpoints", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &CoreV1EndpointsWatcher{watcher}, nil
}

func (c *CoreV1) ListEndpoints(ctx context.Context, namespace string, options ...Option) (*apiv1.EndpointsList, error) {
	url := c.client.urlFor("", "v1", namespace, "endpoints", "", options...)
	resp := new(apiv1.EndpointsList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) CreateEvent(ctx context.Context, obj *apiv1.Event) (*apiv1.Event, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("", "v1", ns, "events", "")
	resp := new(apiv1.Event)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) UpdateEvent(ctx context.Context, obj *apiv1.Event) (*apiv1.Event, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("", "v1", *md.Namespace, "events", *md.Name)
	resp := new(apiv1.Event)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) DeleteEvent(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("", "v1", namespace, "events", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *CoreV1) GetEvent(ctx context.Context, name, namespace string) (*apiv1.Event, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("", "v1", namespace, "events", name)
	resp := new(apiv1.Event)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type CoreV1EventWatcher struct {
	watcher *watcher
}

func (w *CoreV1EventWatcher) Next() (*versioned.Event, *apiv1.Event, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(apiv1.Event)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *CoreV1EventWatcher) Close() error {
	return w.watcher.Close()
}

func (c *CoreV1) WatchEvents(ctx context.Context, namespace string, options ...Option) (*CoreV1EventWatcher, error) {
	url := c.client.urlFor("", "v1", namespace, "events", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &CoreV1EventWatcher{watcher}, nil
}

func (c *CoreV1) ListEvents(ctx context.Context, namespace string, options ...Option) (*apiv1.EventList, error) {
	url := c.client.urlFor("", "v1", namespace, "events", "", options...)
	resp := new(apiv1.EventList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) CreateLimitRange(ctx context.Context, obj *apiv1.LimitRange) (*apiv1.LimitRange, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("", "v1", ns, "limitranges", "")
	resp := new(apiv1.LimitRange)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) UpdateLimitRange(ctx context.Context, obj *apiv1.LimitRange) (*apiv1.LimitRange, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("", "v1", *md.Namespace, "limitranges", *md.Name)
	resp := new(apiv1.LimitRange)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) DeleteLimitRange(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("", "v1", namespace, "limitranges", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *CoreV1) GetLimitRange(ctx context.Context, name, namespace string) (*apiv1.LimitRange, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("", "v1", namespace, "limitranges", name)
	resp := new(apiv1.LimitRange)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type CoreV1LimitRangeWatcher struct {
	watcher *watcher
}

func (w *CoreV1LimitRangeWatcher) Next() (*versioned.Event, *apiv1.LimitRange, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(apiv1.LimitRange)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *CoreV1LimitRangeWatcher) Close() error {
	return w.watcher.Close()
}

func (c *CoreV1) WatchLimitRanges(ctx context.Context, namespace string, options ...Option) (*CoreV1LimitRangeWatcher, error) {
	url := c.client.urlFor("", "v1", namespace, "limitranges", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &CoreV1LimitRangeWatcher{watcher}, nil
}

func (c *CoreV1) ListLimitRanges(ctx context.Context, namespace string, options ...Option) (*apiv1.LimitRangeList, error) {
	url := c.client.urlFor("", "v1", namespace, "limitranges", "", options...)
	resp := new(apiv1.LimitRangeList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) CreateNamespace(ctx context.Context, obj *apiv1.Namespace) (*apiv1.Namespace, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("", "v1", ns, "namespaces", "")
	resp := new(apiv1.Namespace)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) UpdateNamespace(ctx context.Context, obj *apiv1.Namespace) (*apiv1.Namespace, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("", "v1", *md.Namespace, "namespaces", *md.Name)
	resp := new(apiv1.Namespace)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) DeleteNamespace(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("", "v1", AllNamespaces, "namespaces", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *CoreV1) GetNamespace(ctx context.Context, name string) (*apiv1.Namespace, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("", "v1", AllNamespaces, "namespaces", name)
	resp := new(apiv1.Namespace)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type CoreV1NamespaceWatcher struct {
	watcher *watcher
}

func (w *CoreV1NamespaceWatcher) Next() (*versioned.Event, *apiv1.Namespace, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(apiv1.Namespace)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *CoreV1NamespaceWatcher) Close() error {
	return w.watcher.Close()
}

func (c *CoreV1) WatchNamespaces(ctx context.Context, options ...Option) (*CoreV1NamespaceWatcher, error) {
	url := c.client.urlFor("", "v1", AllNamespaces, "namespaces", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &CoreV1NamespaceWatcher{watcher}, nil
}

func (c *CoreV1) ListNamespaces(ctx context.Context, options ...Option) (*apiv1.NamespaceList, error) {
	url := c.client.urlFor("", "v1", AllNamespaces, "namespaces", "", options...)
	resp := new(apiv1.NamespaceList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) CreateNode(ctx context.Context, obj *apiv1.Node) (*apiv1.Node, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("", "v1", ns, "nodes", "")
	resp := new(apiv1.Node)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) UpdateNode(ctx context.Context, obj *apiv1.Node) (*apiv1.Node, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("", "v1", *md.Namespace, "nodes", *md.Name)
	resp := new(apiv1.Node)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) DeleteNode(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("", "v1", AllNamespaces, "nodes", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *CoreV1) GetNode(ctx context.Context, name string) (*apiv1.Node, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("", "v1", AllNamespaces, "nodes", name)
	resp := new(apiv1.Node)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type CoreV1NodeWatcher struct {
	watcher *watcher
}

func (w *CoreV1NodeWatcher) Next() (*versioned.Event, *apiv1.Node, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(apiv1.Node)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *CoreV1NodeWatcher) Close() error {
	return w.watcher.Close()
}

func (c *CoreV1) WatchNodes(ctx context.Context, options ...Option) (*CoreV1NodeWatcher, error) {
	url := c.client.urlFor("", "v1", AllNamespaces, "nodes", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &CoreV1NodeWatcher{watcher}, nil
}

func (c *CoreV1) ListNodes(ctx context.Context, options ...Option) (*apiv1.NodeList, error) {
	url := c.client.urlFor("", "v1", AllNamespaces, "nodes", "", options...)
	resp := new(apiv1.NodeList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) CreatePersistentVolume(ctx context.Context, obj *apiv1.PersistentVolume) (*apiv1.PersistentVolume, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("", "v1", ns, "persistentvolumes", "")
	resp := new(apiv1.PersistentVolume)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) UpdatePersistentVolume(ctx context.Context, obj *apiv1.PersistentVolume) (*apiv1.PersistentVolume, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("", "v1", *md.Namespace, "persistentvolumes", *md.Name)
	resp := new(apiv1.PersistentVolume)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) DeletePersistentVolume(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("", "v1", AllNamespaces, "persistentvolumes", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *CoreV1) GetPersistentVolume(ctx context.Context, name string) (*apiv1.PersistentVolume, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("", "v1", AllNamespaces, "persistentvolumes", name)
	resp := new(apiv1.PersistentVolume)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type CoreV1PersistentVolumeWatcher struct {
	watcher *watcher
}

func (w *CoreV1PersistentVolumeWatcher) Next() (*versioned.Event, *apiv1.PersistentVolume, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(apiv1.PersistentVolume)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *CoreV1PersistentVolumeWatcher) Close() error {
	return w.watcher.Close()
}

func (c *CoreV1) WatchPersistentVolumes(ctx context.Context, options ...Option) (*CoreV1PersistentVolumeWatcher, error) {
	url := c.client.urlFor("", "v1", AllNamespaces, "persistentvolumes", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &CoreV1PersistentVolumeWatcher{watcher}, nil
}

func (c *CoreV1) ListPersistentVolumes(ctx context.Context, options ...Option) (*apiv1.PersistentVolumeList, error) {
	url := c.client.urlFor("", "v1", AllNamespaces, "persistentvolumes", "", options...)
	resp := new(apiv1.PersistentVolumeList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) CreatePersistentVolumeClaim(ctx context.Context, obj *apiv1.PersistentVolumeClaim) (*apiv1.PersistentVolumeClaim, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("", "v1", ns, "persistentvolumeclaims", "")
	resp := new(apiv1.PersistentVolumeClaim)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) UpdatePersistentVolumeClaim(ctx context.Context, obj *apiv1.PersistentVolumeClaim) (*apiv1.PersistentVolumeClaim, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("", "v1", *md.Namespace, "persistentvolumeclaims", *md.Name)
	resp := new(apiv1.PersistentVolumeClaim)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) DeletePersistentVolumeClaim(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("", "v1", namespace, "persistentvolumeclaims", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *CoreV1) GetPersistentVolumeClaim(ctx context.Context, name, namespace string) (*apiv1.PersistentVolumeClaim, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("", "v1", namespace, "persistentvolumeclaims", name)
	resp := new(apiv1.PersistentVolumeClaim)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type CoreV1PersistentVolumeClaimWatcher struct {
	watcher *watcher
}

func (w *CoreV1PersistentVolumeClaimWatcher) Next() (*versioned.Event, *apiv1.PersistentVolumeClaim, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(apiv1.PersistentVolumeClaim)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *CoreV1PersistentVolumeClaimWatcher) Close() error {
	return w.watcher.Close()
}

func (c *CoreV1) WatchPersistentVolumeClaims(ctx context.Context, namespace string, options ...Option) (*CoreV1PersistentVolumeClaimWatcher, error) {
	url := c.client.urlFor("", "v1", namespace, "persistentvolumeclaims", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &CoreV1PersistentVolumeClaimWatcher{watcher}, nil
}

func (c *CoreV1) ListPersistentVolumeClaims(ctx context.Context, namespace string, options ...Option) (*apiv1.PersistentVolumeClaimList, error) {
	url := c.client.urlFor("", "v1", namespace, "persistentvolumeclaims", "", options...)
	resp := new(apiv1.PersistentVolumeClaimList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) CreatePod(ctx context.Context, obj *apiv1.Pod) (*apiv1.Pod, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("", "v1", ns, "pods", "")
	resp := new(apiv1.Pod)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) UpdatePod(ctx context.Context, obj *apiv1.Pod) (*apiv1.Pod, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("", "v1", *md.Namespace, "pods", *md.Name)
	resp := new(apiv1.Pod)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) DeletePod(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("", "v1", namespace, "pods", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *CoreV1) GetPod(ctx context.Context, name, namespace string) (*apiv1.Pod, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("", "v1", namespace, "pods", name)
	resp := new(apiv1.Pod)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type CoreV1PodWatcher struct {
	watcher *watcher
}

func (w *CoreV1PodWatcher) Next() (*versioned.Event, *apiv1.Pod, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(apiv1.Pod)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *CoreV1PodWatcher) Close() error {
	return w.watcher.Close()
}

func (c *CoreV1) WatchPods(ctx context.Context, namespace string, options ...Option) (*CoreV1PodWatcher, error) {
	url := c.client.urlFor("", "v1", namespace, "pods", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &CoreV1PodWatcher{watcher}, nil
}

func (c *CoreV1) ListPods(ctx context.Context, namespace string, options ...Option) (*apiv1.PodList, error) {
	url := c.client.urlFor("", "v1", namespace, "pods", "", options...)
	resp := new(apiv1.PodList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) CreatePodStatusResult(ctx context.Context, obj *apiv1.PodStatusResult) (*apiv1.PodStatusResult, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("", "v1", ns, "podstatusresults", "")
	resp := new(apiv1.PodStatusResult)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) UpdatePodStatusResult(ctx context.Context, obj *apiv1.PodStatusResult) (*apiv1.PodStatusResult, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("", "v1", *md.Namespace, "podstatusresults", *md.Name)
	resp := new(apiv1.PodStatusResult)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) DeletePodStatusResult(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("", "v1", namespace, "podstatusresults", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *CoreV1) GetPodStatusResult(ctx context.Context, name, namespace string) (*apiv1.PodStatusResult, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("", "v1", namespace, "podstatusresults", name)
	resp := new(apiv1.PodStatusResult)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) CreatePodTemplate(ctx context.Context, obj *apiv1.PodTemplate) (*apiv1.PodTemplate, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("", "v1", ns, "podtemplates", "")
	resp := new(apiv1.PodTemplate)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) UpdatePodTemplate(ctx context.Context, obj *apiv1.PodTemplate) (*apiv1.PodTemplate, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("", "v1", *md.Namespace, "podtemplates", *md.Name)
	resp := new(apiv1.PodTemplate)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) DeletePodTemplate(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("", "v1", namespace, "podtemplates", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *CoreV1) GetPodTemplate(ctx context.Context, name, namespace string) (*apiv1.PodTemplate, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("", "v1", namespace, "podtemplates", name)
	resp := new(apiv1.PodTemplate)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type CoreV1PodTemplateWatcher struct {
	watcher *watcher
}

func (w *CoreV1PodTemplateWatcher) Next() (*versioned.Event, *apiv1.PodTemplate, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(apiv1.PodTemplate)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *CoreV1PodTemplateWatcher) Close() error {
	return w.watcher.Close()
}

func (c *CoreV1) WatchPodTemplates(ctx context.Context, namespace string, options ...Option) (*CoreV1PodTemplateWatcher, error) {
	url := c.client.urlFor("", "v1", namespace, "podtemplates", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &CoreV1PodTemplateWatcher{watcher}, nil
}

func (c *CoreV1) ListPodTemplates(ctx context.Context, namespace string, options ...Option) (*apiv1.PodTemplateList, error) {
	url := c.client.urlFor("", "v1", namespace, "podtemplates", "", options...)
	resp := new(apiv1.PodTemplateList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) CreatePodTemplateSpec(ctx context.Context, obj *apiv1.PodTemplateSpec) (*apiv1.PodTemplateSpec, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("", "v1", ns, "podtemplatespecs", "")
	resp := new(apiv1.PodTemplateSpec)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) UpdatePodTemplateSpec(ctx context.Context, obj *apiv1.PodTemplateSpec) (*apiv1.PodTemplateSpec, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("", "v1", *md.Namespace, "podtemplatespecs", *md.Name)
	resp := new(apiv1.PodTemplateSpec)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) DeletePodTemplateSpec(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("", "v1", namespace, "podtemplatespecs", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *CoreV1) GetPodTemplateSpec(ctx context.Context, name, namespace string) (*apiv1.PodTemplateSpec, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("", "v1", namespace, "podtemplatespecs", name)
	resp := new(apiv1.PodTemplateSpec)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) CreateRangeAllocation(ctx context.Context, obj *apiv1.RangeAllocation) (*apiv1.RangeAllocation, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("", "v1", ns, "rangeallocations", "")
	resp := new(apiv1.RangeAllocation)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) UpdateRangeAllocation(ctx context.Context, obj *apiv1.RangeAllocation) (*apiv1.RangeAllocation, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("", "v1", *md.Namespace, "rangeallocations", *md.Name)
	resp := new(apiv1.RangeAllocation)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) DeleteRangeAllocation(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("", "v1", namespace, "rangeallocations", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *CoreV1) GetRangeAllocation(ctx context.Context, name, namespace string) (*apiv1.RangeAllocation, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("", "v1", namespace, "rangeallocations", name)
	resp := new(apiv1.RangeAllocation)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) CreateReplicationController(ctx context.Context, obj *apiv1.ReplicationController) (*apiv1.ReplicationController, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("", "v1", ns, "replicationcontrollers", "")
	resp := new(apiv1.ReplicationController)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) UpdateReplicationController(ctx context.Context, obj *apiv1.ReplicationController) (*apiv1.ReplicationController, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("", "v1", *md.Namespace, "replicationcontrollers", *md.Name)
	resp := new(apiv1.ReplicationController)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) DeleteReplicationController(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("", "v1", namespace, "replicationcontrollers", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *CoreV1) GetReplicationController(ctx context.Context, name, namespace string) (*apiv1.ReplicationController, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("", "v1", namespace, "replicationcontrollers", name)
	resp := new(apiv1.ReplicationController)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type CoreV1ReplicationControllerWatcher struct {
	watcher *watcher
}

func (w *CoreV1ReplicationControllerWatcher) Next() (*versioned.Event, *apiv1.ReplicationController, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(apiv1.ReplicationController)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *CoreV1ReplicationControllerWatcher) Close() error {
	return w.watcher.Close()
}

func (c *CoreV1) WatchReplicationControllers(ctx context.Context, namespace string, options ...Option) (*CoreV1ReplicationControllerWatcher, error) {
	url := c.client.urlFor("", "v1", namespace, "replicationcontrollers", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &CoreV1ReplicationControllerWatcher{watcher}, nil
}

func (c *CoreV1) ListReplicationControllers(ctx context.Context, namespace string, options ...Option) (*apiv1.ReplicationControllerList, error) {
	url := c.client.urlFor("", "v1", namespace, "replicationcontrollers", "", options...)
	resp := new(apiv1.ReplicationControllerList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) CreateResourceQuota(ctx context.Context, obj *apiv1.ResourceQuota) (*apiv1.ResourceQuota, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("", "v1", ns, "resourcequotas", "")
	resp := new(apiv1.ResourceQuota)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) UpdateResourceQuota(ctx context.Context, obj *apiv1.ResourceQuota) (*apiv1.ResourceQuota, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("", "v1", *md.Namespace, "resourcequotas", *md.Name)
	resp := new(apiv1.ResourceQuota)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) DeleteResourceQuota(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("", "v1", namespace, "resourcequotas", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *CoreV1) GetResourceQuota(ctx context.Context, name, namespace string) (*apiv1.ResourceQuota, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("", "v1", namespace, "resourcequotas", name)
	resp := new(apiv1.ResourceQuota)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type CoreV1ResourceQuotaWatcher struct {
	watcher *watcher
}

func (w *CoreV1ResourceQuotaWatcher) Next() (*versioned.Event, *apiv1.ResourceQuota, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(apiv1.ResourceQuota)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *CoreV1ResourceQuotaWatcher) Close() error {
	return w.watcher.Close()
}

func (c *CoreV1) WatchResourceQuotas(ctx context.Context, namespace string, options ...Option) (*CoreV1ResourceQuotaWatcher, error) {
	url := c.client.urlFor("", "v1", namespace, "resourcequotas", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &CoreV1ResourceQuotaWatcher{watcher}, nil
}

func (c *CoreV1) ListResourceQuotas(ctx context.Context, namespace string, options ...Option) (*apiv1.ResourceQuotaList, error) {
	url := c.client.urlFor("", "v1", namespace, "resourcequotas", "", options...)
	resp := new(apiv1.ResourceQuotaList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) CreateSecret(ctx context.Context, obj *apiv1.Secret) (*apiv1.Secret, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("", "v1", ns, "secrets", "")
	resp := new(apiv1.Secret)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) UpdateSecret(ctx context.Context, obj *apiv1.Secret) (*apiv1.Secret, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("", "v1", *md.Namespace, "secrets", *md.Name)
	resp := new(apiv1.Secret)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) DeleteSecret(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("", "v1", namespace, "secrets", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *CoreV1) GetSecret(ctx context.Context, name, namespace string) (*apiv1.Secret, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("", "v1", namespace, "secrets", name)
	resp := new(apiv1.Secret)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type CoreV1SecretWatcher struct {
	watcher *watcher
}

func (w *CoreV1SecretWatcher) Next() (*versioned.Event, *apiv1.Secret, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(apiv1.Secret)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *CoreV1SecretWatcher) Close() error {
	return w.watcher.Close()
}

func (c *CoreV1) WatchSecrets(ctx context.Context, namespace string, options ...Option) (*CoreV1SecretWatcher, error) {
	url := c.client.urlFor("", "v1", namespace, "secrets", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &CoreV1SecretWatcher{watcher}, nil
}

func (c *CoreV1) ListSecrets(ctx context.Context, namespace string, options ...Option) (*apiv1.SecretList, error) {
	url := c.client.urlFor("", "v1", namespace, "secrets", "", options...)
	resp := new(apiv1.SecretList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) CreateService(ctx context.Context, obj *apiv1.Service) (*apiv1.Service, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("", "v1", ns, "services", "")
	resp := new(apiv1.Service)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) UpdateService(ctx context.Context, obj *apiv1.Service) (*apiv1.Service, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("", "v1", *md.Namespace, "services", *md.Name)
	resp := new(apiv1.Service)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) DeleteService(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("", "v1", namespace, "services", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *CoreV1) GetService(ctx context.Context, name, namespace string) (*apiv1.Service, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("", "v1", namespace, "services", name)
	resp := new(apiv1.Service)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type CoreV1ServiceWatcher struct {
	watcher *watcher
}

func (w *CoreV1ServiceWatcher) Next() (*versioned.Event, *apiv1.Service, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(apiv1.Service)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *CoreV1ServiceWatcher) Close() error {
	return w.watcher.Close()
}

func (c *CoreV1) WatchServices(ctx context.Context, namespace string, options ...Option) (*CoreV1ServiceWatcher, error) {
	url := c.client.urlFor("", "v1", namespace, "services", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &CoreV1ServiceWatcher{watcher}, nil
}

func (c *CoreV1) ListServices(ctx context.Context, namespace string, options ...Option) (*apiv1.ServiceList, error) {
	url := c.client.urlFor("", "v1", namespace, "services", "", options...)
	resp := new(apiv1.ServiceList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) CreateServiceAccount(ctx context.Context, obj *apiv1.ServiceAccount) (*apiv1.ServiceAccount, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("", "v1", ns, "serviceaccounts", "")
	resp := new(apiv1.ServiceAccount)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) UpdateServiceAccount(ctx context.Context, obj *apiv1.ServiceAccount) (*apiv1.ServiceAccount, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("", "v1", *md.Namespace, "serviceaccounts", *md.Name)
	resp := new(apiv1.ServiceAccount)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CoreV1) DeleteServiceAccount(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("", "v1", namespace, "serviceaccounts", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *CoreV1) GetServiceAccount(ctx context.Context, name, namespace string) (*apiv1.ServiceAccount, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("", "v1", namespace, "serviceaccounts", name)
	resp := new(apiv1.ServiceAccount)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type CoreV1ServiceAccountWatcher struct {
	watcher *watcher
}

func (w *CoreV1ServiceAccountWatcher) Next() (*versioned.Event, *apiv1.ServiceAccount, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(apiv1.ServiceAccount)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *CoreV1ServiceAccountWatcher) Close() error {
	return w.watcher.Close()
}

func (c *CoreV1) WatchServiceAccounts(ctx context.Context, namespace string, options ...Option) (*CoreV1ServiceAccountWatcher, error) {
	url := c.client.urlFor("", "v1", namespace, "serviceaccounts", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &CoreV1ServiceAccountWatcher{watcher}, nil
}

func (c *CoreV1) ListServiceAccounts(ctx context.Context, namespace string, options ...Option) (*apiv1.ServiceAccountList, error) {
	url := c.client.urlFor("", "v1", namespace, "serviceaccounts", "", options...)
	resp := new(apiv1.ServiceAccountList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// AppsV1Alpha1 returns a client for interacting with the apps/v1alpha1 API group.
func (c *Client) AppsV1Alpha1() *AppsV1Alpha1 {
	return &AppsV1Alpha1{c}
}

// AppsV1Alpha1 is a client for interacting with the apps/v1alpha1 API group.
type AppsV1Alpha1 struct {
	client *Client
}

func (c *AppsV1Alpha1) CreatePetSet(ctx context.Context, obj *appsv1alpha1.PetSet) (*appsv1alpha1.PetSet, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("apps", "v1alpha1", ns, "petsets", "")
	resp := new(appsv1alpha1.PetSet)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *AppsV1Alpha1) UpdatePetSet(ctx context.Context, obj *appsv1alpha1.PetSet) (*appsv1alpha1.PetSet, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("apps", "v1alpha1", *md.Namespace, "petsets", *md.Name)
	resp := new(appsv1alpha1.PetSet)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *AppsV1Alpha1) DeletePetSet(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("apps", "v1alpha1", namespace, "petsets", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *AppsV1Alpha1) GetPetSet(ctx context.Context, name, namespace string) (*appsv1alpha1.PetSet, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("apps", "v1alpha1", namespace, "petsets", name)
	resp := new(appsv1alpha1.PetSet)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type AppsV1Alpha1PetSetWatcher struct {
	watcher *watcher
}

func (w *AppsV1Alpha1PetSetWatcher) Next() (*versioned.Event, *appsv1alpha1.PetSet, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(appsv1alpha1.PetSet)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *AppsV1Alpha1PetSetWatcher) Close() error {
	return w.watcher.Close()
}

func (c *AppsV1Alpha1) WatchPetSets(ctx context.Context, namespace string, options ...Option) (*AppsV1Alpha1PetSetWatcher, error) {
	url := c.client.urlFor("apps", "v1alpha1", namespace, "petsets", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &AppsV1Alpha1PetSetWatcher{watcher}, nil
}

func (c *AppsV1Alpha1) ListPetSets(ctx context.Context, namespace string, options ...Option) (*appsv1alpha1.PetSetList, error) {
	url := c.client.urlFor("apps", "v1alpha1", namespace, "petsets", "", options...)
	resp := new(appsv1alpha1.PetSetList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// AppsV1Beta1 returns a client for interacting with the apps/v1beta1 API group.
func (c *Client) AppsV1Beta1() *AppsV1Beta1 {
	return &AppsV1Beta1{c}
}

// AppsV1Beta1 is a client for interacting with the apps/v1beta1 API group.
type AppsV1Beta1 struct {
	client *Client
}

func (c *AppsV1Beta1) CreateDeployment(ctx context.Context, obj *appsv1beta1.Deployment) (*appsv1beta1.Deployment, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("apps", "v1beta1", ns, "deployments", "")
	resp := new(appsv1beta1.Deployment)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *AppsV1Beta1) UpdateDeployment(ctx context.Context, obj *appsv1beta1.Deployment) (*appsv1beta1.Deployment, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("apps", "v1beta1", *md.Namespace, "deployments", *md.Name)
	resp := new(appsv1beta1.Deployment)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *AppsV1Beta1) DeleteDeployment(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("apps", "v1beta1", namespace, "deployments", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *AppsV1Beta1) GetDeployment(ctx context.Context, name, namespace string) (*appsv1beta1.Deployment, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("apps", "v1beta1", namespace, "deployments", name)
	resp := new(appsv1beta1.Deployment)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type AppsV1Beta1DeploymentWatcher struct {
	watcher *watcher
}

func (w *AppsV1Beta1DeploymentWatcher) Next() (*versioned.Event, *appsv1beta1.Deployment, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(appsv1beta1.Deployment)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *AppsV1Beta1DeploymentWatcher) Close() error {
	return w.watcher.Close()
}

func (c *AppsV1Beta1) WatchDeployments(ctx context.Context, namespace string, options ...Option) (*AppsV1Beta1DeploymentWatcher, error) {
	url := c.client.urlFor("apps", "v1beta1", namespace, "deployments", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &AppsV1Beta1DeploymentWatcher{watcher}, nil
}

func (c *AppsV1Beta1) ListDeployments(ctx context.Context, namespace string, options ...Option) (*appsv1beta1.DeploymentList, error) {
	url := c.client.urlFor("apps", "v1beta1", namespace, "deployments", "", options...)
	resp := new(appsv1beta1.DeploymentList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *AppsV1Beta1) CreateScale(ctx context.Context, obj *appsv1beta1.Scale) (*appsv1beta1.Scale, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("apps", "v1beta1", ns, "scales", "")
	resp := new(appsv1beta1.Scale)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *AppsV1Beta1) UpdateScale(ctx context.Context, obj *appsv1beta1.Scale) (*appsv1beta1.Scale, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("apps", "v1beta1", *md.Namespace, "scales", *md.Name)
	resp := new(appsv1beta1.Scale)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *AppsV1Beta1) DeleteScale(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("apps", "v1beta1", namespace, "scales", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *AppsV1Beta1) GetScale(ctx context.Context, name, namespace string) (*appsv1beta1.Scale, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("apps", "v1beta1", namespace, "scales", name)
	resp := new(appsv1beta1.Scale)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *AppsV1Beta1) CreateStatefulSet(ctx context.Context, obj *appsv1beta1.StatefulSet) (*appsv1beta1.StatefulSet, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("apps", "v1beta1", ns, "statefulsets", "")
	resp := new(appsv1beta1.StatefulSet)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *AppsV1Beta1) UpdateStatefulSet(ctx context.Context, obj *appsv1beta1.StatefulSet) (*appsv1beta1.StatefulSet, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("apps", "v1beta1", *md.Namespace, "statefulsets", *md.Name)
	resp := new(appsv1beta1.StatefulSet)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *AppsV1Beta1) DeleteStatefulSet(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("apps", "v1beta1", namespace, "statefulsets", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *AppsV1Beta1) GetStatefulSet(ctx context.Context, name, namespace string) (*appsv1beta1.StatefulSet, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("apps", "v1beta1", namespace, "statefulsets", name)
	resp := new(appsv1beta1.StatefulSet)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type AppsV1Beta1StatefulSetWatcher struct {
	watcher *watcher
}

func (w *AppsV1Beta1StatefulSetWatcher) Next() (*versioned.Event, *appsv1beta1.StatefulSet, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(appsv1beta1.StatefulSet)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *AppsV1Beta1StatefulSetWatcher) Close() error {
	return w.watcher.Close()
}

func (c *AppsV1Beta1) WatchStatefulSets(ctx context.Context, namespace string, options ...Option) (*AppsV1Beta1StatefulSetWatcher, error) {
	url := c.client.urlFor("apps", "v1beta1", namespace, "statefulsets", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &AppsV1Beta1StatefulSetWatcher{watcher}, nil
}

func (c *AppsV1Beta1) ListStatefulSets(ctx context.Context, namespace string, options ...Option) (*appsv1beta1.StatefulSetList, error) {
	url := c.client.urlFor("apps", "v1beta1", namespace, "statefulsets", "", options...)
	resp := new(appsv1beta1.StatefulSetList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// AuthenticationV1 returns a client for interacting with the authentication.k8s.io/v1 API group.
func (c *Client) AuthenticationV1() *AuthenticationV1 {
	return &AuthenticationV1{c}
}

// AuthenticationV1 is a client for interacting with the authentication.k8s.io/v1 API group.
type AuthenticationV1 struct {
	client *Client
}

func (c *AuthenticationV1) CreateTokenReview(ctx context.Context, obj *authenticationv1.TokenReview) (*authenticationv1.TokenReview, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("authentication.k8s.io", "v1", ns, "tokenreviews", "")
	resp := new(authenticationv1.TokenReview)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *AuthenticationV1) UpdateTokenReview(ctx context.Context, obj *authenticationv1.TokenReview) (*authenticationv1.TokenReview, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("authentication.k8s.io", "v1", *md.Namespace, "tokenreviews", *md.Name)
	resp := new(authenticationv1.TokenReview)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *AuthenticationV1) DeleteTokenReview(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("authentication.k8s.io", "v1", AllNamespaces, "tokenreviews", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *AuthenticationV1) GetTokenReview(ctx context.Context, name string) (*authenticationv1.TokenReview, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("authentication.k8s.io", "v1", AllNamespaces, "tokenreviews", name)
	resp := new(authenticationv1.TokenReview)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// AuthenticationV1Beta1 returns a client for interacting with the authentication.k8s.io/v1beta1 API group.
func (c *Client) AuthenticationV1Beta1() *AuthenticationV1Beta1 {
	return &AuthenticationV1Beta1{c}
}

// AuthenticationV1Beta1 is a client for interacting with the authentication.k8s.io/v1beta1 API group.
type AuthenticationV1Beta1 struct {
	client *Client
}

func (c *AuthenticationV1Beta1) CreateTokenReview(ctx context.Context, obj *authenticationv1beta1.TokenReview) (*authenticationv1beta1.TokenReview, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("authentication.k8s.io", "v1beta1", ns, "tokenreviews", "")
	resp := new(authenticationv1beta1.TokenReview)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *AuthenticationV1Beta1) UpdateTokenReview(ctx context.Context, obj *authenticationv1beta1.TokenReview) (*authenticationv1beta1.TokenReview, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("authentication.k8s.io", "v1beta1", *md.Namespace, "tokenreviews", *md.Name)
	resp := new(authenticationv1beta1.TokenReview)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *AuthenticationV1Beta1) DeleteTokenReview(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("authentication.k8s.io", "v1beta1", AllNamespaces, "tokenreviews", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *AuthenticationV1Beta1) GetTokenReview(ctx context.Context, name string) (*authenticationv1beta1.TokenReview, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("authentication.k8s.io", "v1beta1", AllNamespaces, "tokenreviews", name)
	resp := new(authenticationv1beta1.TokenReview)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// AuthorizationV1 returns a client for interacting with the authorization.k8s.io/v1 API group.
func (c *Client) AuthorizationV1() *AuthorizationV1 {
	return &AuthorizationV1{c}
}

// AuthorizationV1 is a client for interacting with the authorization.k8s.io/v1 API group.
type AuthorizationV1 struct {
	client *Client
}

func (c *AuthorizationV1) CreateLocalSubjectAccessReview(ctx context.Context, obj *authorizationv1.LocalSubjectAccessReview) (*authorizationv1.LocalSubjectAccessReview, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("authorization.k8s.io", "v1", ns, "localsubjectaccessreviews", "")
	resp := new(authorizationv1.LocalSubjectAccessReview)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *AuthorizationV1) UpdateLocalSubjectAccessReview(ctx context.Context, obj *authorizationv1.LocalSubjectAccessReview) (*authorizationv1.LocalSubjectAccessReview, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("authorization.k8s.io", "v1", *md.Namespace, "localsubjectaccessreviews", *md.Name)
	resp := new(authorizationv1.LocalSubjectAccessReview)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *AuthorizationV1) DeleteLocalSubjectAccessReview(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("authorization.k8s.io", "v1", namespace, "localsubjectaccessreviews", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *AuthorizationV1) GetLocalSubjectAccessReview(ctx context.Context, name, namespace string) (*authorizationv1.LocalSubjectAccessReview, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("authorization.k8s.io", "v1", namespace, "localsubjectaccessreviews", name)
	resp := new(authorizationv1.LocalSubjectAccessReview)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *AuthorizationV1) CreateSelfSubjectAccessReview(ctx context.Context, obj *authorizationv1.SelfSubjectAccessReview) (*authorizationv1.SelfSubjectAccessReview, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("authorization.k8s.io", "v1", ns, "selfsubjectaccessreviews", "")
	resp := new(authorizationv1.SelfSubjectAccessReview)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *AuthorizationV1) UpdateSelfSubjectAccessReview(ctx context.Context, obj *authorizationv1.SelfSubjectAccessReview) (*authorizationv1.SelfSubjectAccessReview, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("authorization.k8s.io", "v1", *md.Namespace, "selfsubjectaccessreviews", *md.Name)
	resp := new(authorizationv1.SelfSubjectAccessReview)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *AuthorizationV1) DeleteSelfSubjectAccessReview(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("authorization.k8s.io", "v1", AllNamespaces, "selfsubjectaccessreviews", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *AuthorizationV1) GetSelfSubjectAccessReview(ctx context.Context, name string) (*authorizationv1.SelfSubjectAccessReview, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("authorization.k8s.io", "v1", AllNamespaces, "selfsubjectaccessreviews", name)
	resp := new(authorizationv1.SelfSubjectAccessReview)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *AuthorizationV1) CreateSubjectAccessReview(ctx context.Context, obj *authorizationv1.SubjectAccessReview) (*authorizationv1.SubjectAccessReview, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("authorization.k8s.io", "v1", ns, "subjectaccessreviews", "")
	resp := new(authorizationv1.SubjectAccessReview)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *AuthorizationV1) UpdateSubjectAccessReview(ctx context.Context, obj *authorizationv1.SubjectAccessReview) (*authorizationv1.SubjectAccessReview, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("authorization.k8s.io", "v1", *md.Namespace, "subjectaccessreviews", *md.Name)
	resp := new(authorizationv1.SubjectAccessReview)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *AuthorizationV1) DeleteSubjectAccessReview(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("authorization.k8s.io", "v1", AllNamespaces, "subjectaccessreviews", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *AuthorizationV1) GetSubjectAccessReview(ctx context.Context, name string) (*authorizationv1.SubjectAccessReview, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("authorization.k8s.io", "v1", AllNamespaces, "subjectaccessreviews", name)
	resp := new(authorizationv1.SubjectAccessReview)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// AuthorizationV1Beta1 returns a client for interacting with the authorization.k8s.io/v1beta1 API group.
func (c *Client) AuthorizationV1Beta1() *AuthorizationV1Beta1 {
	return &AuthorizationV1Beta1{c}
}

// AuthorizationV1Beta1 is a client for interacting with the authorization.k8s.io/v1beta1 API group.
type AuthorizationV1Beta1 struct {
	client *Client
}

func (c *AuthorizationV1Beta1) CreateLocalSubjectAccessReview(ctx context.Context, obj *authorizationv1beta1.LocalSubjectAccessReview) (*authorizationv1beta1.LocalSubjectAccessReview, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("authorization.k8s.io", "v1beta1", ns, "localsubjectaccessreviews", "")
	resp := new(authorizationv1beta1.LocalSubjectAccessReview)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *AuthorizationV1Beta1) UpdateLocalSubjectAccessReview(ctx context.Context, obj *authorizationv1beta1.LocalSubjectAccessReview) (*authorizationv1beta1.LocalSubjectAccessReview, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("authorization.k8s.io", "v1beta1", *md.Namespace, "localsubjectaccessreviews", *md.Name)
	resp := new(authorizationv1beta1.LocalSubjectAccessReview)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *AuthorizationV1Beta1) DeleteLocalSubjectAccessReview(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("authorization.k8s.io", "v1beta1", namespace, "localsubjectaccessreviews", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *AuthorizationV1Beta1) GetLocalSubjectAccessReview(ctx context.Context, name, namespace string) (*authorizationv1beta1.LocalSubjectAccessReview, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("authorization.k8s.io", "v1beta1", namespace, "localsubjectaccessreviews", name)
	resp := new(authorizationv1beta1.LocalSubjectAccessReview)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *AuthorizationV1Beta1) CreateSelfSubjectAccessReview(ctx context.Context, obj *authorizationv1beta1.SelfSubjectAccessReview) (*authorizationv1beta1.SelfSubjectAccessReview, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("authorization.k8s.io", "v1beta1", ns, "selfsubjectaccessreviews", "")
	resp := new(authorizationv1beta1.SelfSubjectAccessReview)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *AuthorizationV1Beta1) UpdateSelfSubjectAccessReview(ctx context.Context, obj *authorizationv1beta1.SelfSubjectAccessReview) (*authorizationv1beta1.SelfSubjectAccessReview, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("authorization.k8s.io", "v1beta1", *md.Namespace, "selfsubjectaccessreviews", *md.Name)
	resp := new(authorizationv1beta1.SelfSubjectAccessReview)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *AuthorizationV1Beta1) DeleteSelfSubjectAccessReview(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("authorization.k8s.io", "v1beta1", AllNamespaces, "selfsubjectaccessreviews", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *AuthorizationV1Beta1) GetSelfSubjectAccessReview(ctx context.Context, name string) (*authorizationv1beta1.SelfSubjectAccessReview, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("authorization.k8s.io", "v1beta1", AllNamespaces, "selfsubjectaccessreviews", name)
	resp := new(authorizationv1beta1.SelfSubjectAccessReview)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *AuthorizationV1Beta1) CreateSubjectAccessReview(ctx context.Context, obj *authorizationv1beta1.SubjectAccessReview) (*authorizationv1beta1.SubjectAccessReview, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("authorization.k8s.io", "v1beta1", ns, "subjectaccessreviews", "")
	resp := new(authorizationv1beta1.SubjectAccessReview)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *AuthorizationV1Beta1) UpdateSubjectAccessReview(ctx context.Context, obj *authorizationv1beta1.SubjectAccessReview) (*authorizationv1beta1.SubjectAccessReview, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("authorization.k8s.io", "v1beta1", *md.Namespace, "subjectaccessreviews", *md.Name)
	resp := new(authorizationv1beta1.SubjectAccessReview)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *AuthorizationV1Beta1) DeleteSubjectAccessReview(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("authorization.k8s.io", "v1beta1", AllNamespaces, "subjectaccessreviews", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *AuthorizationV1Beta1) GetSubjectAccessReview(ctx context.Context, name string) (*authorizationv1beta1.SubjectAccessReview, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("authorization.k8s.io", "v1beta1", AllNamespaces, "subjectaccessreviews", name)
	resp := new(authorizationv1beta1.SubjectAccessReview)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// AutoscalingV1 returns a client for interacting with the autoscaling/v1 API group.
func (c *Client) AutoscalingV1() *AutoscalingV1 {
	return &AutoscalingV1{c}
}

// AutoscalingV1 is a client for interacting with the autoscaling/v1 API group.
type AutoscalingV1 struct {
	client *Client
}

func (c *AutoscalingV1) CreateHorizontalPodAutoscaler(ctx context.Context, obj *autoscalingv1.HorizontalPodAutoscaler) (*autoscalingv1.HorizontalPodAutoscaler, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("autoscaling", "v1", ns, "horizontalpodautoscalers", "")
	resp := new(autoscalingv1.HorizontalPodAutoscaler)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *AutoscalingV1) UpdateHorizontalPodAutoscaler(ctx context.Context, obj *autoscalingv1.HorizontalPodAutoscaler) (*autoscalingv1.HorizontalPodAutoscaler, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("autoscaling", "v1", *md.Namespace, "horizontalpodautoscalers", *md.Name)
	resp := new(autoscalingv1.HorizontalPodAutoscaler)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *AutoscalingV1) DeleteHorizontalPodAutoscaler(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("autoscaling", "v1", namespace, "horizontalpodautoscalers", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *AutoscalingV1) GetHorizontalPodAutoscaler(ctx context.Context, name, namespace string) (*autoscalingv1.HorizontalPodAutoscaler, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("autoscaling", "v1", namespace, "horizontalpodautoscalers", name)
	resp := new(autoscalingv1.HorizontalPodAutoscaler)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type AutoscalingV1HorizontalPodAutoscalerWatcher struct {
	watcher *watcher
}

func (w *AutoscalingV1HorizontalPodAutoscalerWatcher) Next() (*versioned.Event, *autoscalingv1.HorizontalPodAutoscaler, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(autoscalingv1.HorizontalPodAutoscaler)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *AutoscalingV1HorizontalPodAutoscalerWatcher) Close() error {
	return w.watcher.Close()
}

func (c *AutoscalingV1) WatchHorizontalPodAutoscalers(ctx context.Context, namespace string, options ...Option) (*AutoscalingV1HorizontalPodAutoscalerWatcher, error) {
	url := c.client.urlFor("autoscaling", "v1", namespace, "horizontalpodautoscalers", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &AutoscalingV1HorizontalPodAutoscalerWatcher{watcher}, nil
}

func (c *AutoscalingV1) ListHorizontalPodAutoscalers(ctx context.Context, namespace string, options ...Option) (*autoscalingv1.HorizontalPodAutoscalerList, error) {
	url := c.client.urlFor("autoscaling", "v1", namespace, "horizontalpodautoscalers", "", options...)
	resp := new(autoscalingv1.HorizontalPodAutoscalerList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *AutoscalingV1) CreateScale(ctx context.Context, obj *autoscalingv1.Scale) (*autoscalingv1.Scale, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("autoscaling", "v1", ns, "scales", "")
	resp := new(autoscalingv1.Scale)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *AutoscalingV1) UpdateScale(ctx context.Context, obj *autoscalingv1.Scale) (*autoscalingv1.Scale, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("autoscaling", "v1", *md.Namespace, "scales", *md.Name)
	resp := new(autoscalingv1.Scale)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *AutoscalingV1) DeleteScale(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("autoscaling", "v1", namespace, "scales", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *AutoscalingV1) GetScale(ctx context.Context, name, namespace string) (*autoscalingv1.Scale, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("autoscaling", "v1", namespace, "scales", name)
	resp := new(autoscalingv1.Scale)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// AutoscalingV2Alpha1 returns a client for interacting with the autoscaling/v2alpha1 API group.
func (c *Client) AutoscalingV2Alpha1() *AutoscalingV2Alpha1 {
	return &AutoscalingV2Alpha1{c}
}

// AutoscalingV2Alpha1 is a client for interacting with the autoscaling/v2alpha1 API group.
type AutoscalingV2Alpha1 struct {
	client *Client
}

func (c *AutoscalingV2Alpha1) CreateHorizontalPodAutoscaler(ctx context.Context, obj *autoscalingv2alpha1.HorizontalPodAutoscaler) (*autoscalingv2alpha1.HorizontalPodAutoscaler, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("autoscaling", "v2alpha1", ns, "horizontalpodautoscalers", "")
	resp := new(autoscalingv2alpha1.HorizontalPodAutoscaler)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *AutoscalingV2Alpha1) UpdateHorizontalPodAutoscaler(ctx context.Context, obj *autoscalingv2alpha1.HorizontalPodAutoscaler) (*autoscalingv2alpha1.HorizontalPodAutoscaler, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("autoscaling", "v2alpha1", *md.Namespace, "horizontalpodautoscalers", *md.Name)
	resp := new(autoscalingv2alpha1.HorizontalPodAutoscaler)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *AutoscalingV2Alpha1) DeleteHorizontalPodAutoscaler(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("autoscaling", "v2alpha1", namespace, "horizontalpodautoscalers", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *AutoscalingV2Alpha1) GetHorizontalPodAutoscaler(ctx context.Context, name, namespace string) (*autoscalingv2alpha1.HorizontalPodAutoscaler, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("autoscaling", "v2alpha1", namespace, "horizontalpodautoscalers", name)
	resp := new(autoscalingv2alpha1.HorizontalPodAutoscaler)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type AutoscalingV2Alpha1HorizontalPodAutoscalerWatcher struct {
	watcher *watcher
}

func (w *AutoscalingV2Alpha1HorizontalPodAutoscalerWatcher) Next() (*versioned.Event, *autoscalingv2alpha1.HorizontalPodAutoscaler, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(autoscalingv2alpha1.HorizontalPodAutoscaler)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *AutoscalingV2Alpha1HorizontalPodAutoscalerWatcher) Close() error {
	return w.watcher.Close()
}

func (c *AutoscalingV2Alpha1) WatchHorizontalPodAutoscalers(ctx context.Context, namespace string, options ...Option) (*AutoscalingV2Alpha1HorizontalPodAutoscalerWatcher, error) {
	url := c.client.urlFor("autoscaling", "v2alpha1", namespace, "horizontalpodautoscalers", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &AutoscalingV2Alpha1HorizontalPodAutoscalerWatcher{watcher}, nil
}

func (c *AutoscalingV2Alpha1) ListHorizontalPodAutoscalers(ctx context.Context, namespace string, options ...Option) (*autoscalingv2alpha1.HorizontalPodAutoscalerList, error) {
	url := c.client.urlFor("autoscaling", "v2alpha1", namespace, "horizontalpodautoscalers", "", options...)
	resp := new(autoscalingv2alpha1.HorizontalPodAutoscalerList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// BatchV1 returns a client for interacting with the batch/v1 API group.
func (c *Client) BatchV1() *BatchV1 {
	return &BatchV1{c}
}

// BatchV1 is a client for interacting with the batch/v1 API group.
type BatchV1 struct {
	client *Client
}

func (c *BatchV1) CreateJob(ctx context.Context, obj *batchv1.Job) (*batchv1.Job, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("batch", "v1", ns, "jobs", "")
	resp := new(batchv1.Job)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *BatchV1) UpdateJob(ctx context.Context, obj *batchv1.Job) (*batchv1.Job, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("batch", "v1", *md.Namespace, "jobs", *md.Name)
	resp := new(batchv1.Job)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *BatchV1) DeleteJob(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("batch", "v1", namespace, "jobs", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *BatchV1) GetJob(ctx context.Context, name, namespace string) (*batchv1.Job, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("batch", "v1", namespace, "jobs", name)
	resp := new(batchv1.Job)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type BatchV1JobWatcher struct {
	watcher *watcher
}

func (w *BatchV1JobWatcher) Next() (*versioned.Event, *batchv1.Job, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(batchv1.Job)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *BatchV1JobWatcher) Close() error {
	return w.watcher.Close()
}

func (c *BatchV1) WatchJobs(ctx context.Context, namespace string, options ...Option) (*BatchV1JobWatcher, error) {
	url := c.client.urlFor("batch", "v1", namespace, "jobs", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &BatchV1JobWatcher{watcher}, nil
}

func (c *BatchV1) ListJobs(ctx context.Context, namespace string, options ...Option) (*batchv1.JobList, error) {
	url := c.client.urlFor("batch", "v1", namespace, "jobs", "", options...)
	resp := new(batchv1.JobList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// BatchV2Alpha1 returns a client for interacting with the batch/v2alpha1 API group.
func (c *Client) BatchV2Alpha1() *BatchV2Alpha1 {
	return &BatchV2Alpha1{c}
}

// BatchV2Alpha1 is a client for interacting with the batch/v2alpha1 API group.
type BatchV2Alpha1 struct {
	client *Client
}

func (c *BatchV2Alpha1) CreateCronJob(ctx context.Context, obj *batchv2alpha1.CronJob) (*batchv2alpha1.CronJob, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("batch", "v2alpha1", ns, "cronjobs", "")
	resp := new(batchv2alpha1.CronJob)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *BatchV2Alpha1) UpdateCronJob(ctx context.Context, obj *batchv2alpha1.CronJob) (*batchv2alpha1.CronJob, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("batch", "v2alpha1", *md.Namespace, "cronjobs", *md.Name)
	resp := new(batchv2alpha1.CronJob)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *BatchV2Alpha1) DeleteCronJob(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("batch", "v2alpha1", namespace, "cronjobs", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *BatchV2Alpha1) GetCronJob(ctx context.Context, name, namespace string) (*batchv2alpha1.CronJob, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("batch", "v2alpha1", namespace, "cronjobs", name)
	resp := new(batchv2alpha1.CronJob)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type BatchV2Alpha1CronJobWatcher struct {
	watcher *watcher
}

func (w *BatchV2Alpha1CronJobWatcher) Next() (*versioned.Event, *batchv2alpha1.CronJob, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(batchv2alpha1.CronJob)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *BatchV2Alpha1CronJobWatcher) Close() error {
	return w.watcher.Close()
}

func (c *BatchV2Alpha1) WatchCronJobs(ctx context.Context, namespace string, options ...Option) (*BatchV2Alpha1CronJobWatcher, error) {
	url := c.client.urlFor("batch", "v2alpha1", namespace, "cronjobs", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &BatchV2Alpha1CronJobWatcher{watcher}, nil
}

func (c *BatchV2Alpha1) ListCronJobs(ctx context.Context, namespace string, options ...Option) (*batchv2alpha1.CronJobList, error) {
	url := c.client.urlFor("batch", "v2alpha1", namespace, "cronjobs", "", options...)
	resp := new(batchv2alpha1.CronJobList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *BatchV2Alpha1) CreateJobTemplate(ctx context.Context, obj *batchv2alpha1.JobTemplate) (*batchv2alpha1.JobTemplate, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("batch", "v2alpha1", ns, "jobtemplates", "")
	resp := new(batchv2alpha1.JobTemplate)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *BatchV2Alpha1) UpdateJobTemplate(ctx context.Context, obj *batchv2alpha1.JobTemplate) (*batchv2alpha1.JobTemplate, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("batch", "v2alpha1", *md.Namespace, "jobtemplates", *md.Name)
	resp := new(batchv2alpha1.JobTemplate)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *BatchV2Alpha1) DeleteJobTemplate(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("batch", "v2alpha1", namespace, "jobtemplates", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *BatchV2Alpha1) GetJobTemplate(ctx context.Context, name, namespace string) (*batchv2alpha1.JobTemplate, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("batch", "v2alpha1", namespace, "jobtemplates", name)
	resp := new(batchv2alpha1.JobTemplate)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// CertificatesV1Alpha1 returns a client for interacting with the certificates.k8s.io/v1alpha1 API group.
func (c *Client) CertificatesV1Alpha1() *CertificatesV1Alpha1 {
	return &CertificatesV1Alpha1{c}
}

// CertificatesV1Alpha1 is a client for interacting with the certificates.k8s.io/v1alpha1 API group.
type CertificatesV1Alpha1 struct {
	client *Client
}

func (c *CertificatesV1Alpha1) CreateCertificateSigningRequest(ctx context.Context, obj *certificatesv1alpha1.CertificateSigningRequest) (*certificatesv1alpha1.CertificateSigningRequest, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("certificates.k8s.io", "v1alpha1", ns, "certificatesigningrequests", "")
	resp := new(certificatesv1alpha1.CertificateSigningRequest)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CertificatesV1Alpha1) UpdateCertificateSigningRequest(ctx context.Context, obj *certificatesv1alpha1.CertificateSigningRequest) (*certificatesv1alpha1.CertificateSigningRequest, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("certificates.k8s.io", "v1alpha1", *md.Namespace, "certificatesigningrequests", *md.Name)
	resp := new(certificatesv1alpha1.CertificateSigningRequest)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CertificatesV1Alpha1) DeleteCertificateSigningRequest(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("certificates.k8s.io", "v1alpha1", AllNamespaces, "certificatesigningrequests", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *CertificatesV1Alpha1) GetCertificateSigningRequest(ctx context.Context, name string) (*certificatesv1alpha1.CertificateSigningRequest, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("certificates.k8s.io", "v1alpha1", AllNamespaces, "certificatesigningrequests", name)
	resp := new(certificatesv1alpha1.CertificateSigningRequest)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type CertificatesV1Alpha1CertificateSigningRequestWatcher struct {
	watcher *watcher
}

func (w *CertificatesV1Alpha1CertificateSigningRequestWatcher) Next() (*versioned.Event, *certificatesv1alpha1.CertificateSigningRequest, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(certificatesv1alpha1.CertificateSigningRequest)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *CertificatesV1Alpha1CertificateSigningRequestWatcher) Close() error {
	return w.watcher.Close()
}

func (c *CertificatesV1Alpha1) WatchCertificateSigningRequests(ctx context.Context, options ...Option) (*CertificatesV1Alpha1CertificateSigningRequestWatcher, error) {
	url := c.client.urlFor("certificates.k8s.io", "v1alpha1", AllNamespaces, "certificatesigningrequests", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &CertificatesV1Alpha1CertificateSigningRequestWatcher{watcher}, nil
}

func (c *CertificatesV1Alpha1) ListCertificateSigningRequests(ctx context.Context, options ...Option) (*certificatesv1alpha1.CertificateSigningRequestList, error) {
	url := c.client.urlFor("certificates.k8s.io", "v1alpha1", AllNamespaces, "certificatesigningrequests", "", options...)
	resp := new(certificatesv1alpha1.CertificateSigningRequestList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// CertificatesV1Beta1 returns a client for interacting with the certificates.k8s.io/v1beta1 API group.
func (c *Client) CertificatesV1Beta1() *CertificatesV1Beta1 {
	return &CertificatesV1Beta1{c}
}

// CertificatesV1Beta1 is a client for interacting with the certificates.k8s.io/v1beta1 API group.
type CertificatesV1Beta1 struct {
	client *Client
}

func (c *CertificatesV1Beta1) CreateCertificateSigningRequest(ctx context.Context, obj *certificatesv1beta1.CertificateSigningRequest) (*certificatesv1beta1.CertificateSigningRequest, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("certificates.k8s.io", "v1beta1", ns, "certificatesigningrequests", "")
	resp := new(certificatesv1beta1.CertificateSigningRequest)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CertificatesV1Beta1) UpdateCertificateSigningRequest(ctx context.Context, obj *certificatesv1beta1.CertificateSigningRequest) (*certificatesv1beta1.CertificateSigningRequest, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("certificates.k8s.io", "v1beta1", *md.Namespace, "certificatesigningrequests", *md.Name)
	resp := new(certificatesv1beta1.CertificateSigningRequest)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *CertificatesV1Beta1) DeleteCertificateSigningRequest(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("certificates.k8s.io", "v1beta1", AllNamespaces, "certificatesigningrequests", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *CertificatesV1Beta1) GetCertificateSigningRequest(ctx context.Context, name string) (*certificatesv1beta1.CertificateSigningRequest, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("certificates.k8s.io", "v1beta1", AllNamespaces, "certificatesigningrequests", name)
	resp := new(certificatesv1beta1.CertificateSigningRequest)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type CertificatesV1Beta1CertificateSigningRequestWatcher struct {
	watcher *watcher
}

func (w *CertificatesV1Beta1CertificateSigningRequestWatcher) Next() (*versioned.Event, *certificatesv1beta1.CertificateSigningRequest, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(certificatesv1beta1.CertificateSigningRequest)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *CertificatesV1Beta1CertificateSigningRequestWatcher) Close() error {
	return w.watcher.Close()
}

func (c *CertificatesV1Beta1) WatchCertificateSigningRequests(ctx context.Context, options ...Option) (*CertificatesV1Beta1CertificateSigningRequestWatcher, error) {
	url := c.client.urlFor("certificates.k8s.io", "v1beta1", AllNamespaces, "certificatesigningrequests", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &CertificatesV1Beta1CertificateSigningRequestWatcher{watcher}, nil
}

func (c *CertificatesV1Beta1) ListCertificateSigningRequests(ctx context.Context, options ...Option) (*certificatesv1beta1.CertificateSigningRequestList, error) {
	url := c.client.urlFor("certificates.k8s.io", "v1beta1", AllNamespaces, "certificatesigningrequests", "", options...)
	resp := new(certificatesv1beta1.CertificateSigningRequestList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// ExtensionsV1Beta1 returns a client for interacting with the extensions/v1beta1 API group.
func (c *Client) ExtensionsV1Beta1() *ExtensionsV1Beta1 {
	return &ExtensionsV1Beta1{c}
}

// ExtensionsV1Beta1 is a client for interacting with the extensions/v1beta1 API group.
type ExtensionsV1Beta1 struct {
	client *Client
}

func (c *ExtensionsV1Beta1) CreateDaemonSet(ctx context.Context, obj *extensionsv1beta1.DaemonSet) (*extensionsv1beta1.DaemonSet, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("extensions", "v1beta1", ns, "daemonsets", "")
	resp := new(extensionsv1beta1.DaemonSet)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *ExtensionsV1Beta1) UpdateDaemonSet(ctx context.Context, obj *extensionsv1beta1.DaemonSet) (*extensionsv1beta1.DaemonSet, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("extensions", "v1beta1", *md.Namespace, "daemonsets", *md.Name)
	resp := new(extensionsv1beta1.DaemonSet)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *ExtensionsV1Beta1) DeleteDaemonSet(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("extensions", "v1beta1", namespace, "daemonsets", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *ExtensionsV1Beta1) GetDaemonSet(ctx context.Context, name, namespace string) (*extensionsv1beta1.DaemonSet, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("extensions", "v1beta1", namespace, "daemonsets", name)
	resp := new(extensionsv1beta1.DaemonSet)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type ExtensionsV1Beta1DaemonSetWatcher struct {
	watcher *watcher
}

func (w *ExtensionsV1Beta1DaemonSetWatcher) Next() (*versioned.Event, *extensionsv1beta1.DaemonSet, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(extensionsv1beta1.DaemonSet)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *ExtensionsV1Beta1DaemonSetWatcher) Close() error {
	return w.watcher.Close()
}

func (c *ExtensionsV1Beta1) WatchDaemonSets(ctx context.Context, namespace string, options ...Option) (*ExtensionsV1Beta1DaemonSetWatcher, error) {
	url := c.client.urlFor("extensions", "v1beta1", namespace, "daemonsets", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &ExtensionsV1Beta1DaemonSetWatcher{watcher}, nil
}

func (c *ExtensionsV1Beta1) ListDaemonSets(ctx context.Context, namespace string, options ...Option) (*extensionsv1beta1.DaemonSetList, error) {
	url := c.client.urlFor("extensions", "v1beta1", namespace, "daemonsets", "", options...)
	resp := new(extensionsv1beta1.DaemonSetList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *ExtensionsV1Beta1) CreateDeployment(ctx context.Context, obj *extensionsv1beta1.Deployment) (*extensionsv1beta1.Deployment, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("extensions", "v1beta1", ns, "deployments", "")
	resp := new(extensionsv1beta1.Deployment)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *ExtensionsV1Beta1) UpdateDeployment(ctx context.Context, obj *extensionsv1beta1.Deployment) (*extensionsv1beta1.Deployment, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("extensions", "v1beta1", *md.Namespace, "deployments", *md.Name)
	resp := new(extensionsv1beta1.Deployment)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *ExtensionsV1Beta1) DeleteDeployment(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("extensions", "v1beta1", namespace, "deployments", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *ExtensionsV1Beta1) GetDeployment(ctx context.Context, name, namespace string) (*extensionsv1beta1.Deployment, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("extensions", "v1beta1", namespace, "deployments", name)
	resp := new(extensionsv1beta1.Deployment)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type ExtensionsV1Beta1DeploymentWatcher struct {
	watcher *watcher
}

func (w *ExtensionsV1Beta1DeploymentWatcher) Next() (*versioned.Event, *extensionsv1beta1.Deployment, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(extensionsv1beta1.Deployment)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *ExtensionsV1Beta1DeploymentWatcher) Close() error {
	return w.watcher.Close()
}

func (c *ExtensionsV1Beta1) WatchDeployments(ctx context.Context, namespace string, options ...Option) (*ExtensionsV1Beta1DeploymentWatcher, error) {
	url := c.client.urlFor("extensions", "v1beta1", namespace, "deployments", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &ExtensionsV1Beta1DeploymentWatcher{watcher}, nil
}

func (c *ExtensionsV1Beta1) ListDeployments(ctx context.Context, namespace string, options ...Option) (*extensionsv1beta1.DeploymentList, error) {
	url := c.client.urlFor("extensions", "v1beta1", namespace, "deployments", "", options...)
	resp := new(extensionsv1beta1.DeploymentList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *ExtensionsV1Beta1) CreateIngress(ctx context.Context, obj *extensionsv1beta1.Ingress) (*extensionsv1beta1.Ingress, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("extensions", "v1beta1", ns, "ingresses", "")
	resp := new(extensionsv1beta1.Ingress)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *ExtensionsV1Beta1) UpdateIngress(ctx context.Context, obj *extensionsv1beta1.Ingress) (*extensionsv1beta1.Ingress, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("extensions", "v1beta1", *md.Namespace, "ingresses", *md.Name)
	resp := new(extensionsv1beta1.Ingress)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *ExtensionsV1Beta1) DeleteIngress(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("extensions", "v1beta1", namespace, "ingresses", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *ExtensionsV1Beta1) GetIngress(ctx context.Context, name, namespace string) (*extensionsv1beta1.Ingress, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("extensions", "v1beta1", namespace, "ingresses", name)
	resp := new(extensionsv1beta1.Ingress)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type ExtensionsV1Beta1IngressWatcher struct {
	watcher *watcher
}

func (w *ExtensionsV1Beta1IngressWatcher) Next() (*versioned.Event, *extensionsv1beta1.Ingress, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(extensionsv1beta1.Ingress)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *ExtensionsV1Beta1IngressWatcher) Close() error {
	return w.watcher.Close()
}

func (c *ExtensionsV1Beta1) WatchIngresses(ctx context.Context, namespace string, options ...Option) (*ExtensionsV1Beta1IngressWatcher, error) {
	url := c.client.urlFor("extensions", "v1beta1", namespace, "ingresses", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &ExtensionsV1Beta1IngressWatcher{watcher}, nil
}

func (c *ExtensionsV1Beta1) ListIngresses(ctx context.Context, namespace string, options ...Option) (*extensionsv1beta1.IngressList, error) {
	url := c.client.urlFor("extensions", "v1beta1", namespace, "ingresses", "", options...)
	resp := new(extensionsv1beta1.IngressList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *ExtensionsV1Beta1) CreateNetworkPolicy(ctx context.Context, obj *extensionsv1beta1.NetworkPolicy) (*extensionsv1beta1.NetworkPolicy, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("extensions", "v1beta1", ns, "networkpolicies", "")
	resp := new(extensionsv1beta1.NetworkPolicy)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *ExtensionsV1Beta1) UpdateNetworkPolicy(ctx context.Context, obj *extensionsv1beta1.NetworkPolicy) (*extensionsv1beta1.NetworkPolicy, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("extensions", "v1beta1", *md.Namespace, "networkpolicies", *md.Name)
	resp := new(extensionsv1beta1.NetworkPolicy)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *ExtensionsV1Beta1) DeleteNetworkPolicy(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("extensions", "v1beta1", namespace, "networkpolicies", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *ExtensionsV1Beta1) GetNetworkPolicy(ctx context.Context, name, namespace string) (*extensionsv1beta1.NetworkPolicy, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("extensions", "v1beta1", namespace, "networkpolicies", name)
	resp := new(extensionsv1beta1.NetworkPolicy)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type ExtensionsV1Beta1NetworkPolicyWatcher struct {
	watcher *watcher
}

func (w *ExtensionsV1Beta1NetworkPolicyWatcher) Next() (*versioned.Event, *extensionsv1beta1.NetworkPolicy, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(extensionsv1beta1.NetworkPolicy)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *ExtensionsV1Beta1NetworkPolicyWatcher) Close() error {
	return w.watcher.Close()
}

func (c *ExtensionsV1Beta1) WatchNetworkPolicies(ctx context.Context, namespace string, options ...Option) (*ExtensionsV1Beta1NetworkPolicyWatcher, error) {
	url := c.client.urlFor("extensions", "v1beta1", namespace, "networkpolicies", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &ExtensionsV1Beta1NetworkPolicyWatcher{watcher}, nil
}

func (c *ExtensionsV1Beta1) ListNetworkPolicies(ctx context.Context, namespace string, options ...Option) (*extensionsv1beta1.NetworkPolicyList, error) {
	url := c.client.urlFor("extensions", "v1beta1", namespace, "networkpolicies", "", options...)
	resp := new(extensionsv1beta1.NetworkPolicyList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *ExtensionsV1Beta1) CreatePodSecurityPolicy(ctx context.Context, obj *extensionsv1beta1.PodSecurityPolicy) (*extensionsv1beta1.PodSecurityPolicy, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("extensions", "v1beta1", ns, "podsecuritypolicies", "")
	resp := new(extensionsv1beta1.PodSecurityPolicy)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *ExtensionsV1Beta1) UpdatePodSecurityPolicy(ctx context.Context, obj *extensionsv1beta1.PodSecurityPolicy) (*extensionsv1beta1.PodSecurityPolicy, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("extensions", "v1beta1", *md.Namespace, "podsecuritypolicies", *md.Name)
	resp := new(extensionsv1beta1.PodSecurityPolicy)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *ExtensionsV1Beta1) DeletePodSecurityPolicy(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("extensions", "v1beta1", AllNamespaces, "podsecuritypolicies", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *ExtensionsV1Beta1) GetPodSecurityPolicy(ctx context.Context, name string) (*extensionsv1beta1.PodSecurityPolicy, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("extensions", "v1beta1", AllNamespaces, "podsecuritypolicies", name)
	resp := new(extensionsv1beta1.PodSecurityPolicy)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type ExtensionsV1Beta1PodSecurityPolicyWatcher struct {
	watcher *watcher
}

func (w *ExtensionsV1Beta1PodSecurityPolicyWatcher) Next() (*versioned.Event, *extensionsv1beta1.PodSecurityPolicy, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(extensionsv1beta1.PodSecurityPolicy)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *ExtensionsV1Beta1PodSecurityPolicyWatcher) Close() error {
	return w.watcher.Close()
}

func (c *ExtensionsV1Beta1) WatchPodSecurityPolicies(ctx context.Context, options ...Option) (*ExtensionsV1Beta1PodSecurityPolicyWatcher, error) {
	url := c.client.urlFor("extensions", "v1beta1", AllNamespaces, "podsecuritypolicies", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &ExtensionsV1Beta1PodSecurityPolicyWatcher{watcher}, nil
}

func (c *ExtensionsV1Beta1) ListPodSecurityPolicies(ctx context.Context, options ...Option) (*extensionsv1beta1.PodSecurityPolicyList, error) {
	url := c.client.urlFor("extensions", "v1beta1", AllNamespaces, "podsecuritypolicies", "", options...)
	resp := new(extensionsv1beta1.PodSecurityPolicyList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *ExtensionsV1Beta1) CreateReplicaSet(ctx context.Context, obj *extensionsv1beta1.ReplicaSet) (*extensionsv1beta1.ReplicaSet, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("extensions", "v1beta1", ns, "replicasets", "")
	resp := new(extensionsv1beta1.ReplicaSet)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *ExtensionsV1Beta1) UpdateReplicaSet(ctx context.Context, obj *extensionsv1beta1.ReplicaSet) (*extensionsv1beta1.ReplicaSet, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("extensions", "v1beta1", *md.Namespace, "replicasets", *md.Name)
	resp := new(extensionsv1beta1.ReplicaSet)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *ExtensionsV1Beta1) DeleteReplicaSet(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("extensions", "v1beta1", namespace, "replicasets", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *ExtensionsV1Beta1) GetReplicaSet(ctx context.Context, name, namespace string) (*extensionsv1beta1.ReplicaSet, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("extensions", "v1beta1", namespace, "replicasets", name)
	resp := new(extensionsv1beta1.ReplicaSet)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type ExtensionsV1Beta1ReplicaSetWatcher struct {
	watcher *watcher
}

func (w *ExtensionsV1Beta1ReplicaSetWatcher) Next() (*versioned.Event, *extensionsv1beta1.ReplicaSet, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(extensionsv1beta1.ReplicaSet)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *ExtensionsV1Beta1ReplicaSetWatcher) Close() error {
	return w.watcher.Close()
}

func (c *ExtensionsV1Beta1) WatchReplicaSets(ctx context.Context, namespace string, options ...Option) (*ExtensionsV1Beta1ReplicaSetWatcher, error) {
	url := c.client.urlFor("extensions", "v1beta1", namespace, "replicasets", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &ExtensionsV1Beta1ReplicaSetWatcher{watcher}, nil
}

func (c *ExtensionsV1Beta1) ListReplicaSets(ctx context.Context, namespace string, options ...Option) (*extensionsv1beta1.ReplicaSetList, error) {
	url := c.client.urlFor("extensions", "v1beta1", namespace, "replicasets", "", options...)
	resp := new(extensionsv1beta1.ReplicaSetList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *ExtensionsV1Beta1) CreateScale(ctx context.Context, obj *extensionsv1beta1.Scale) (*extensionsv1beta1.Scale, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("extensions", "v1beta1", ns, "scales", "")
	resp := new(extensionsv1beta1.Scale)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *ExtensionsV1Beta1) UpdateScale(ctx context.Context, obj *extensionsv1beta1.Scale) (*extensionsv1beta1.Scale, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("extensions", "v1beta1", *md.Namespace, "scales", *md.Name)
	resp := new(extensionsv1beta1.Scale)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *ExtensionsV1Beta1) DeleteScale(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("extensions", "v1beta1", namespace, "scales", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *ExtensionsV1Beta1) GetScale(ctx context.Context, name, namespace string) (*extensionsv1beta1.Scale, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("extensions", "v1beta1", namespace, "scales", name)
	resp := new(extensionsv1beta1.Scale)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *ExtensionsV1Beta1) CreateThirdPartyResource(ctx context.Context, obj *extensionsv1beta1.ThirdPartyResource) (*extensionsv1beta1.ThirdPartyResource, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("extensions", "v1beta1", ns, "thirdpartyresources", "")
	resp := new(extensionsv1beta1.ThirdPartyResource)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *ExtensionsV1Beta1) UpdateThirdPartyResource(ctx context.Context, obj *extensionsv1beta1.ThirdPartyResource) (*extensionsv1beta1.ThirdPartyResource, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("extensions", "v1beta1", *md.Namespace, "thirdpartyresources", *md.Name)
	resp := new(extensionsv1beta1.ThirdPartyResource)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *ExtensionsV1Beta1) DeleteThirdPartyResource(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("extensions", "v1beta1", AllNamespaces, "thirdpartyresources", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *ExtensionsV1Beta1) GetThirdPartyResource(ctx context.Context, name string) (*extensionsv1beta1.ThirdPartyResource, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("extensions", "v1beta1", AllNamespaces, "thirdpartyresources", name)
	resp := new(extensionsv1beta1.ThirdPartyResource)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type ExtensionsV1Beta1ThirdPartyResourceWatcher struct {
	watcher *watcher
}

func (w *ExtensionsV1Beta1ThirdPartyResourceWatcher) Next() (*versioned.Event, *extensionsv1beta1.ThirdPartyResource, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(extensionsv1beta1.ThirdPartyResource)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *ExtensionsV1Beta1ThirdPartyResourceWatcher) Close() error {
	return w.watcher.Close()
}

func (c *ExtensionsV1Beta1) WatchThirdPartyResources(ctx context.Context, options ...Option) (*ExtensionsV1Beta1ThirdPartyResourceWatcher, error) {
	url := c.client.urlFor("extensions", "v1beta1", AllNamespaces, "thirdpartyresources", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &ExtensionsV1Beta1ThirdPartyResourceWatcher{watcher}, nil
}

func (c *ExtensionsV1Beta1) ListThirdPartyResources(ctx context.Context, options ...Option) (*extensionsv1beta1.ThirdPartyResourceList, error) {
	url := c.client.urlFor("extensions", "v1beta1", AllNamespaces, "thirdpartyresources", "", options...)
	resp := new(extensionsv1beta1.ThirdPartyResourceList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *ExtensionsV1Beta1) CreateThirdPartyResourceData(ctx context.Context, obj *extensionsv1beta1.ThirdPartyResourceData) (*extensionsv1beta1.ThirdPartyResourceData, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("extensions", "v1beta1", ns, "thirdpartyresourcedatas", "")
	resp := new(extensionsv1beta1.ThirdPartyResourceData)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *ExtensionsV1Beta1) UpdateThirdPartyResourceData(ctx context.Context, obj *extensionsv1beta1.ThirdPartyResourceData) (*extensionsv1beta1.ThirdPartyResourceData, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("extensions", "v1beta1", *md.Namespace, "thirdpartyresourcedatas", *md.Name)
	resp := new(extensionsv1beta1.ThirdPartyResourceData)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *ExtensionsV1Beta1) DeleteThirdPartyResourceData(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("extensions", "v1beta1", namespace, "thirdpartyresourcedatas", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *ExtensionsV1Beta1) GetThirdPartyResourceData(ctx context.Context, name, namespace string) (*extensionsv1beta1.ThirdPartyResourceData, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("extensions", "v1beta1", namespace, "thirdpartyresourcedatas", name)
	resp := new(extensionsv1beta1.ThirdPartyResourceData)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type ExtensionsV1Beta1ThirdPartyResourceDataWatcher struct {
	watcher *watcher
}

func (w *ExtensionsV1Beta1ThirdPartyResourceDataWatcher) Next() (*versioned.Event, *extensionsv1beta1.ThirdPartyResourceData, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(extensionsv1beta1.ThirdPartyResourceData)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *ExtensionsV1Beta1ThirdPartyResourceDataWatcher) Close() error {
	return w.watcher.Close()
}

func (c *ExtensionsV1Beta1) WatchThirdPartyResourceDatas(ctx context.Context, namespace string, options ...Option) (*ExtensionsV1Beta1ThirdPartyResourceDataWatcher, error) {
	url := c.client.urlFor("extensions", "v1beta1", namespace, "thirdpartyresourcedatas", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &ExtensionsV1Beta1ThirdPartyResourceDataWatcher{watcher}, nil
}

func (c *ExtensionsV1Beta1) ListThirdPartyResourceDatas(ctx context.Context, namespace string, options ...Option) (*extensionsv1beta1.ThirdPartyResourceDataList, error) {
	url := c.client.urlFor("extensions", "v1beta1", namespace, "thirdpartyresourcedatas", "", options...)
	resp := new(extensionsv1beta1.ThirdPartyResourceDataList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// ImagepolicyV1Alpha1 returns a client for interacting with the imagepolicy/v1alpha1 API group.
func (c *Client) ImagepolicyV1Alpha1() *ImagepolicyV1Alpha1 {
	return &ImagepolicyV1Alpha1{c}
}

// ImagepolicyV1Alpha1 is a client for interacting with the imagepolicy/v1alpha1 API group.
type ImagepolicyV1Alpha1 struct {
	client *Client
}

func (c *ImagepolicyV1Alpha1) CreateImageReview(ctx context.Context, obj *imagepolicyv1alpha1.ImageReview) (*imagepolicyv1alpha1.ImageReview, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("imagepolicy", "v1alpha1", ns, "imagereviews", "")
	resp := new(imagepolicyv1alpha1.ImageReview)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *ImagepolicyV1Alpha1) UpdateImageReview(ctx context.Context, obj *imagepolicyv1alpha1.ImageReview) (*imagepolicyv1alpha1.ImageReview, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("imagepolicy", "v1alpha1", *md.Namespace, "imagereviews", *md.Name)
	resp := new(imagepolicyv1alpha1.ImageReview)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *ImagepolicyV1Alpha1) DeleteImageReview(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("imagepolicy", "v1alpha1", AllNamespaces, "imagereviews", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *ImagepolicyV1Alpha1) GetImageReview(ctx context.Context, name string) (*imagepolicyv1alpha1.ImageReview, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("imagepolicy", "v1alpha1", AllNamespaces, "imagereviews", name)
	resp := new(imagepolicyv1alpha1.ImageReview)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// PolicyV1Alpha1 returns a client for interacting with the policy/v1alpha1 API group.
func (c *Client) PolicyV1Alpha1() *PolicyV1Alpha1 {
	return &PolicyV1Alpha1{c}
}

// PolicyV1Alpha1 is a client for interacting with the policy/v1alpha1 API group.
type PolicyV1Alpha1 struct {
	client *Client
}

func (c *PolicyV1Alpha1) CreateEviction(ctx context.Context, obj *policyv1alpha1.Eviction) (*policyv1alpha1.Eviction, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("policy", "v1alpha1", ns, "evictions", "")
	resp := new(policyv1alpha1.Eviction)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *PolicyV1Alpha1) UpdateEviction(ctx context.Context, obj *policyv1alpha1.Eviction) (*policyv1alpha1.Eviction, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("policy", "v1alpha1", *md.Namespace, "evictions", *md.Name)
	resp := new(policyv1alpha1.Eviction)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *PolicyV1Alpha1) DeleteEviction(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("policy", "v1alpha1", namespace, "evictions", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *PolicyV1Alpha1) GetEviction(ctx context.Context, name, namespace string) (*policyv1alpha1.Eviction, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("policy", "v1alpha1", namespace, "evictions", name)
	resp := new(policyv1alpha1.Eviction)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *PolicyV1Alpha1) CreatePodDisruptionBudget(ctx context.Context, obj *policyv1alpha1.PodDisruptionBudget) (*policyv1alpha1.PodDisruptionBudget, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("policy", "v1alpha1", ns, "poddisruptionbudgets", "")
	resp := new(policyv1alpha1.PodDisruptionBudget)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *PolicyV1Alpha1) UpdatePodDisruptionBudget(ctx context.Context, obj *policyv1alpha1.PodDisruptionBudget) (*policyv1alpha1.PodDisruptionBudget, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("policy", "v1alpha1", *md.Namespace, "poddisruptionbudgets", *md.Name)
	resp := new(policyv1alpha1.PodDisruptionBudget)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *PolicyV1Alpha1) DeletePodDisruptionBudget(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("policy", "v1alpha1", namespace, "poddisruptionbudgets", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *PolicyV1Alpha1) GetPodDisruptionBudget(ctx context.Context, name, namespace string) (*policyv1alpha1.PodDisruptionBudget, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("policy", "v1alpha1", namespace, "poddisruptionbudgets", name)
	resp := new(policyv1alpha1.PodDisruptionBudget)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type PolicyV1Alpha1PodDisruptionBudgetWatcher struct {
	watcher *watcher
}

func (w *PolicyV1Alpha1PodDisruptionBudgetWatcher) Next() (*versioned.Event, *policyv1alpha1.PodDisruptionBudget, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(policyv1alpha1.PodDisruptionBudget)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *PolicyV1Alpha1PodDisruptionBudgetWatcher) Close() error {
	return w.watcher.Close()
}

func (c *PolicyV1Alpha1) WatchPodDisruptionBudgets(ctx context.Context, namespace string, options ...Option) (*PolicyV1Alpha1PodDisruptionBudgetWatcher, error) {
	url := c.client.urlFor("policy", "v1alpha1", namespace, "poddisruptionbudgets", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &PolicyV1Alpha1PodDisruptionBudgetWatcher{watcher}, nil
}

func (c *PolicyV1Alpha1) ListPodDisruptionBudgets(ctx context.Context, namespace string, options ...Option) (*policyv1alpha1.PodDisruptionBudgetList, error) {
	url := c.client.urlFor("policy", "v1alpha1", namespace, "poddisruptionbudgets", "", options...)
	resp := new(policyv1alpha1.PodDisruptionBudgetList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// PolicyV1Beta1 returns a client for interacting with the policy/v1beta1 API group.
func (c *Client) PolicyV1Beta1() *PolicyV1Beta1 {
	return &PolicyV1Beta1{c}
}

// PolicyV1Beta1 is a client for interacting with the policy/v1beta1 API group.
type PolicyV1Beta1 struct {
	client *Client
}

func (c *PolicyV1Beta1) CreateEviction(ctx context.Context, obj *policyv1beta1.Eviction) (*policyv1beta1.Eviction, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("policy", "v1beta1", ns, "evictions", "")
	resp := new(policyv1beta1.Eviction)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *PolicyV1Beta1) UpdateEviction(ctx context.Context, obj *policyv1beta1.Eviction) (*policyv1beta1.Eviction, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("policy", "v1beta1", *md.Namespace, "evictions", *md.Name)
	resp := new(policyv1beta1.Eviction)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *PolicyV1Beta1) DeleteEviction(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("policy", "v1beta1", namespace, "evictions", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *PolicyV1Beta1) GetEviction(ctx context.Context, name, namespace string) (*policyv1beta1.Eviction, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("policy", "v1beta1", namespace, "evictions", name)
	resp := new(policyv1beta1.Eviction)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *PolicyV1Beta1) CreatePodDisruptionBudget(ctx context.Context, obj *policyv1beta1.PodDisruptionBudget) (*policyv1beta1.PodDisruptionBudget, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("policy", "v1beta1", ns, "poddisruptionbudgets", "")
	resp := new(policyv1beta1.PodDisruptionBudget)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *PolicyV1Beta1) UpdatePodDisruptionBudget(ctx context.Context, obj *policyv1beta1.PodDisruptionBudget) (*policyv1beta1.PodDisruptionBudget, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("policy", "v1beta1", *md.Namespace, "poddisruptionbudgets", *md.Name)
	resp := new(policyv1beta1.PodDisruptionBudget)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *PolicyV1Beta1) DeletePodDisruptionBudget(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("policy", "v1beta1", namespace, "poddisruptionbudgets", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *PolicyV1Beta1) GetPodDisruptionBudget(ctx context.Context, name, namespace string) (*policyv1beta1.PodDisruptionBudget, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("policy", "v1beta1", namespace, "poddisruptionbudgets", name)
	resp := new(policyv1beta1.PodDisruptionBudget)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type PolicyV1Beta1PodDisruptionBudgetWatcher struct {
	watcher *watcher
}

func (w *PolicyV1Beta1PodDisruptionBudgetWatcher) Next() (*versioned.Event, *policyv1beta1.PodDisruptionBudget, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(policyv1beta1.PodDisruptionBudget)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *PolicyV1Beta1PodDisruptionBudgetWatcher) Close() error {
	return w.watcher.Close()
}

func (c *PolicyV1Beta1) WatchPodDisruptionBudgets(ctx context.Context, namespace string, options ...Option) (*PolicyV1Beta1PodDisruptionBudgetWatcher, error) {
	url := c.client.urlFor("policy", "v1beta1", namespace, "poddisruptionbudgets", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &PolicyV1Beta1PodDisruptionBudgetWatcher{watcher}, nil
}

func (c *PolicyV1Beta1) ListPodDisruptionBudgets(ctx context.Context, namespace string, options ...Option) (*policyv1beta1.PodDisruptionBudgetList, error) {
	url := c.client.urlFor("policy", "v1beta1", namespace, "poddisruptionbudgets", "", options...)
	resp := new(policyv1beta1.PodDisruptionBudgetList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// RBACV1Alpha1 returns a client for interacting with the rbac.authorization.k8s.io/v1alpha1 API group.
func (c *Client) RBACV1Alpha1() *RBACV1Alpha1 {
	return &RBACV1Alpha1{c}
}

// RBACV1Alpha1 is a client for interacting with the rbac.authorization.k8s.io/v1alpha1 API group.
type RBACV1Alpha1 struct {
	client *Client
}

func (c *RBACV1Alpha1) CreateClusterRole(ctx context.Context, obj *rbacv1alpha1.ClusterRole) (*rbacv1alpha1.ClusterRole, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1alpha1", ns, "clusterroles", "")
	resp := new(rbacv1alpha1.ClusterRole)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *RBACV1Alpha1) UpdateClusterRole(ctx context.Context, obj *rbacv1alpha1.ClusterRole) (*rbacv1alpha1.ClusterRole, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1alpha1", *md.Namespace, "clusterroles", *md.Name)
	resp := new(rbacv1alpha1.ClusterRole)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *RBACV1Alpha1) DeleteClusterRole(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1alpha1", AllNamespaces, "clusterroles", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *RBACV1Alpha1) GetClusterRole(ctx context.Context, name string) (*rbacv1alpha1.ClusterRole, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1alpha1", AllNamespaces, "clusterroles", name)
	resp := new(rbacv1alpha1.ClusterRole)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type RBACV1Alpha1ClusterRoleWatcher struct {
	watcher *watcher
}

func (w *RBACV1Alpha1ClusterRoleWatcher) Next() (*versioned.Event, *rbacv1alpha1.ClusterRole, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(rbacv1alpha1.ClusterRole)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *RBACV1Alpha1ClusterRoleWatcher) Close() error {
	return w.watcher.Close()
}

func (c *RBACV1Alpha1) WatchClusterRoles(ctx context.Context, options ...Option) (*RBACV1Alpha1ClusterRoleWatcher, error) {
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1alpha1", AllNamespaces, "clusterroles", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &RBACV1Alpha1ClusterRoleWatcher{watcher}, nil
}

func (c *RBACV1Alpha1) ListClusterRoles(ctx context.Context, options ...Option) (*rbacv1alpha1.ClusterRoleList, error) {
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1alpha1", AllNamespaces, "clusterroles", "", options...)
	resp := new(rbacv1alpha1.ClusterRoleList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *RBACV1Alpha1) CreateClusterRoleBinding(ctx context.Context, obj *rbacv1alpha1.ClusterRoleBinding) (*rbacv1alpha1.ClusterRoleBinding, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1alpha1", ns, "clusterrolebindings", "")
	resp := new(rbacv1alpha1.ClusterRoleBinding)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *RBACV1Alpha1) UpdateClusterRoleBinding(ctx context.Context, obj *rbacv1alpha1.ClusterRoleBinding) (*rbacv1alpha1.ClusterRoleBinding, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1alpha1", *md.Namespace, "clusterrolebindings", *md.Name)
	resp := new(rbacv1alpha1.ClusterRoleBinding)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *RBACV1Alpha1) DeleteClusterRoleBinding(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1alpha1", AllNamespaces, "clusterrolebindings", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *RBACV1Alpha1) GetClusterRoleBinding(ctx context.Context, name string) (*rbacv1alpha1.ClusterRoleBinding, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1alpha1", AllNamespaces, "clusterrolebindings", name)
	resp := new(rbacv1alpha1.ClusterRoleBinding)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type RBACV1Alpha1ClusterRoleBindingWatcher struct {
	watcher *watcher
}

func (w *RBACV1Alpha1ClusterRoleBindingWatcher) Next() (*versioned.Event, *rbacv1alpha1.ClusterRoleBinding, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(rbacv1alpha1.ClusterRoleBinding)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *RBACV1Alpha1ClusterRoleBindingWatcher) Close() error {
	return w.watcher.Close()
}

func (c *RBACV1Alpha1) WatchClusterRoleBindings(ctx context.Context, options ...Option) (*RBACV1Alpha1ClusterRoleBindingWatcher, error) {
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1alpha1", AllNamespaces, "clusterrolebindings", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &RBACV1Alpha1ClusterRoleBindingWatcher{watcher}, nil
}

func (c *RBACV1Alpha1) ListClusterRoleBindings(ctx context.Context, options ...Option) (*rbacv1alpha1.ClusterRoleBindingList, error) {
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1alpha1", AllNamespaces, "clusterrolebindings", "", options...)
	resp := new(rbacv1alpha1.ClusterRoleBindingList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *RBACV1Alpha1) CreateRole(ctx context.Context, obj *rbacv1alpha1.Role) (*rbacv1alpha1.Role, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1alpha1", ns, "roles", "")
	resp := new(rbacv1alpha1.Role)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *RBACV1Alpha1) UpdateRole(ctx context.Context, obj *rbacv1alpha1.Role) (*rbacv1alpha1.Role, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1alpha1", *md.Namespace, "roles", *md.Name)
	resp := new(rbacv1alpha1.Role)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *RBACV1Alpha1) DeleteRole(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1alpha1", namespace, "roles", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *RBACV1Alpha1) GetRole(ctx context.Context, name, namespace string) (*rbacv1alpha1.Role, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1alpha1", namespace, "roles", name)
	resp := new(rbacv1alpha1.Role)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type RBACV1Alpha1RoleWatcher struct {
	watcher *watcher
}

func (w *RBACV1Alpha1RoleWatcher) Next() (*versioned.Event, *rbacv1alpha1.Role, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(rbacv1alpha1.Role)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *RBACV1Alpha1RoleWatcher) Close() error {
	return w.watcher.Close()
}

func (c *RBACV1Alpha1) WatchRoles(ctx context.Context, namespace string, options ...Option) (*RBACV1Alpha1RoleWatcher, error) {
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1alpha1", namespace, "roles", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &RBACV1Alpha1RoleWatcher{watcher}, nil
}

func (c *RBACV1Alpha1) ListRoles(ctx context.Context, namespace string, options ...Option) (*rbacv1alpha1.RoleList, error) {
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1alpha1", namespace, "roles", "", options...)
	resp := new(rbacv1alpha1.RoleList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *RBACV1Alpha1) CreateRoleBinding(ctx context.Context, obj *rbacv1alpha1.RoleBinding) (*rbacv1alpha1.RoleBinding, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1alpha1", ns, "rolebindings", "")
	resp := new(rbacv1alpha1.RoleBinding)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *RBACV1Alpha1) UpdateRoleBinding(ctx context.Context, obj *rbacv1alpha1.RoleBinding) (*rbacv1alpha1.RoleBinding, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1alpha1", *md.Namespace, "rolebindings", *md.Name)
	resp := new(rbacv1alpha1.RoleBinding)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *RBACV1Alpha1) DeleteRoleBinding(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1alpha1", namespace, "rolebindings", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *RBACV1Alpha1) GetRoleBinding(ctx context.Context, name, namespace string) (*rbacv1alpha1.RoleBinding, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1alpha1", namespace, "rolebindings", name)
	resp := new(rbacv1alpha1.RoleBinding)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type RBACV1Alpha1RoleBindingWatcher struct {
	watcher *watcher
}

func (w *RBACV1Alpha1RoleBindingWatcher) Next() (*versioned.Event, *rbacv1alpha1.RoleBinding, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(rbacv1alpha1.RoleBinding)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *RBACV1Alpha1RoleBindingWatcher) Close() error {
	return w.watcher.Close()
}

func (c *RBACV1Alpha1) WatchRoleBindings(ctx context.Context, namespace string, options ...Option) (*RBACV1Alpha1RoleBindingWatcher, error) {
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1alpha1", namespace, "rolebindings", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &RBACV1Alpha1RoleBindingWatcher{watcher}, nil
}

func (c *RBACV1Alpha1) ListRoleBindings(ctx context.Context, namespace string, options ...Option) (*rbacv1alpha1.RoleBindingList, error) {
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1alpha1", namespace, "rolebindings", "", options...)
	resp := new(rbacv1alpha1.RoleBindingList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// RBACV1Beta1 returns a client for interacting with the rbac.authorization.k8s.io/v1beta1 API group.
func (c *Client) RBACV1Beta1() *RBACV1Beta1 {
	return &RBACV1Beta1{c}
}

// RBACV1Beta1 is a client for interacting with the rbac.authorization.k8s.io/v1beta1 API group.
type RBACV1Beta1 struct {
	client *Client
}

func (c *RBACV1Beta1) CreateClusterRole(ctx context.Context, obj *rbacv1beta1.ClusterRole) (*rbacv1beta1.ClusterRole, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1beta1", ns, "clusterroles", "")
	resp := new(rbacv1beta1.ClusterRole)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *RBACV1Beta1) UpdateClusterRole(ctx context.Context, obj *rbacv1beta1.ClusterRole) (*rbacv1beta1.ClusterRole, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1beta1", *md.Namespace, "clusterroles", *md.Name)
	resp := new(rbacv1beta1.ClusterRole)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *RBACV1Beta1) DeleteClusterRole(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1beta1", AllNamespaces, "clusterroles", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *RBACV1Beta1) GetClusterRole(ctx context.Context, name string) (*rbacv1beta1.ClusterRole, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1beta1", AllNamespaces, "clusterroles", name)
	resp := new(rbacv1beta1.ClusterRole)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type RBACV1Beta1ClusterRoleWatcher struct {
	watcher *watcher
}

func (w *RBACV1Beta1ClusterRoleWatcher) Next() (*versioned.Event, *rbacv1beta1.ClusterRole, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(rbacv1beta1.ClusterRole)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *RBACV1Beta1ClusterRoleWatcher) Close() error {
	return w.watcher.Close()
}

func (c *RBACV1Beta1) WatchClusterRoles(ctx context.Context, options ...Option) (*RBACV1Beta1ClusterRoleWatcher, error) {
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1beta1", AllNamespaces, "clusterroles", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &RBACV1Beta1ClusterRoleWatcher{watcher}, nil
}

func (c *RBACV1Beta1) ListClusterRoles(ctx context.Context, options ...Option) (*rbacv1beta1.ClusterRoleList, error) {
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1beta1", AllNamespaces, "clusterroles", "", options...)
	resp := new(rbacv1beta1.ClusterRoleList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *RBACV1Beta1) CreateClusterRoleBinding(ctx context.Context, obj *rbacv1beta1.ClusterRoleBinding) (*rbacv1beta1.ClusterRoleBinding, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1beta1", ns, "clusterrolebindings", "")
	resp := new(rbacv1beta1.ClusterRoleBinding)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *RBACV1Beta1) UpdateClusterRoleBinding(ctx context.Context, obj *rbacv1beta1.ClusterRoleBinding) (*rbacv1beta1.ClusterRoleBinding, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1beta1", *md.Namespace, "clusterrolebindings", *md.Name)
	resp := new(rbacv1beta1.ClusterRoleBinding)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *RBACV1Beta1) DeleteClusterRoleBinding(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1beta1", AllNamespaces, "clusterrolebindings", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *RBACV1Beta1) GetClusterRoleBinding(ctx context.Context, name string) (*rbacv1beta1.ClusterRoleBinding, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1beta1", AllNamespaces, "clusterrolebindings", name)
	resp := new(rbacv1beta1.ClusterRoleBinding)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type RBACV1Beta1ClusterRoleBindingWatcher struct {
	watcher *watcher
}

func (w *RBACV1Beta1ClusterRoleBindingWatcher) Next() (*versioned.Event, *rbacv1beta1.ClusterRoleBinding, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(rbacv1beta1.ClusterRoleBinding)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *RBACV1Beta1ClusterRoleBindingWatcher) Close() error {
	return w.watcher.Close()
}

func (c *RBACV1Beta1) WatchClusterRoleBindings(ctx context.Context, options ...Option) (*RBACV1Beta1ClusterRoleBindingWatcher, error) {
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1beta1", AllNamespaces, "clusterrolebindings", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &RBACV1Beta1ClusterRoleBindingWatcher{watcher}, nil
}

func (c *RBACV1Beta1) ListClusterRoleBindings(ctx context.Context, options ...Option) (*rbacv1beta1.ClusterRoleBindingList, error) {
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1beta1", AllNamespaces, "clusterrolebindings", "", options...)
	resp := new(rbacv1beta1.ClusterRoleBindingList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *RBACV1Beta1) CreateRole(ctx context.Context, obj *rbacv1beta1.Role) (*rbacv1beta1.Role, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1beta1", ns, "roles", "")
	resp := new(rbacv1beta1.Role)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *RBACV1Beta1) UpdateRole(ctx context.Context, obj *rbacv1beta1.Role) (*rbacv1beta1.Role, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1beta1", *md.Namespace, "roles", *md.Name)
	resp := new(rbacv1beta1.Role)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *RBACV1Beta1) DeleteRole(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1beta1", namespace, "roles", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *RBACV1Beta1) GetRole(ctx context.Context, name, namespace string) (*rbacv1beta1.Role, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1beta1", namespace, "roles", name)
	resp := new(rbacv1beta1.Role)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type RBACV1Beta1RoleWatcher struct {
	watcher *watcher
}

func (w *RBACV1Beta1RoleWatcher) Next() (*versioned.Event, *rbacv1beta1.Role, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(rbacv1beta1.Role)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *RBACV1Beta1RoleWatcher) Close() error {
	return w.watcher.Close()
}

func (c *RBACV1Beta1) WatchRoles(ctx context.Context, namespace string, options ...Option) (*RBACV1Beta1RoleWatcher, error) {
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1beta1", namespace, "roles", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &RBACV1Beta1RoleWatcher{watcher}, nil
}

func (c *RBACV1Beta1) ListRoles(ctx context.Context, namespace string, options ...Option) (*rbacv1beta1.RoleList, error) {
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1beta1", namespace, "roles", "", options...)
	resp := new(rbacv1beta1.RoleList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *RBACV1Beta1) CreateRoleBinding(ctx context.Context, obj *rbacv1beta1.RoleBinding) (*rbacv1beta1.RoleBinding, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1beta1", ns, "rolebindings", "")
	resp := new(rbacv1beta1.RoleBinding)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *RBACV1Beta1) UpdateRoleBinding(ctx context.Context, obj *rbacv1beta1.RoleBinding) (*rbacv1beta1.RoleBinding, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1beta1", *md.Namespace, "rolebindings", *md.Name)
	resp := new(rbacv1beta1.RoleBinding)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *RBACV1Beta1) DeleteRoleBinding(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1beta1", namespace, "rolebindings", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *RBACV1Beta1) GetRoleBinding(ctx context.Context, name, namespace string) (*rbacv1beta1.RoleBinding, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1beta1", namespace, "rolebindings", name)
	resp := new(rbacv1beta1.RoleBinding)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type RBACV1Beta1RoleBindingWatcher struct {
	watcher *watcher
}

func (w *RBACV1Beta1RoleBindingWatcher) Next() (*versioned.Event, *rbacv1beta1.RoleBinding, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(rbacv1beta1.RoleBinding)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *RBACV1Beta1RoleBindingWatcher) Close() error {
	return w.watcher.Close()
}

func (c *RBACV1Beta1) WatchRoleBindings(ctx context.Context, namespace string, options ...Option) (*RBACV1Beta1RoleBindingWatcher, error) {
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1beta1", namespace, "rolebindings", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &RBACV1Beta1RoleBindingWatcher{watcher}, nil
}

func (c *RBACV1Beta1) ListRoleBindings(ctx context.Context, namespace string, options ...Option) (*rbacv1beta1.RoleBindingList, error) {
	url := c.client.urlFor("rbac.authorization.k8s.io", "v1beta1", namespace, "rolebindings", "", options...)
	resp := new(rbacv1beta1.RoleBindingList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// SettingsV1Alpha1 returns a client for interacting with the settings/v1alpha1 API group.
func (c *Client) SettingsV1Alpha1() *SettingsV1Alpha1 {
	return &SettingsV1Alpha1{c}
}

// SettingsV1Alpha1 is a client for interacting with the settings/v1alpha1 API group.
type SettingsV1Alpha1 struct {
	client *Client
}

func (c *SettingsV1Alpha1) CreatePodPreset(ctx context.Context, obj *settingsv1alpha1.PodPreset) (*settingsv1alpha1.PodPreset, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("settings", "v1alpha1", ns, "podpresets", "")
	resp := new(settingsv1alpha1.PodPreset)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *SettingsV1Alpha1) UpdatePodPreset(ctx context.Context, obj *settingsv1alpha1.PodPreset) (*settingsv1alpha1.PodPreset, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !true && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if true {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("settings", "v1alpha1", *md.Namespace, "podpresets", *md.Name)
	resp := new(settingsv1alpha1.PodPreset)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *SettingsV1Alpha1) DeletePodPreset(ctx context.Context, name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("settings", "v1alpha1", namespace, "podpresets", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *SettingsV1Alpha1) GetPodPreset(ctx context.Context, name, namespace string) (*settingsv1alpha1.PodPreset, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("settings", "v1alpha1", namespace, "podpresets", name)
	resp := new(settingsv1alpha1.PodPreset)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type SettingsV1Alpha1PodPresetWatcher struct {
	watcher *watcher
}

func (w *SettingsV1Alpha1PodPresetWatcher) Next() (*versioned.Event, *settingsv1alpha1.PodPreset, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(settingsv1alpha1.PodPreset)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *SettingsV1Alpha1PodPresetWatcher) Close() error {
	return w.watcher.Close()
}

func (c *SettingsV1Alpha1) WatchPodPresets(ctx context.Context, namespace string, options ...Option) (*SettingsV1Alpha1PodPresetWatcher, error) {
	url := c.client.urlFor("settings", "v1alpha1", namespace, "podpresets", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &SettingsV1Alpha1PodPresetWatcher{watcher}, nil
}

func (c *SettingsV1Alpha1) ListPodPresets(ctx context.Context, namespace string, options ...Option) (*settingsv1alpha1.PodPresetList, error) {
	url := c.client.urlFor("settings", "v1alpha1", namespace, "podpresets", "", options...)
	resp := new(settingsv1alpha1.PodPresetList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// StorageV1 returns a client for interacting with the storage.k8s.io/v1 API group.
func (c *Client) StorageV1() *StorageV1 {
	return &StorageV1{c}
}

// StorageV1 is a client for interacting with the storage.k8s.io/v1 API group.
type StorageV1 struct {
	client *Client
}

func (c *StorageV1) CreateStorageClass(ctx context.Context, obj *storagev1.StorageClass) (*storagev1.StorageClass, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("storage.k8s.io", "v1", ns, "storageclasses", "")
	resp := new(storagev1.StorageClass)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *StorageV1) UpdateStorageClass(ctx context.Context, obj *storagev1.StorageClass) (*storagev1.StorageClass, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("storage.k8s.io", "v1", *md.Namespace, "storageclasses", *md.Name)
	resp := new(storagev1.StorageClass)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *StorageV1) DeleteStorageClass(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("storage.k8s.io", "v1", AllNamespaces, "storageclasses", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *StorageV1) GetStorageClass(ctx context.Context, name string) (*storagev1.StorageClass, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("storage.k8s.io", "v1", AllNamespaces, "storageclasses", name)
	resp := new(storagev1.StorageClass)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type StorageV1StorageClassWatcher struct {
	watcher *watcher
}

func (w *StorageV1StorageClassWatcher) Next() (*versioned.Event, *storagev1.StorageClass, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(storagev1.StorageClass)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *StorageV1StorageClassWatcher) Close() error {
	return w.watcher.Close()
}

func (c *StorageV1) WatchStorageClasses(ctx context.Context, options ...Option) (*StorageV1StorageClassWatcher, error) {
	url := c.client.urlFor("storage.k8s.io", "v1", AllNamespaces, "storageclasses", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &StorageV1StorageClassWatcher{watcher}, nil
}

func (c *StorageV1) ListStorageClasses(ctx context.Context, options ...Option) (*storagev1.StorageClassList, error) {
	url := c.client.urlFor("storage.k8s.io", "v1", AllNamespaces, "storageclasses", "", options...)
	resp := new(storagev1.StorageClassList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// StorageV1Beta1 returns a client for interacting with the storage.k8s.io/v1beta1 API group.
func (c *Client) StorageV1Beta1() *StorageV1Beta1 {
	return &StorageV1Beta1{c}
}

// StorageV1Beta1 is a client for interacting with the storage.k8s.io/v1beta1 API group.
type StorageV1Beta1 struct {
	client *Client
}

func (c *StorageV1Beta1) CreateStorageClass(ctx context.Context, obj *storagev1beta1.StorageClass) (*storagev1beta1.StorageClass, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("storage.k8s.io", "v1beta1", ns, "storageclasses", "")
	resp := new(storagev1beta1.StorageClass)
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *StorageV1Beta1) UpdateStorageClass(ctx context.Context, obj *storagev1beta1.StorageClass) (*storagev1beta1.StorageClass, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !false && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if false {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("storage.k8s.io", "v1beta1", *md.Namespace, "storageclasses", *md.Name)
	resp := new(storagev1beta1.StorageClass)
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *StorageV1Beta1) DeleteStorageClass(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("storage.k8s.io", "v1beta1", AllNamespaces, "storageclasses", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *StorageV1Beta1) GetStorageClass(ctx context.Context, name string) (*storagev1beta1.StorageClass, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("storage.k8s.io", "v1beta1", AllNamespaces, "storageclasses", name)
	resp := new(storagev1beta1.StorageClass)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type StorageV1Beta1StorageClassWatcher struct {
	watcher *watcher
}

func (w *StorageV1Beta1StorageClassWatcher) Next() (*versioned.Event, *storagev1beta1.StorageClass, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new(storagev1beta1.StorageClass)
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *StorageV1Beta1StorageClassWatcher) Close() error {
	return w.watcher.Close()
}

func (c *StorageV1Beta1) WatchStorageClasses(ctx context.Context, options ...Option) (*StorageV1Beta1StorageClassWatcher, error) {
	url := c.client.urlFor("storage.k8s.io", "v1beta1", AllNamespaces, "storageclasses", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &StorageV1Beta1StorageClassWatcher{watcher}, nil
}

func (c *StorageV1Beta1) ListStorageClasses(ctx context.Context, options ...Option) (*storagev1beta1.StorageClassList, error) {
	url := c.client.urlFor("storage.k8s.io", "v1beta1", AllNamespaces, "storageclasses", "", options...)
	resp := new(storagev1beta1.StorageClassList)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}
