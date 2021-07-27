module github.com/elastic/beats/v7

go 1.16

require (
	4d63.com/tz v1.1.1-0.20191124060701-6d37baae851b
	cloud.google.com/go v0.51.0
	cloud.google.com/go/bigquery v1.0.1
	cloud.google.com/go/pubsub v1.0.1
	cloud.google.com/go/storage v1.0.0
	code.cloudfoundry.org/go-diodes v0.0.0-20190809170250-f77fb823c7ee // indirect
	code.cloudfoundry.org/go-loggregator v7.4.0+incompatible
	code.cloudfoundry.org/rfc5424 v0.0.0-20180905210152-236a6d29298a // indirect
	github.com/Azure/azure-event-hubs-go/v3 v3.1.2
	github.com/Azure/azure-sdk-for-go v37.1.0+incompatible
	github.com/Azure/azure-storage-blob-go v0.8.0
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/Azure/go-autorest/autorest v0.9.6
	github.com/Azure/go-autorest/autorest/adal v0.8.2
	github.com/Azure/go-autorest/autorest/azure/auth v0.4.2
	github.com/Azure/go-autorest/autorest/date v0.2.0
	github.com/Masterminds/semver v1.4.2
	github.com/Microsoft/go-winio v0.4.15-0.20190919025122-fc70bd9a86b5
	github.com/Shopify/sarama v1.27.0
	github.com/StackExchange/wmi v0.0.0-20170221213301-9f32b5905fd6
	github.com/aerospike/aerospike-client-go v1.27.1-0.20170612174108-0f3b54da6bdc
	github.com/akavel/rsrc v0.8.0 // indirect
	github.com/andrewkroh/sys v0.0.0-20151128191922-287798fe3e43
	github.com/antlr/antlr4 v0.0.0-20200820155224-be881fa6b91d
	github.com/apoydence/eachers v0.0.0-20181020210610-23942921fe77 // indirect
	github.com/armon/go-socks5 v0.0.0-20160902184237-e75332964ef5
	github.com/aws/aws-lambda-go v1.6.0
	github.com/aws/aws-sdk-go-v2 v0.9.0
	github.com/awslabs/goformation/v4 v4.1.0
	github.com/blakesmith/ar v0.0.0-20150311145944-8bd4349a67f2
	github.com/bsm/sarama-cluster v2.1.14-0.20180625083203-7e67d87a6b3f+incompatible
	github.com/cavaliercoder/badio v0.0.0-20160213150051-ce5280129e9e // indirect
	github.com/cavaliercoder/go-rpm v0.0.0-20190131055624-7a9c54e3d83e
	github.com/cespare/xxhash/v2 v2.1.1
	github.com/cloudfoundry-community/go-cfclient v0.0.0-20190808214049-35bcce23fc5f
	github.com/cloudfoundry/noaa v2.1.0+incompatible
	github.com/cloudfoundry/sonde-go v0.0.0-20171206171820-b33733203bb4
	github.com/containerd/fifo v0.0.0-20190816180239-bda0ff6ed73c
	github.com/coreos/go-systemd/v22 v22.0.0
	github.com/coreos/pkg v0.0.0-20180108230652-97fdf19511ea
	github.com/davecgh/go-xdr v0.0.0-20161123171359-e6a2ba005892 // indirect
	github.com/denisenkom/go-mssqldb v0.0.0-20200206145737-bbfc9a55622e
	github.com/devigned/tab v0.1.2-0.20190607222403-0c15cf42f9a2 // indirect
	github.com/dgraph-io/badger/v2 v2.2007.3-0.20201012072640-f5a7e0a1c83b
	github.com/dgrijalva/jwt-go v3.2.1-0.20190620180102-5e25c22bd5d6+incompatible // indirect
	github.com/digitalocean/go-libvirt v0.0.0-20180301200012-6075ea3c39a1
	github.com/dlclark/regexp2 v1.1.7-0.20171009020623-7632a260cbaf // indirect
	github.com/docker/docker v1.4.2-0.20170802015333-8af4db6f002a
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/docker/go-plugins-helpers v0.0.0-20181025120712-1e6269c305b8
	github.com/docker/go-units v0.4.0
	github.com/dolmen-go/contextio v0.0.0-20200217195037-68fc5150bcd5
	github.com/dop251/goja v0.0.0-20200831102558-9af81ddcf0e1
	github.com/dop251/goja_nodejs v0.0.0-20171011081505-adff31b136e6
	github.com/dustin/go-humanize v1.0.0
	github.com/eapache/go-resiliency v1.2.0
	github.com/eclipse/paho.mqtt.golang v1.2.1-0.20200121105743-0d940dd29fd2
	github.com/elastic/ecs v1.10.0
	github.com/elastic/elastic-agent-client/v7 v7.0.0-20210308165121-7dd05ee2b5a5
	github.com/elastic/go-concert v0.1.0
	github.com/elastic/go-libaudit/v2 v2.2.0
	github.com/elastic/go-licenser v0.3.1
	github.com/elastic/go-lookslike v0.3.0
	github.com/elastic/go-lumber v0.1.0
	github.com/elastic/go-perf v0.0.0-20191212140718-9c656876f595
	github.com/elastic/go-seccomp-bpf v1.1.0
	github.com/elastic/go-structform v0.0.9
	github.com/elastic/go-sysinfo v1.7.0
	github.com/elastic/go-txfile v0.0.7
	github.com/elastic/go-ucfg v0.8.3
	github.com/elastic/go-windows v1.0.1
	github.com/elastic/gosigar v0.14.1
	github.com/fatih/color v1.9.0
	github.com/fsnotify/fsevents v0.1.1
	github.com/fsnotify/fsnotify v1.4.9
	github.com/go-ole/go-ole v1.2.5-0.20190920104607-14974a1cf647 // indirect
	github.com/go-sourcemap/sourcemap v2.1.2+incompatible // indirect
	github.com/go-sql-driver/mysql v1.4.1
	github.com/go-test/deep v1.0.7
	github.com/gocarina/gocsv v0.0.0-20170324095351-ffef3ffc77be
	github.com/godbus/dbus v0.0.0-20190422162347-ade71ed3457e
	github.com/godror/godror v0.10.4
	github.com/gofrs/flock v0.7.2-0.20190320160742-5135e617513b
	github.com/gofrs/uuid v3.3.0+incompatible
	github.com/gogo/protobuf v1.3.1
	github.com/golang/protobuf v1.4.3
	github.com/golang/snappy v0.0.1
	github.com/gomodule/redigo v1.8.3
	github.com/google/flatbuffers v1.7.2-0.20170925184458-7a6b2bf521e9
	github.com/google/go-cmp v0.5.2
	github.com/google/gopacket v1.1.18-0.20191009163724-0ad7f2610e34
	github.com/google/uuid v1.1.2
	github.com/gorhill/cronexpr v0.0.0-20180427100037-88b0669f7d75
	github.com/gorilla/mux v1.7.2
	github.com/grpc-ecosystem/grpc-gateway v1.13.0 // indirect
	github.com/h2non/filetype v1.1.1
	github.com/hashicorp/go-multierror v1.1.0
	github.com/hashicorp/go-retryablehttp v0.6.6
	github.com/hashicorp/golang-lru v0.5.4
	github.com/hashicorp/nomad/api v0.0.0-20201203164818-6318a8ac7bf8
	github.com/hectane/go-acl v0.0.0-20190604041725-da78bae5fc95
	github.com/insomniacslk/dhcp v0.0.0-20180716145214-633285ba52b2
	github.com/jarcoal/httpmock v1.0.4
	github.com/jmoiron/sqlx v1.2.1-0.20190826204134-d7d95172beb5
	github.com/joeshaw/multierror v0.0.0-20140124173710-69b34d4ec901
	github.com/jonboulle/clockwork v0.2.2
	github.com/josephspurrier/goversioninfo v0.0.0-20190209210621-63e6d1acd3dd
	github.com/jpillora/backoff v1.0.0 // indirect
	github.com/kardianos/service v1.1.0
	github.com/kolide/osquery-go v0.0.0-20200604192029-b019be7063ac
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/lib/pq v1.1.2-0.20190507191818-2ff3cb3adc01
	github.com/magefile/mage v1.11.0
	github.com/mailru/easyjson v0.7.1 // indirect
	github.com/mattn/go-colorable v0.1.6
	github.com/mattn/go-ieproxy v0.0.0-20191113090002-7c0f6868bffe // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/miekg/dns v1.1.25
	github.com/mitchellh/gox v1.0.1
	github.com/mitchellh/hashstructure v0.0.0-20170116052023-ab25296c0f51
	github.com/mitchellh/mapstructure v1.3.3
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/oklog/ulid v1.3.1
	github.com/opencontainers/go-digest v1.0.0-rc1.0.20190228220655-ac19fd6e7483 // indirect
	github.com/opencontainers/image-spec v1.0.2-0.20190823105129-775207bd45b6 // indirect
	github.com/otiai10/copy v1.2.0
	github.com/pierrre/gotestcover v0.0.0-20160517101806-924dca7d15f0
	github.com/pkg/errors v0.9.1
	github.com/pmezard/go-difflib v1.0.0
	github.com/prometheus/client_golang v1.1.1-0.20190913103102-20428fa0bffc // indirect
	github.com/prometheus/client_model v0.0.0-20190812154241-14fe0d1b01d4
	github.com/prometheus/common v0.7.0
	github.com/prometheus/procfs v0.0.11
	github.com/prometheus/prometheus v2.5.0+incompatible
	github.com/rcrowley/go-metrics v0.0.0-20200313005456-10cdbea86bc0
	github.com/samuel/go-parser v0.0.0-20130731160455-ca8abbf65d0e // indirect
	github.com/samuel/go-thrift v0.0.0-20140522043831-2187045faa54
	github.com/sanathkr/yaml v1.0.1-0.20170819201035-0056894fa522 // indirect
	github.com/satori/go.uuid v1.2.0 // indirect
	github.com/shirou/gopsutil v3.20.12+incompatible
	github.com/shopspring/decimal v1.2.0
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/objx v0.2.0 // indirect
	github.com/stretchr/testify v1.7.0
	github.com/tsg/go-daemon v0.0.0-20200207173439-e704b93fd89b
	github.com/tsg/gopacket v0.0.0-20200626092518-2ab8e397a786
	github.com/ugorji/go/codec v1.1.8
	github.com/urso/sderr v0.0.0-20200210124243-c2a16f3d43ec
	github.com/vmware/govmomi v0.0.0-20170802214208-2cad15190b41
	github.com/xdg/scram v0.0.0-20180814205039-7eeb5667e42c
	github.com/yuin/gopher-lua v0.0.0-20170403160031-b402f3114ec7 // indirect
	go.elastic.co/apm v1.8.1-0.20200909061013-2aef45b9cf4b
	go.elastic.co/apm/module/apmelasticsearch v1.7.2
	go.elastic.co/apm/module/apmhttp v1.7.2
	go.elastic.co/ecszap v0.3.0
	go.elastic.co/go-licence-detector v0.4.0
	go.etcd.io/bbolt v1.3.4
	go.uber.org/atomic v1.5.0
	go.uber.org/multierr v1.3.0
	go.uber.org/zap v1.14.0
	golang.org/x/crypto v0.0.0-20210616213533-5ff15b29337e
	golang.org/x/lint v0.0.0-20200130185559-910be7a94367
	golang.org/x/net v0.0.0-20210226172049-e18ecbb05110
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/sync v0.0.0-20200317015054-43a5402ce75a
	golang.org/x/sys v0.0.0-20210615035016-665e8c7367d1
	golang.org/x/text v0.3.5
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0
	golang.org/x/tools v0.0.0-20200731060945-b5fad4ed8dd6
	google.golang.org/api v0.15.0
	google.golang.org/genproto v0.0.0-20210303154014-9728d6b83eeb
	google.golang.org/grpc v1.29.1
	google.golang.org/protobuf v1.25.0
	gopkg.in/inf.v0 v0.9.1
	gopkg.in/jcmturner/gokrb5.v7 v7.5.0
	gopkg.in/mgo.v2 v2.0.0-20160818020120-3f83fa500528
	gopkg.in/yaml.v2 v2.3.0
	gotest.tools v2.2.0+incompatible
	gotest.tools/gotestsum v0.6.0
	howett.net/plist v0.0.0-20181124034731-591f970eefbb
	k8s.io/api v0.19.4
	k8s.io/apimachinery v0.19.4
	k8s.io/client-go v0.19.4
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v12.2.0+incompatible
	github.com/Microsoft/go-winio => github.com/bi-zone/go-winio v0.4.15
	github.com/Shopify/sarama => github.com/elastic/sarama v1.19.1-0.20210120173147-5c8cb347d877
	github.com/cucumber/godog => github.com/cucumber/godog v0.8.1
	github.com/docker/docker => github.com/docker/engine v0.0.0-20191113042239-ea84732a7725
	github.com/docker/go-plugins-helpers => github.com/elastic/go-plugins-helpers v0.0.0-20200207104224-bdf17607b79f
	github.com/dop251/goja => github.com/andrewkroh/goja v0.0.0-20190128172624-dd2ac4456e20
	github.com/dop251/goja_nodejs => github.com/dop251/goja_nodejs v0.0.0-20171011081505-adff31b136e6
	github.com/fsnotify/fsevents => github.com/elastic/fsevents v0.0.0-20181029231046-e1d381a4d270
	github.com/fsnotify/fsnotify => github.com/adriansr/fsnotify v0.0.0-20180417234312-c9bbe1f46f1d
	github.com/google/gopacket => github.com/adriansr/gopacket v1.1.18-0.20200327165309-dd62abfa8a41
	github.com/insomniacslk/dhcp => github.com/elastic/dhcp v0.0.0-20200227161230-57ec251c7eb3 // indirect
	github.com/kardianos/service => github.com/blakerouse/service v1.1.1-0.20200924160513-057808572ffa
	github.com/tonistiigi/fifo => github.com/containerd/fifo v0.0.0-20190816180239-bda0ff6ed73c
	golang.org/x/tools => golang.org/x/tools v0.0.0-20200602230032-c00d67ef29d0 // release 1.14
)
