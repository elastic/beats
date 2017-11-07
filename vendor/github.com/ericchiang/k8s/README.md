# A simple Go client for Kubernetes

[![GoDoc](https://godoc.org/github.com/ericchiang/k8s?status.svg)](https://godoc.org/github.com/ericchiang/k8s)

A slimmed down Go client generated using Kubernetes' new [protocol buffer][protobuf] support. This package behaves similarly to [official Kubernetes' Go client][client-go], but only imports two external dependencies.

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/ericchiang/k8s"
)

func main() {
    client, err := k8s.NewInClusterClient()
    if err != nil {
        log.Fatal(err)
    }

    nodes, err := client.CoreV1().ListNodes(context.Background())
    if err != nil {
        log.Fatal(err)
    }
    for _, node := range nodes.Items {
        fmt.Printf("name=%q schedulable=%t\n", *node.Metadata.Name, !*node.Spec.Unschedulable)
    }
}
```

## Should I use this or client-go?

client-go is a framework for building production ready controllers, components that regularly watch API resources and push the system towards a desired state. If you're writing a program that watches several resources in a loop for long durations, client-go's informers framework is a battle tested solution which will scale with the size of the cluster.

This client should be used by programs that just need to talk to the Kubernetes API without prescriptive solutions for caching, reconciliation on failures, or work queues. This often includes components are relatively Kubernetes agnostic, but use the Kubernetes API for small tasks when running in Kubernetes. For example, performing leader election or persisting small amounts of state in annotations or configmaps.

TL;DR - Use client-go if you're writing a controller.

## Requirements

* Go 1.7+ (this package uses "context" features added in 1.7)
* Kubernetes 1.3+ (protobuf support was added in 1.3)
* [github.com/golang/protobuf/proto][go-proto] (protobuf serialization)
* [golang.org/x/net/http2][go-http2] (HTTP/2 support)

## Versioned supported

This client supports every API group version present since 1.3.

## Usage

### Namespace

When performing a list or watch operation, the namespace to list or watch in is provided as an argument.

```go
pods, err := core.ListPods(ctx, "custom-namespace") // Pods from the "custom-namespace"
```

A special value `AllNamespaces` indicates that the list or watch should be performed on all cluster resources.

```go
pods, err := core.ListPods(ctx, k8s.AllNamespaces) // Pods in all namespaces.
```

Both in-cluster and out-of-cluster clients are initialized with a primary namespace. This is the recommended value to use when listing or watching.

```go
client, err := k8s.NewInClusterClient()
if err != nil {
    // handle error
}

// List pods in the namespace the client is running in.
pods, err := client.CoreV1().ListPods(ctx, client.Namespace)
```

### Label selectors

Label selectors can be provided to any list operation.

```go
l := new(k8s.LabelSelector)
l.Eq("tier", "production")
l.In("app", "database", "frontend")

pods, err := client.CoreV1().ListPods(ctx, client.Namespace, l.Selector())
```

### Working with resources

Use the generated API types directly to create and modify resources.

```go
import (
    "context"

    "github.com/ericchiang/k8s"
    "github.com/ericchiang/k8s/api/v1"
    metav1 "github.com/ericchiang/k8s/apis/meta/v1"
)

func createConfigMap(client *k8s.Client, name string, values map[string]string) error {
    cm := &v1.ConfigMap{
        Metadata: &metav1.ObjectMeta{
            Name:      &name,
            Namespace: &client.Namespace,
        },
        Data: values,
    }
    // Will return the created configmap as well.
    _, err := client.CoreV1().CreateConfigMap(context.TODO(), cm)
    return err
}
```

API structs use pointers to `int`, `bool`, and `string` types to differentiate between the zero value and an unsupplied one. This package provides [convenience methods][string] for creating pointers to literals of basic types.

### Creating out-of-cluster clients

Out-of-cluster clients can be constructed by either creating an `http.Client` manually or parsing a [`Config`][config] object. The following is an example of creating a client from a kubeconfig:

```go
import (
    "io/ioutil"

    "github.com/ericchiang/k8s"

    "github.com/ghodss/yaml"
)

// loadClient parses a kubeconfig from a file and returns a Kubernetes
// client. It does not support extensions or client auth providers.
func loadClient(kubeconfigPath string) (*k8s.Client, error) {
    data, err := ioutil.ReadFile(kubeconfigPath)
    if err != nil {
        return nil, fmt.Errorf("read kubeconfig: %v", err)
    }

    // Unmarshal YAML into a Kubernetes config object.
    var config k8s.Config
    if err := yaml.Unmarshal(data, &config); err != nil {
        return nil, fmt.Errorf("unmarshal kubeconfig: %v", err)
    }
    return k8s.NewClient(&config)
}
```

### Errors

Errors returned by the Kubernetes API are formatted as [`unversioned.Status`][unversioned-status] objects and surfaced by clients as [`*k8s.APIError`][k8s-error]s. Programs that need to inspect error codes or failure details can use a type cast to access this information.

```go
// createConfigMap creates a configmap in the client's default namespace
// but does not return an error if a configmap of the same name already
// exists.
func createConfigMap(client *k8s.Client, name string, values map[string]string) error {
    cm := &v1.ConfigMap{
        Metadata: &metav1.ObjectMeta{
            Name:      &name,
            Namespace: &client.Namespace,
        },
        Data: values,
    }

    _, err := client.CoreV1().CreateConfigMap(context.TODO(), cm)

    // If an HTTP error was returned by the API server, it will be of type
    // *k8s.APIError. This can be used to inspect the status code.
    if apiErr, ok := err.(*k8s.APIError); ok {
        // Resource already exists. Carry on.
        if apiErr.Code == http.StatusConflict {
            return nil
        }
    }
    return fmt.Errorf("create configmap: %v", err)
}
```

[client-go]: https://github.com/kubernetes/client-go
[go-proto]: https://godoc.org/github.com/golang/protobuf/proto
[go-http2]: https://godoc.org/golang.org/x/net/http2
[protobuf]: https://developers.google.com/protocol-buffers/
[unversioned-status]: https://godoc.org/github.com/ericchiang/k8s/api/unversioned#Status
[k8s-error]: https://godoc.org/github.com/ericchiang/k8s#APIError
[config]: https://godoc.org/github.com/ericchiang/k8s#Config
[string]: https://godoc.org/github.com/ericchiang/k8s#String
