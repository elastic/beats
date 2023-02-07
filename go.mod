module github.com/elastic/elastic-agent-autodiscover

go 1.19

require (
	github.com/docker/docker v20.10.12+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/elastic/elastic-agent-libs v0.2.11
	github.com/magefile/mage v1.12.1
	github.com/stretchr/testify v1.7.0
	golang.org/x/net v0.0.0-20220225172249-27dd8689420f
	k8s.io/api v0.23.4
	k8s.io/apimachinery v0.23.4
	k8s.io/client-go v0.23.4
)

require (
	github.com/containerd/containerd v1.5.13 // indirect
	github.com/docker/distribution v2.8.0+incompatible // indirect
	github.com/elastic/go-licenser v0.4.0
	github.com/elastic/go-ucfg v0.8.5
	github.com/morikuni/aec v1.0.0 // indirect
	go.elastic.co/go-licence-detector v0.5.0
	gopkg.in/yaml.v2 v2.4.0
)
