package k8s

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"reflect"
	"strings"

	metav1 "github.com/ericchiang/k8s/apis/meta/v1"
)

type resourceType struct {
	apiGroup   string
	apiVersion string
	name       string
	namespaced bool
}

var (
	resources     = map[reflect.Type]resourceType{}
	resourceLists = map[reflect.Type]resourceType{}
)

// Resource is a Kubernetes resource, such as a Node or Pod.
type Resource interface {
	GetMetadata() *metav1.ObjectMeta
}

// Resource is list of common Kubernetes resources, such as a NodeList or
// PodList.
type ResourceList interface {
	GetMetadata() *metav1.ListMeta
}

func Register(apiGroup, apiVersion, name string, namespaced bool, r Resource) {
	rt := reflect.TypeOf(r)
	if _, ok := resources[rt]; ok {
		panic(fmt.Sprintf("resource registered twice %T", r))
	}
	resources[rt] = resourceType{apiGroup, apiVersion, name, namespaced}
}

func RegisterList(apiGroup, apiVersion, name string, namespaced bool, l ResourceList) {
	rt := reflect.TypeOf(l)
	if _, ok := resources[rt]; ok {
		panic(fmt.Sprintf("resource registered twice %T", l))
	}
	resourceLists[rt] = resourceType{apiGroup, apiVersion, name, namespaced}
}

func urlFor(endpoint, apiGroup, apiVersion, namespace, resource, name string, options ...Option) string {
	basePath := "apis/"
	if apiGroup == "" {
		basePath = "api/"
	}

	var p string
	if namespace != "" {
		p = path.Join(basePath, apiGroup, apiVersion, "namespaces", namespace, resource, name)
	} else {
		p = path.Join(basePath, apiGroup, apiVersion, resource, name)
	}
	e := ""
	if strings.HasSuffix(endpoint, "/") {
		e = endpoint + p
	} else {
		e = endpoint + "/" + p
	}
	if len(options) == 0 {
		return e
	}

	v := url.Values{}
	for _, option := range options {
		key, val := option.queryParam()
		v.Set(key, val)
	}
	return e + "?" + v.Encode()
}

func urlForPath(endpoint, path string) string {
	if strings.HasPrefix(path, "/") {
		path = path[1:]
	}
	if strings.HasSuffix(endpoint, "/") {
		return endpoint + path
	}
	return endpoint + "/" + path
}

func resourceURL(endpoint string, r Resource, withName bool, options ...Option) (string, error) {
	t, ok := resources[reflect.TypeOf(r)]
	if !ok {
		return "", fmt.Errorf("unregistered type %T", r)
	}
	meta := r.GetMetadata()
	if meta == nil {
		return "", errors.New("resource has no object meta")
	}
	switch {
	case t.namespaced && (meta.Namespace == nil || *meta.Namespace == ""):
		return "", errors.New("no resource namespace provided")
	case !t.namespaced && (meta.Namespace != nil && *meta.Namespace != ""):
		return "", errors.New("resource not namespaced")
	case withName && (meta.Name == nil || *meta.Name == ""):
		return "", errors.New("no resource name provided")
	}
	name := ""
	if withName {
		name = *meta.Name
	}
	namespace := ""
	if t.namespaced {
		namespace = *meta.Namespace
	}

	return urlFor(endpoint, t.apiGroup, t.apiVersion, namespace, t.name, name, options...), nil
}

func resourceGetURL(endpoint, namespace, name string, r Resource, options ...Option) (string, error) {
	t, ok := resources[reflect.TypeOf(r)]
	if !ok {
		return "", fmt.Errorf("unregistered type %T", r)
	}

	if !t.namespaced && namespace != "" {
		return "", fmt.Errorf("type not namespaced")
	}
	if t.namespaced && namespace == "" {
		return "", fmt.Errorf("no namespace provided")
	}

	return urlFor(endpoint, t.apiGroup, t.apiVersion, namespace, t.name, name, options...), nil
}

func resourceListURL(endpoint, namespace string, r ResourceList, options ...Option) (string, error) {
	t, ok := resourceLists[reflect.TypeOf(r)]
	if !ok {
		return "", fmt.Errorf("unregistered type %T", r)
	}

	if !t.namespaced && namespace != "" {
		return "", fmt.Errorf("type not namespaced")
	}

	return urlFor(endpoint, t.apiGroup, t.apiVersion, namespace, t.name, "", options...), nil
}

func resourceWatchURL(endpoint, namespace string, r Resource, options ...Option) (string, error) {
	t, ok := resources[reflect.TypeOf(r)]
	if !ok {
		return "", fmt.Errorf("unregistered type %T", r)
	}

	if !t.namespaced && namespace != "" {
		return "", fmt.Errorf("type not namespaced")
	}

	url := urlFor(endpoint, t.apiGroup, t.apiVersion, namespace, t.name, "", options...)
	if strings.Contains(url, "?") {
		url = url + "&watch=true"
	} else {
		url = url + "?watch=true"
	}
	return url, nil
}
