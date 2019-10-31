module github.com/elastic/beats

go 1.13

replace (
	github.com/awslabs/goformation v1.2.1 => github.com/awslabs/goformation v1.2.1

	github.com/docker/docker v1.4.2-0.20190822205725-ed20165a37b4 => github.com/docker/engine v1.4.2-0.20190822205725-ed20165a37b4

	github.com/dop251/goja v0.0.0-20190912223329-aa89e6a4c733 => github.com/andrewkroh/goja v0.0.0-20190128172624-dd2ac4456e20

	github.com/elastic/go-perf v0.0.0-20190822174212-9bc9b58a3de9 => github.com/michalpristas/go-perf v0.0.0-20191031073750-9e95cbdc2071

	github.com/fsnotify/fsevents v0.0.0-20181029231046-e1d381a4d270 => github.com/elastic/fsevents v0.0.0-20181029231046-e1d381a4d270

	github.com/fsnotify/fsnotify v1.4.7 => github.com/adriansr/fsnotify v1.4.7

	github.com/google/gopacket v1.1.17 => github.com/adriansr/gopacket v1.1.17
)

require (
	4d63.com/tz v1.1.0
	cloud.google.com/go/pubsub v1.0.1
	github.com/Azure/azure-sdk-for-go v35.0.0+incompatible
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/Azure/go-autorest v11.1.2+incompatible
	github.com/Azure/go-autorest/tracing v0.5.0 // indirect
	github.com/Microsoft/go-winio v0.4.14
	github.com/OneOfOne/xxhash v1.2.5
	github.com/Shopify/sarama v1.24.0
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d
	github.com/aerospike/aerospike-client-go v2.4.0+incompatible
	github.com/andrewkroh/sys v0.0.0-20151128191922-287798fe3e43
	github.com/antlr/antlr4 v0.0.0-20191011202612-ad2bd05285ca
	github.com/armon/go-socks5 v0.0.0-20160902184237-e75332964ef5
	github.com/aws/aws-lambda-go v1.13.2
	github.com/aws/aws-sdk-go-v2 v0.15.0
	github.com/awslabs/goformation v1.2.1
	github.com/blakesmith/ar v0.0.0-20190502131153-809d4375e1fb
	github.com/bsm/sarama-cluster v2.1.15+incompatible
	github.com/cavaliercoder/badio v0.0.0-20160213150051-ce5280129e9e // indirect
	github.com/cavaliercoder/go-rpm v0.0.0-20190131055624-7a9c54e3d83e
	github.com/coreos/bbolt v1.3.3
	github.com/coreos/go-systemd v0.0.0-20190719114852-fd7a80b32e1f
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f
	github.com/davecgh/go-spew v1.1.1
	github.com/denisenkom/go-mssqldb v0.0.0-20191001013358-cfbb681360f0
	github.com/digitalocean/go-libvirt v0.0.0-20190715144809-7b622097a793
	github.com/dimchansky/utfbom v1.1.0 // indirect
	github.com/dlclark/regexp2 v1.2.0 // indirect
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/docker v1.4.2-0.20190822205725-ed20165a37b4
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-units v0.4.0 // indirect
	github.com/dop251/goja v0.0.0-20190912223329-aa89e6a4c733
	github.com/dop251/goja_nodejs v0.0.0-20171011081505-adff31b136e6
	github.com/dustin/go-humanize v1.0.0
	github.com/elastic/ecs v1.2.0
	github.com/elastic/go-libaudit v0.4.0
	github.com/elastic/go-lookslike v0.3.0
	github.com/elastic/go-lumber v0.1.0
	github.com/elastic/go-perf v0.0.0-20190822174212-9bc9b58a3de9
	github.com/elastic/go-seccomp-bpf v1.1.0
	github.com/elastic/go-structform v0.0.6
	github.com/elastic/go-sysinfo v1.1.0
	github.com/elastic/go-txfile v0.0.6
	github.com/elastic/go-ucfg v0.7.0
	github.com/elastic/gosigar v0.10.5
	github.com/fatih/color v1.7.0
	github.com/fsnotify/fsevents v0.0.0-20181029231046-e1d381a4d270
	github.com/fsnotify/fsnotify v1.4.7
	github.com/garyburd/redigo v1.6.0
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/go-sourcemap/sourcemap v2.1.2+incompatible // indirect
	github.com/go-sql-driver/mysql v1.4.1
	github.com/gocarina/gocsv v0.0.0-20190927101021-3ecffd272576
	github.com/gofrs/flock v0.7.1 // indirect
	github.com/gofrs/uuid v3.2.0+incompatible
	github.com/golang/protobuf v1.3.2
	github.com/golang/snappy v0.0.1
	github.com/google/flatbuffers v1.11.0
	github.com/google/go-cmp v0.3.1
	github.com/google/gopacket v1.1.17
	github.com/gorhill/cronexpr v0.0.0-20180427100037-88b0669f7d75
	github.com/gorilla/mux v1.7.3 // indirect
	github.com/hashicorp/go-multierror v1.0.0
	github.com/insomniacslk/dhcp v0.0.0-20191025184527-fe3f5c4e2b53
	github.com/joeshaw/multierror v0.0.0-20140124173710-69b34d4ec901
	github.com/jstemmer/go-junit-report v0.9.1
	github.com/klauspost/cpuid v1.2.1 // indirect
	github.com/lib/pq v1.2.0
	github.com/magefile/mage v1.9.0
	github.com/mattn/go-colorable v0.1.4
	github.com/miekg/dns v1.1.16
	github.com/mitchellh/hashstructure v1.0.0
	github.com/mitchellh/mapstructure v1.1.2
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/oklog/ulid v1.3.1
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/pkg/errors v0.8.1
	github.com/pmezard/go-difflib v1.0.0
	github.com/prometheus/client_model v0.0.0-20190812154241-14fe0d1b01d4
	github.com/prometheus/common v0.7.0
	github.com/prometheus/procfs v0.0.5
	github.com/rcrowley/go-metrics v0.0.0-20190826022208-cac0b30c2563
	github.com/samuel/go-thrift v0.0.0-20190219015601-e8b6b52668fe
	github.com/sanathkr/yaml v1.0.0 // indirect
	github.com/shirou/gopsutil v2.19.9+incompatible
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.4.0
	github.com/theckman/go-flock v0.7.1
	github.com/tsg/gopacket v0.0.0-20190320122513-dd3d0e41124a
	github.com/u-root/u-root v6.0.0+incompatible // indirect
	github.com/urso/ecslog v0.0.0-20190806172324-49c373406d28
	github.com/vmware/govmomi v0.21.0
	github.com/yuin/gopher-lua v0.0.0-20190514113301-1cd887cd7036 // indirect
	go.etcd.io/bbolt v1.3.3 // indirect
	go.uber.org/atomic v1.5.0
	go.uber.org/multierr v1.3.0
	go.uber.org/zap v1.12.0
	golang.org/x/crypto v0.0.0-20191029031824-8986dd9e96cf
	golang.org/x/net v0.0.0-20191028085509-fe3aa8a45271
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	golang.org/x/sys v0.0.0-20190826190057-c7b8b68b1456
	golang.org/x/text v0.3.2
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0
	golang.org/x/tools v0.0.0-20191030062658-86caa796c7ab
	google.golang.org/api v0.13.0
	google.golang.org/grpc v1.24.0
	gopkg.in/goracle.v2 v2.22.0
	gopkg.in/inf.v0 v0.9.1
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22
	gopkg.in/yaml.v2 v2.2.4
	gotest.tools v2.2.0+incompatible // indirect
	howett.net/plist v0.0.0-20181124034731-591f970eefbb
	k8s.io/api v0.0.0-20191025225708-5524a3672fbb
	k8s.io/apimachinery v0.0.0-20191025225532-af6325b3a843
	k8s.io/client-go v12.0.0+incompatible
)
