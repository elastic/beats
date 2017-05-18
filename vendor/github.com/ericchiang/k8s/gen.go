// +build ignore

package main

import (
	"bytes"
	"errors"
	"fmt"
	"go/types"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"sort"
	"strings"
	"text/template"

	"golang.org/x/tools/go/loader"
)

func main() {
	if err := load(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(2)
	}
}

func isInterface(obj interface{}) (*types.Interface, bool) {
	switch obj := obj.(type) {
	case *types.TypeName:
		return isInterface(obj.Type())
	case *types.Named:
		return isInterface(obj.Underlying())
	case *types.Interface:
		return obj, true
	default:
		return nil, false
	}
}

type Resource struct {
	Name       string
	Namespaced bool
	HasList    bool
	Pluralized string
}

type byName []Resource

func (n byName) Len() int           { return len(n) }
func (n byName) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }
func (n byName) Less(i, j int) bool { return n[i].Name < n[j].Name }

type Package struct {
	Name       string
	APIGroup   string
	APIVersion string
	ImportPath string
	ImportName string
	Resources  []Resource
}

type byGroup []Package

func (r byGroup) Len() int      { return len(r) }
func (r byGroup) Swap(i, j int) { r[i], r[j] = r[j], r[i] }

func (r byGroup) Less(i, j int) bool {
	if r[i].APIGroup != r[j].APIGroup {
		return r[i].APIGroup < r[j].APIGroup
	}
	return r[i].APIVersion < r[j].APIVersion
}

// Incorrect but this is basically what Kubernetes does.
func pluralize(s string) string {
	switch {
	case strings.HasSuffix(s, "points"):
		// NOTE: the k8s "endpoints" resource is already pluralized
		return s
	case strings.HasSuffix(s, "s"):
		return s + "es"
	case strings.HasSuffix(s, "y"):
		return s[:len(s)-1] + "ies"
	default:
		return s + "s"
	}
}

var tmpl = template.Must(template.New("").Funcs(template.FuncMap{
	"pluralize": pluralize,
}).Parse(`
// {{ .Name }} returns a client for interacting with the {{ .APIGroup }}/{{ .APIVersion }} API group.
func (c *Client) {{ .Name }}() *{{ .Name }} {
	return &{{ .Name }}{c}
}

// {{ .Name }} is a client for interacting with the {{ .APIGroup }}/{{ .APIVersion }} API group.
type {{ .Name }} struct {
	client *Client
}
{{ range $i, $r := .Resources }}
func (c *{{ $.Name }}) Create{{ $r.Name }}(ctx context.Context, obj *{{ $.ImportName }}.{{ $r.Name }}) (*{{ $.ImportName }}.{{ $r.Name }}, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !{{ $r.Namespaced }} && ns != ""{
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if {{ $r.Namespaced }} {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("{{ $.APIGroup }}", "{{ $.APIVersion }}", ns, "{{ $r.Pluralized }}", "")
	resp := new({{ $.ImportName }}.{{ $r.Name }})
	err := c.client.create(ctx, pbCodec, "POST", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *{{ $.Name }}) Update{{ $r.Name }}(ctx context.Context, obj *{{ $.ImportName }}.{{ $r.Name }}) (*{{ $.ImportName }}.{{ $r.Name }}, error) {
	md := obj.GetMetadata()
	if md.Name != nil && *md.Name == "" {
		return nil, fmt.Errorf("no name for given object")
	}

	ns := ""
	if md.Namespace != nil {
		ns = *md.Namespace
	}
	if !{{ $r.Namespaced }} && ns != "" {
		return nil, fmt.Errorf("resource isn't namespaced")
	}

	if {{ $r.Namespaced }} {
		if ns == "" {
			return nil, fmt.Errorf("no resource namespace provided")
		}
		md.Namespace = &ns
	}
	url := c.client.urlFor("{{ $.APIGroup }}", "{{ $.APIVersion }}", *md.Namespace, "{{ $r.Pluralized }}", *md.Name)
	resp := new({{ $.ImportName }}.{{ $r.Name }})
	err := c.client.create(ctx, pbCodec, "PUT", url, obj, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *{{ $.Name }}) Delete{{ $r.Name }}(ctx context.Context, name string{{ if $r.Namespaced }}, namespace string{{ end }}) error {
	if name == "" {
		return fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("{{ $.APIGroup }}", "{{ $.APIVersion }}", {{ if $r.Namespaced }}namespace{{ else }}AllNamespaces{{ end }}, "{{ $r.Pluralized }}", name)
	return c.client.delete(ctx, pbCodec, url)
}

func (c *{{ $.Name }}) Get{{ $r.Name }}(ctx context.Context, name{{ if $r.Namespaced }}, namespace{{ end }} string) (*{{ $.ImportName }}.{{ $r.Name }}, error) {
	if name == "" {
		return nil, fmt.Errorf("create: no name for given object")
	}
	url := c.client.urlFor("{{ $.APIGroup }}", "{{ $.APIVersion }}", {{ if $r.Namespaced }}namespace{{ else }}AllNamespaces{{ end }}, "{{ $r.Pluralized }}", name)
	resp := new({{ $.ImportName }}.{{ $r.Name }})
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

{{- if $r.HasList }}

type {{ $.Name }}{{ $r.Name }}Watcher struct {
	watcher *watcher
}

func (w *{{ $.Name }}{{ $r.Name }}Watcher) Next() (*versioned.Event, *{{ $.ImportName }}.{{ $r.Name }}, error) {
	event, unknown, err := w.watcher.next()
	if err != nil {
		return nil, nil, err
	}
	resp := new({{ $.ImportName }}.{{ $r.Name }})
	if err := proto.Unmarshal(unknown.Raw, resp); err != nil {
		return nil, nil, err
	}
	return event, resp, nil
}

func (w *{{ $.Name }}{{ $r.Name }}Watcher) Close() error {
	return w.watcher.Close()
}

func (c *{{ $.Name }}) Watch{{ $r.Name | pluralize }}(ctx context.Context{{ if $r.Namespaced }}, namespace string{{ end }}, options ...Option) (*{{ $.Name }}{{ $r.Name }}Watcher, error) {
	url := c.client.urlFor("{{ $.APIGroup }}", "{{ $.APIVersion }}", {{ if $r.Namespaced }}namespace{{ else }}AllNamespaces{{ end }}, "{{ $r.Pluralized }}", "", options...)
	watcher, err := c.client.watch(ctx, url)
	if err != nil {
		return nil, err
	}
	return &{{ $.Name }}{{ $r.Name }}Watcher{watcher}, nil
}

func (c *{{ $.Name }}) List{{ $r.Name | pluralize }}(ctx context.Context{{ if $r.Namespaced }}, namespace string{{ end }}, options ...Option) (*{{ $.ImportName }}.{{ $r.Name }}List, error) {
	url := c.client.urlFor("{{ $.APIGroup }}", "{{ $.APIVersion }}", {{ if $r.Namespaced }}namespace{{ else }}AllNamespaces{{ end }}, "{{ $r.Pluralized }}", "", options...)
	resp := new({{ $.ImportName }}.{{ $r.Name }}List)
	if err := c.client.get(ctx, pbCodec, url, resp); err != nil {
		return nil, err
	}
	return resp, nil
}{{ end }}
{{ end }}
`))

var (
	apiGroupName = map[string]string{
		"authentication": "authentication.k8s.io",
		"authorization":  "authorization.k8s.io",
		"certificates":   "certificates.k8s.io",
		"rbac":           "rbac.authorization.k8s.io",
		"storage":        "storage.k8s.io",
	}
	notNamespaced = map[string]bool{
		"ClusterRole":        true,
		"ClusterRoleBinding": true,

		"ComponentStatus":  true,
		"Node":             true,
		"Namespace":        true,
		"PersistentVolume": true,

		"PodSecurityPolicy":  true,
		"ThirdPartyResource": true,

		"CertificateSigningRequest": true,

		"TokenReview": true,

		"SubjectAccessReview":     true,
		"SelfSubjectAccessReview": true,

		"ImageReview": true,

		"StorageClass": true,
	}
)

func clientName(apiGroup, apiVersion string) string {
	switch apiGroup {
	case "":
		apiGroup = "Core"
	case "rbac":
		apiGroup = "RBAC"
	default:
		apiGroup = strings.Title(apiGroup)
	}
	r := strings.NewReplacer("alpha", "Alpha", "beta", "Beta")
	return apiGroup + r.Replace(strings.Title(apiVersion))
}

func load() error {
	out, err := exec.Command("go", "list", "./...").CombinedOutput()
	if err != nil {
		return fmt.Errorf("go list: %v %s", err, out)
	}

	var conf loader.Config
	if _, err := conf.FromArgs(strings.Fields(string(out)), false); err != nil {
		return fmt.Errorf("from args: %v", err)
	}

	prog, err := conf.Load()
	if err != nil {
		return fmt.Errorf("load: %v", err)
	}
	thisPkg, ok := prog.Imported["github.com/ericchiang/k8s"]
	if !ok {
		return errors.New("could not find this package")
	}

	// Types defined in tpr.go. It's hacky, but to "load" interfaces as their
	// go/types equilvalent, we either have to:
	//
	//   * Define them in code somewhere (what we're doing here).
	//   * Manually construct them using go/types (blah).
	//   * Parse them from an inlined string (doesn't work in combination with other pkgs).
	//
	var interfaces []*types.Interface
	for _, s := range []string{"object", "after16Object"} {
		obj := thisPkg.Pkg.Scope().Lookup(s)
		if obj == nil {
			return errors.New("failed to lookup object interface")
		}
		intr, ok := isInterface(obj)
		if !ok {
			return errors.New("failed to convert to interface")
		}
		interfaces = append(interfaces, intr)
	}

	var pkgs []Package
	for name, pkgInfo := range prog.Imported {
		pkg := Package{
			APIVersion: path.Base(name),
			APIGroup:   path.Base(path.Dir(name)),
			ImportPath: name,
		}
		pkg.ImportName = pkg.APIGroup + pkg.APIVersion

		if pkg.APIGroup == "api" {
			pkg.APIGroup = ""
		}

		pkg.Name = clientName(pkg.APIGroup, pkg.APIVersion)
		if name, ok := apiGroupName[pkg.APIGroup]; ok {
			pkg.APIGroup = name
		}

		for _, obj := range pkgInfo.Defs {
			tn, ok := obj.(*types.TypeName)
			if !ok {
				continue
			}
			impl := false
			for _, intr := range interfaces {
				impl = impl || types.Implements(types.NewPointer(tn.Type()), intr)
			}
			if !impl {
				continue
			}
			if tn.Name() == "JobTemplateSpec" {
				continue
			}

			pkg.Resources = append(pkg.Resources, Resource{
				Name:       tn.Name(),
				Pluralized: pluralize(strings.ToLower(tn.Name())),
				HasList:    pkgInfo.Pkg.Scope().Lookup(tn.Name()+"List") != nil,
				Namespaced: !notNamespaced[tn.Name()],
			})
		}
		pkgs = append(pkgs, pkg)
	}

	sort.Sort(byGroup(pkgs))

	buff := new(bytes.Buffer)
	buff.WriteString("package k8s\n\n")
	buff.WriteString("import (\n")
	buff.WriteString("\t\"context\"\n")
	buff.WriteString("\t\"fmt\"\n\n")
	for _, pkg := range pkgs {
		if len(pkg.Resources) == 0 {
			continue
		}
		fmt.Fprintf(buff, "\t%s \"%s\"\n", pkg.ImportName, pkg.ImportPath)
	}
	fmt.Fprintf(buff, "\t%q\n", "github.com/ericchiang/k8s/watch/versioned")
	fmt.Fprintf(buff, "\t%q\n", "github.com/golang/protobuf/proto")
	buff.WriteString(")\n")

	for _, pkg := range pkgs {
		sort.Sort(byName(pkg.Resources))
		for _, resource := range pkg.Resources {
			fmt.Println(pkg.APIGroup, pkg.APIVersion, resource.Name)
		}
		if len(pkg.Resources) != 0 {
			if err := tmpl.Execute(buff, pkg); err != nil {
				return fmt.Errorf("execute: %v", err)
			}
		}
	}
	return ioutil.WriteFile("types.go", buff.Bytes(), 0644)
}
