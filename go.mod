module github.com/elastic/beats/v7

go 1.13

require (
	4d63.com/tz v1.1.1-0.20191124060701-6d37baae851b
	cloud.google.com/go v0.51.0
	cloud.google.com/go/pubsub v1.0.1
	cloud.google.com/go/storage v1.0.0
	code.cloudfoundry.org/go-diodes v0.0.0-20190809170250-f77fb823c7ee // indirect
	code.cloudfoundry.org/go-loggregator v7.4.0+incompatible
	code.cloudfoundry.org/rfc5424 v0.0.0-20180905210152-236a6d29298a // indirect
	github.com/Azure/azure-event-hubs-go/v3 v3.1.0
	github.com/Azure/azure-sdk-for-go v37.1.0+incompatible
	github.com/Azure/azure-storage-blob-go v0.8.0
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/Azure/go-autorest/autorest v0.9.4
	github.com/Azure/go-autorest/autorest/adal v0.8.1
	github.com/Azure/go-autorest/autorest/azure/auth v0.4.2
	github.com/Azure/go-autorest/autorest/date v0.2.0
	github.com/Microsoft/go-winio v0.4.15-0.20190919025122-fc70bd9a86b5
	github.com/Shopify/sarama v0.0.0-00010101000000-000000000000
	github.com/StackExchange/wmi v0.0.0-20170221213301-9f32b5905fd6
	github.com/aerospike/aerospike-client-go v1.27.1-0.20170612174108-0f3b54da6bdc
	github.com/akavel/rsrc v0.8.0 // indirect
	github.com/andrewkroh/sys v0.0.0-20151128191922-287798fe3e43
	github.com/antlr/antlr4 v0.0.0-20200225173536-225249fdaef5
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
	github.com/cloudfoundry/sonde-go v0.0.0-20171206171820-b33733203bb4
	github.com/containerd/fifo v0.0.0-20190816180239-bda0ff6ed73c
	github.com/coreos/bbolt v1.3.1-coreos.6.0.20180318001526-af9db2027c98
	github.com/coreos/go-systemd/v22 v22.0.0
	github.com/coreos/pkg v0.0.0-20180108230652-97fdf19511ea
	github.com/davecgh/go-xdr v0.0.0-20161123171359-e6a2ba005892 // indirect
	github.com/denisenkom/go-mssqldb v0.0.0-20181014144952-4e0d7dc8888f
	github.com/devigned/tab v0.1.2-0.20190607222403-0c15cf42f9a2 // indirect
	github.com/dgrijalva/jwt-go v3.2.1-0.20190620180102-5e25c22bd5d6+incompatible // indirect
	github.com/digitalocean/go-libvirt v0.0.0-20180301200012-6075ea3c39a1
	github.com/dlclark/regexp2 v1.1.7-0.20171009020623-7632a260cbaf // indirect
	github.com/docker/docker v1.4.2-0.20170802015333-8af4db6f002a
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/docker/go-plugins-helpers v0.0.0-20181025120712-1e6269c305b8
	github.com/docker/go-units v0.4.0
	github.com/dop251/goja v0.0.0-00010101000000-000000000000
	github.com/dop251/goja_nodejs v0.0.0-20171011081505-adff31b136e6
	github.com/dustin/go-humanize v0.0.0-20171111073723-bb3d318650d4
	github.com/eclipse/paho.mqtt.golang v1.2.1-0.20200121105743-0d940dd29fd2
	github.com/elastic/ecs v1.4.0
	github.com/elastic/go-libaudit v0.4.0
	github.com/elastic/go-licenser v0.2.1
	github.com/elastic/go-lookslike v0.3.0
	github.com/elastic/go-lumber v0.1.0
	github.com/elastic/go-perf v0.0.0-20191212140718-9c656876f595
	github.com/elastic/go-seccomp-bpf v1.1.0
	github.com/elastic/go-structform v0.0.6
	github.com/elastic/go-sysinfo v1.3.0
	github.com/elastic/go-txfile v0.0.7
	github.com/elastic/go-ucfg v0.8.3
	github.com/elastic/go-windows v1.0.1 // indirect
	github.com/elastic/gosigar v0.10.5
	github.com/fatih/color v1.5.0
	github.com/fsnotify/fsevents v0.0.0-00010101000000-000000000000
	github.com/fsnotify/fsnotify v1.4.7
	github.com/garyburd/redigo v1.0.1-0.20160525165706-b8dc90050f24
	github.com/go-ole/go-ole v1.2.5-0.20190920104607-14974a1cf647 // indirect
	github.com/go-sourcemap/sourcemap v2.1.2+incompatible // indirect
	github.com/go-sql-driver/mysql v1.4.1
	github.com/gocarina/gocsv v0.0.0-20170324095351-ffef3ffc77be
	github.com/godbus/dbus v0.0.0-20190422162347-ade71ed3457e
	github.com/godror/godror v0.10.4
	github.com/gofrs/flock v0.7.2-0.20190320160742-5135e617513b
	github.com/gofrs/uuid v3.2.0+incompatible
	github.com/gogo/protobuf v1.3.1
	github.com/golang/protobuf v1.3.2
	github.com/golang/snappy v0.0.1
	github.com/google/flatbuffers v1.7.2-0.20170925184458-7a6b2bf521e9
	github.com/google/go-cmp v0.4.0
	github.com/google/gopacket v1.1.18-0.20191009163724-0ad7f2610e34
	github.com/google/uuid v1.1.2-0.20190416172445-c2e93f3ae59f // indirect
	github.com/googleapis/gnostic v0.3.1-0.20190624222214-25d8b0b66985 // indirect
	github.com/gorhill/cronexpr v0.0.0-20161205141322-d520615e531a
	github.com/gorilla/mux v1.7.2 // indirect
	github.com/gorilla/websocket v1.4.1 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.13.0 // indirect
	github.com/hashicorp/go-multierror v0.0.0-20161216184304-ed905158d874
	github.com/hashicorp/golang-lru v0.5.2-0.20190520140433-59383c442f7d // indirect
	github.com/insomniacslk/dhcp v0.0.0-20180716145214-633285ba52b2
	github.com/jcmturner/gofork v1.0.0 // indirect
	github.com/jmoiron/sqlx v1.2.1-0.20190826204134-d7d95172beb5
	github.com/joeshaw/multierror v0.0.0-20140124173710-69b34d4ec901
	github.com/josephspurrier/goversioninfo v0.0.0-20190209210621-63e6d1acd3dd
	github.com/jpillora/backoff v1.0.0 // indirect
	github.com/jstemmer/go-junit-report v0.9.1
	github.com/klauspost/compress v1.9.3-0.20191122130757-c099ac9f21dd // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/lib/pq v1.1.2-0.20190507191818-2ff3cb3adc01
	github.com/magefile/mage v1.9.0
	github.com/mailru/easyjson v0.7.1 // indirect
	github.com/mattn/go-colorable v0.0.8
	github.com/mattn/go-ieproxy v0.0.0-20191113090002-7c0f6868bffe // indirect
	github.com/mattn/go-isatty v0.0.2 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/miekg/dns v1.1.15
	github.com/mitchellh/gox v1.0.1
	github.com/mitchellh/hashstructure v0.0.0-20170116052023-ab25296c0f51
	github.com/mitchellh/mapstructure v1.1.2
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/oklog/ulid v1.3.1
	github.com/opencontainers/go-digest v1.0.0-rc1.0.20190228220655-ac19fd6e7483 // indirect
	github.com/opencontainers/image-spec v1.0.2-0.20190823105129-775207bd45b6 // indirect
	github.com/pierrre/gotestcover v0.0.0-20160113212533-7b94f124d338
	github.com/pkg/errors v0.9.1
	github.com/pmezard/go-difflib v1.0.0
	github.com/prometheus/client_golang v1.1.1-0.20190913103102-20428fa0bffc // indirect
	github.com/prometheus/client_model v0.0.0-20190812154241-14fe0d1b01d4
	github.com/prometheus/common v0.7.0
	github.com/prometheus/procfs v0.0.9-0.20191208103036-42f6e295b56f
	github.com/prometheus/prometheus v2.5.0+incompatible
	github.com/rcrowley/go-metrics v0.0.0-20181016184325-3113b8401b8a
	github.com/reviewdog/reviewdog v0.9.17
	github.com/samuel/go-parser v0.0.0-20130731160455-ca8abbf65d0e // indirect
	github.com/samuel/go-thrift v0.0.0-20140522043831-2187045faa54
	github.com/sanathkr/yaml v1.0.1-0.20170819201035-0056894fa522 // indirect
	github.com/shirou/gopsutil v2.19.11+incompatible
	github.com/spf13/cobra v0.0.3
	github.com/spf13/pflag v1.0.3
	github.com/stretchr/testify v1.4.0
	github.com/tsg/go-daemon v0.0.0-20200207173439-e704b93fd89b
	github.com/tsg/gopacket v0.0.0-20190320122513-dd3d0e41124a
	github.com/urso/ecslog v0.0.1
	github.com/vmware/govmomi v0.0.0-20170802214208-2cad15190b41
	github.com/xdg/scram v0.0.0-20180814205039-7eeb5667e42c
	github.com/yuin/gopher-lua v0.0.0-20170403160031-b402f3114ec7 // indirect
	go.uber.org/atomic v1.3.1
	go.uber.org/multierr v1.1.1-0.20170829224307-fb7d312c2c04
	go.uber.org/zap v1.7.1
	golang.org/x/crypto v0.0.0-20200204104054-c9f3fb736b72
	golang.org/x/lint v0.0.0-20191125180803-fdd1cda4f05f
	golang.org/x/net v0.0.0-20200202094626-16171245cfb2
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	golang.org/x/sys v0.0.0-20200102141924-c96a22e43c9c
	golang.org/x/text v0.3.2
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0
	golang.org/x/tools v0.0.0-20191227053925-7b8e75db28f4
	google.golang.org/api v0.15.0
	google.golang.org/genproto v0.0.0-20191230161307-f3c370f40bfb
	google.golang.org/grpc v1.27.1
	gopkg.in/inf.v0 v0.9.0
	gopkg.in/jcmturner/gokrb5.v7 v7.3.0 // indirect
	gopkg.in/mgo.v2 v2.0.0-20160818020120-3f83fa500528
	gopkg.in/yaml.v2 v2.2.8
	howett.net/plist v0.0.0-20181124034731-591f970eefbb
	k8s.io/api v0.0.0-20190722141453-b90922c02518
	k8s.io/apimachinery v0.0.0-20190719140911-bfcf53abc9f8
	k8s.io/client-go v0.0.0-20190620085101-78d2af792bab
	k8s.io/klog v0.3.4-0.20190719014911-6a023d6d0e09 // indirect
	k8s.io/utils v0.0.0-20190712204705-3dccf664f023 // indirect
	sigs.k8s.io/yaml v1.1.1-0.20190704183835-4cd0c284b15f // indirect
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v12.2.0+incompatible
	github.com/Shopify/sarama => github.com/elastic/sarama v0.0.0-20191122160421-355d120d0970
	github.com/docker/docker => github.com/docker/engine v0.0.0-20191113042239-ea84732a7725
	github.com/docker/go-plugins-helpers => github.com/elastic/go-plugins-helpers v0.0.0-20200207104224-bdf17607b79f
	github.com/dop251/goja => github.com/andrewkroh/goja v0.0.0-20190128172624-dd2ac4456e20
	github.com/fsnotify/fsevents => github.com/elastic/fsevents v0.0.0-20181029231046-e1d381a4d270
	github.com/fsnotify/fsnotify => github.com/adriansr/fsnotify v0.0.0-20180417234312-c9bbe1f46f1d
	github.com/insomniacslk/dhcp => github.com/elastic/dhcp v0.0.0-20200227161230-57ec251c7eb3 // indirect
	github.com/tonistiigi/fifo => github.com/containerd/fifo v0.0.0-20190816180239-bda0ff6ed73c
)
