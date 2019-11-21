module github.com/elastic/beats/x-pack/functionbeat/provider/gcp/pubsub

go 1.11

require (
	cloud.google.com/go/pubsub v1.1.0 // indirect
	github.com/Shopify/sarama v1.24.1 // indirect
	github.com/dlclark/regexp2 v1.2.0 // indirect
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/docker v1.13.1 // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/docker/libtrust v0.0.0-20160708172513-aabc10ec26b7 // indirect
	github.com/dop251/goja v0.0.0-20190912223329-aa89e6a4c733 // indirect
	github.com/dop251/goja_nodejs v0.0.0-20171011081505-adff31b136e6 // indirect
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/elastic/beats v0.0.0-20191121183958-7b7ce7a8c4f5 // indirect
	github.com/elastic/ecs v1.2.0 // indirect
	github.com/elastic/go-lumber v0.1.0 // indirect
	github.com/elastic/go-seccomp-bpf v1.1.0 // indirect
	github.com/elastic/go-structform v0.0.6 // indirect
	github.com/elastic/go-sysinfo v1.1.1 // indirect
	github.com/elastic/go-txfile v0.0.6 // indirect
	github.com/elastic/gosigar v0.10.5 // indirect
	github.com/fatih/color v1.7.0 // indirect
	github.com/garyburd/redigo v1.6.0 // indirect
	github.com/go-sourcemap/sourcemap v2.1.2+incompatible // indirect
	github.com/gofrs/uuid v3.2.0+incompatible // indirect
	github.com/imdario/mergo v0.3.8 // indirect
	github.com/klauspost/cpuid v1.2.1 // indirect
	github.com/mattn/go-colorable v0.1.4 // indirect
	github.com/miekg/dns v1.1.22 // indirect
	github.com/mitchellh/hashstructure v1.0.0 // indirect
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20190826022208-cac0b30c2563 // indirect
	github.com/sirupsen/logrus v1.4.2 // indirect
	go.uber.org/multierr v1.4.0 // indirect
	go.uber.org/zap v1.13.0 // indirect
	gopkg.in/yaml.v2 v2.2.7 // indirect
	k8s.io/api v0.0.0-20191025225708-5524a3672fbb
	k8s.io/apimachinery v0.0.0-20191025225532-af6325b3a843
	k8s.io/client-go v12.0.0+incompatible
)

replace (
	github.com/Sirupsen/logrus => github.com/sirupsen/logrus v1.4.2
	github.com/docker/docker => github.com/docker/engine v0.0.0-20190717161051-705d9623b7c1
	github.com/dop251/goja v0.0.0-20190912223329-aa89e6a4c733 => github.com/andrewkroh/goja v0.0.0-20190128172624-dd2ac4456e20
	github.com/elastic/go-perf v0.0.0-20190822174212-9bc9b58a3de9 => github.com/michalpristas/go-perf v0.0.0-20191031073750-9e95cbdc2071
	github.com/fsnotify/fsevents v0.0.0-20181029231046-e1d381a4d270 => github.com/elastic/fsevents v0.0.0-20181029231046-e1d381a4d270
	github.com/fsnotify/fsnotify v1.4.7 => github.com/adriansr/fsnotify v1.4.7
	github.com/google/gopacket v1.1.17 => github.com/adriansr/gopacket v1.1.17
)
