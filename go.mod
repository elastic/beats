module github.com/elastic/beats/v7

go 1.18

require (
	cloud.google.com/go/bigquery v1.8.0
	cloud.google.com/go/monitoring v1.1.0
	cloud.google.com/go/pubsub v1.25.1
	code.cloudfoundry.org/go-diodes v0.0.0-20190809170250-f77fb823c7ee // indirect
	code.cloudfoundry.org/go-loggregator v7.4.0+incompatible
	code.cloudfoundry.org/rfc5424 v0.0.0-20180905210152-236a6d29298a // indirect
	github.com/Azure/azure-event-hubs-go/v3 v3.3.15
	github.com/Azure/azure-sdk-for-go v59.0.0+incompatible
	github.com/Azure/azure-storage-blob-go v0.8.0
	github.com/Azure/go-ansiterm v0.0.0-20210617225240-d185dfc1b5a1 // indirect
	github.com/Azure/go-autorest/autorest v0.11.19
	github.com/Azure/go-autorest/autorest/azure/auth v0.4.2
	github.com/Azure/go-autorest/autorest/date v0.3.0
	github.com/Masterminds/semver v1.5.0
	github.com/Microsoft/go-winio v0.5.2
	github.com/PaesslerAG/gval v1.0.0
	github.com/PaesslerAG/jsonpath v0.1.1
	github.com/Shopify/sarama v1.27.0
	github.com/StackExchange/wmi v1.2.1
	github.com/aerospike/aerospike-client-go v1.27.1-0.20170612174108-0f3b54da6bdc
	github.com/akavel/rsrc v0.8.0 // indirect
	github.com/andrewkroh/sys v0.0.0-20151128191922-287798fe3e43
	github.com/apoydence/eachers v0.0.0-20181020210610-23942921fe77 // indirect
	github.com/armon/go-socks5 v0.0.0-20160902184237-e75332964ef5
	github.com/aws/aws-lambda-go v1.13.3
	github.com/aws/aws-sdk-go-v2 v1.16.6
	github.com/aws/aws-sdk-go-v2/config v1.15.12
	github.com/aws/aws-sdk-go-v2/credentials v1.12.7
	github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.18.2
	github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs v1.15.5
	github.com/aws/aws-sdk-go-v2/service/costexplorer v1.18.4
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.36.1
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2 v1.18.4
	github.com/aws/aws-sdk-go-v2/service/iam v1.18.4
	github.com/aws/aws-sdk-go-v2/service/organizations v1.15.2
	github.com/aws/aws-sdk-go-v2/service/rds v1.20.1
	github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi v1.13.5
	github.com/aws/aws-sdk-go-v2/service/s3 v1.26.12
	github.com/aws/aws-sdk-go-v2/service/sqs v1.18.4
	github.com/aws/aws-sdk-go-v2/service/sts v1.16.8
	github.com/awslabs/goformation/v4 v4.1.0
	github.com/blakesmith/ar v0.0.0-20150311145944-8bd4349a67f2
	github.com/bsm/sarama-cluster v2.1.14-0.20180625083203-7e67d87a6b3f+incompatible
	github.com/cavaliercoder/badio v0.0.0-20160213150051-ce5280129e9e // indirect
	github.com/cavaliercoder/go-rpm v0.0.0-20190131055624-7a9c54e3d83e
	github.com/cespare/xxhash/v2 v2.1.2
	github.com/cloudfoundry-community/go-cfclient v0.0.0-20190808214049-35bcce23fc5f
	github.com/cloudfoundry/noaa v2.1.0+incompatible
	github.com/cloudfoundry/sonde-go v0.0.0-20171206171820-b33733203bb4
	github.com/containerd/fifo v1.0.0
	github.com/coreos/go-systemd/v22 v22.3.2
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f
	github.com/davecgh/go-xdr v0.0.0-20161123171359-e6a2ba005892 // indirect
	github.com/denisenkom/go-mssqldb v0.9.0
	github.com/devigned/tab v0.1.2-0.20190607222403-0c15cf42f9a2 // indirect
	github.com/dgraph-io/badger/v3 v3.2103.1
	github.com/digitalocean/go-libvirt v0.0.0-20180301200012-6075ea3c39a1
	github.com/docker/docker v20.10.12+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-plugins-helpers v0.0.0-20181025120712-1e6269c305b8
	github.com/docker/go-units v0.4.0
	github.com/dolmen-go/contextio v0.0.0-20200217195037-68fc5150bcd5
	github.com/dop251/goja v0.0.0-20200831102558-9af81ddcf0e1
	github.com/dop251/goja_nodejs v0.0.0-20171011081505-adff31b136e6
	github.com/dustin/go-humanize v1.0.0
	github.com/eapache/go-resiliency v1.2.0
	github.com/eclipse/paho.mqtt.golang v1.3.5
	github.com/elastic/elastic-agent-client/v7 v7.0.3
	github.com/elastic/go-concert v0.2.0
	github.com/elastic/go-libaudit/v2 v2.3.2
	github.com/elastic/go-licenser v0.4.0
	github.com/elastic/go-lookslike v0.3.0
	github.com/elastic/go-lumber v0.1.2-0.20220819171948-335fde24ea0f
	github.com/elastic/go-perf v0.0.0-20191212140718-9c656876f595
	github.com/elastic/go-seccomp-bpf v1.3.0
	github.com/elastic/go-structform v0.0.10
	github.com/elastic/go-sysinfo v1.9.0
	github.com/elastic/go-ucfg v0.8.6
	github.com/elastic/gosigar v0.14.2
	github.com/fatih/color v1.13.0
	github.com/fearful-symmetry/gorapl v0.0.4
	github.com/fsnotify/fsevents v0.1.1
	github.com/fsnotify/fsnotify v1.5.1
	github.com/go-sourcemap/sourcemap v2.1.2+incompatible // indirect
	github.com/go-sql-driver/mysql v1.6.0
	github.com/go-test/deep v1.0.7
	github.com/gocarina/gocsv v0.0.0-20170324095351-ffef3ffc77be
	github.com/godbus/dbus v0.0.0-20190422162347-ade71ed3457e
	github.com/godror/godror v0.33.2
	github.com/gofrs/flock v0.8.1
	github.com/gofrs/uuid v4.2.0+incompatible
	github.com/gogo/protobuf v1.3.2
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.5.2
	github.com/golang/snappy v0.0.4
	github.com/gomodule/redigo v1.8.3
	github.com/google/flatbuffers v1.12.1
	github.com/google/go-cmp v0.5.8
	github.com/google/gopacket v1.1.19
	github.com/google/uuid v1.3.0
	github.com/gorhill/cronexpr v0.0.0-20180427100037-88b0669f7d75
	github.com/h2non/filetype v1.1.1
	github.com/hashicorp/go-multierror v1.1.1
	github.com/hashicorp/go-retryablehttp v0.6.6
	github.com/hashicorp/golang-lru v0.5.4
	github.com/hashicorp/nomad/api v0.0.0-20201203164818-6318a8ac7bf8
	github.com/hectane/go-acl v0.0.0-20190604041725-da78bae5fc95
	github.com/insomniacslk/dhcp v0.0.0-20180716145214-633285ba52b2
	github.com/jarcoal/httpmock v1.0.4
	github.com/jmoiron/sqlx v1.3.1
	github.com/joeshaw/multierror v0.0.0-20140124173710-69b34d4ec901
	github.com/jonboulle/clockwork v0.2.2
	github.com/josephspurrier/goversioninfo v0.0.0-20190209210621-63e6d1acd3dd
	github.com/jpillora/backoff v1.0.0 // indirect
	github.com/lib/pq v1.10.3
	github.com/magefile/mage v1.14.0
	github.com/mattn/go-colorable v0.1.12
	github.com/mattn/go-ieproxy v0.0.0-20191113090002-7c0f6868bffe // indirect
	github.com/miekg/dns v1.1.42
	github.com/mitchellh/gox v1.0.1
	github.com/mitchellh/hashstructure v1.1.0
	github.com/mitchellh/mapstructure v1.5.0
	github.com/olekukonko/tablewriter v0.0.5
	github.com/osquery/osquery-go v0.0.0-20210622151333-99b4efa62ec5
	github.com/otiai10/copy v1.2.0
	github.com/pierrre/gotestcover v0.0.0-20160517101806-924dca7d15f0
	github.com/pkg/errors v0.9.1
	github.com/pmezard/go-difflib v1.0.0
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.30.0
	github.com/prometheus/procfs v0.8.0
	github.com/prometheus/prometheus v1.8.2-0.20210701133801-b0944590a1c9
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475
	github.com/samuel/go-parser v0.0.0-20130731160455-ca8abbf65d0e // indirect
	github.com/samuel/go-thrift v0.0.0-20140522043831-2187045faa54
	github.com/sanathkr/yaml v1.0.1-0.20170819201035-0056894fa522 // indirect
	github.com/shopspring/decimal v1.2.0
	github.com/spf13/cobra v1.3.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.8.1
	github.com/tsg/go-daemon v0.0.0-20200207173439-e704b93fd89b
	github.com/ugorji/go/codec v1.1.8
	github.com/urso/sderr v0.0.0-20210525210834-52b04e8f5c71
	github.com/vmware/govmomi v0.0.0-20170802214208-2cad15190b41
	github.com/xdg/scram v1.0.3
	go.elastic.co/ecszap v1.0.1
	go.elastic.co/go-licence-detector v0.5.0
	go.etcd.io/bbolt v1.3.6
	go.uber.org/atomic v1.10.0
	go.uber.org/multierr v1.9.0
	go.uber.org/zap v1.24.0
	golang.org/x/crypto v0.0.0-20220525230936-793ad666bf5e
	golang.org/x/lint v0.0.0-20210508222113-6edffad5e616
	golang.org/x/mod v0.6.0-dev.0.20220419223038-86c51ed26bb4
	golang.org/x/net v0.2.0
	golang.org/x/oauth2 v0.0.0-20220822191816-0ebed06d0094
	golang.org/x/sync v0.0.0-20220819030929-7fc1605a5dde
	golang.org/x/sys v0.2.0
	golang.org/x/text v0.4.0
	golang.org/x/time v0.0.0-20220609170525-579cf78fd858
	golang.org/x/tools v0.1.12
	google.golang.org/api v0.94.0
	google.golang.org/genproto v0.0.0-20220829175752-36a9c930ecbf
	google.golang.org/grpc v1.49.0
	google.golang.org/protobuf v1.28.1
	gopkg.in/inf.v0 v0.9.1
	gopkg.in/jcmturner/aescts.v1 v1.0.1 // indirect
	gopkg.in/jcmturner/dnsutils.v1 v1.0.1 // indirect
	gopkg.in/jcmturner/goidentity.v3 v3.0.0 // indirect
	gopkg.in/jcmturner/gokrb5.v7 v7.5.0
	gopkg.in/yaml.v2 v2.4.0
	gotest.tools v2.2.0+incompatible
	gotest.tools/gotestsum v1.7.0
	howett.net/plist v1.0.0
	k8s.io/api v0.23.4
	k8s.io/apimachinery v0.23.4
	k8s.io/client-go v0.23.4
	kernel.org/pub/linux/libs/security/libcap/cap v1.2.57
)

require (
	github.com/Azure/azure-sdk-for-go/sdk/storage/azblob v0.4.1
	github.com/Azure/go-autorest/autorest/adal v0.9.14
	github.com/aws/aws-sdk-go-v2/feature/s3/manager v1.11.17
	github.com/aws/aws-sdk-go-v2/service/cloudformation v1.20.4
	github.com/aws/aws-sdk-go-v2/service/kinesis v1.15.8
	github.com/aws/smithy-go v1.12.0
	github.com/awslabs/kinesis-aggregation/go/v2 v2.0.0-20220623125934-28468a6701b5
	github.com/elastic/bayeux v1.0.5
	github.com/elastic/elastic-agent-autodiscover v0.5.0
	github.com/elastic/elastic-agent-libs v0.3.1
	github.com/elastic/elastic-agent-shipper-client v0.5.1-0.20230228231646-f04347b666f3
	github.com/elastic/elastic-agent-system-metrics v0.4.6
	github.com/elastic/go-elasticsearch/v8 v8.2.0
	github.com/elastic/mito v0.0.0-20221207004749-2f0f2875e464
	github.com/elastic/toutoumomoma v0.0.0-20221026030040-594ef30cb640
	github.com/google/cel-go v0.12.5
	github.com/googleapis/gax-go/v2 v2.5.1
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/pierrec/lz4/v4 v4.1.15
	github.com/shirou/gopsutil/v3 v3.21.12
	go.elastic.co/apm/module/apmelasticsearch/v2 v2.0.0
	go.elastic.co/apm/module/apmhttp/v2 v2.0.0
	go.elastic.co/apm/v2 v2.0.0
	go.mongodb.org/mongo-driver v1.5.1
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
)

require (
	cloud.google.com/go v0.104.0 // indirect
	cloud.google.com/go/compute v1.9.0 // indirect
	cloud.google.com/go/iam v0.3.0 // indirect
	code.cloudfoundry.org/gofileutils v0.0.0-20170111115228-4d0c80011a0f // indirect
	github.com/Azure/azure-amqp-common-go/v3 v3.2.1 // indirect
	github.com/Azure/azure-pipeline-go v0.2.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.1.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.0.0 // indirect
	github.com/Azure/go-amqp v0.16.0 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest/azure/cli v0.3.1 // indirect
	github.com/Azure/go-autorest/autorest/to v0.4.0 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.3.1 // indirect
	github.com/Azure/go-autorest/logger v0.2.1 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v0.5.1 // indirect
	github.com/antlr/antlr4/runtime/Go/antlr v0.0.0-20220418222510-f25a4f6275ed // indirect
	github.com/apache/thrift v0.13.1-0.20200603211036-eac4d0c79a5f // indirect
	github.com/armon/go-radix v1.0.0 // indirect
	github.com/aws/aws-sdk-go v1.38.60 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.4.3 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.12.7 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.13 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.4.7 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.14 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.0.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.9.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.1.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.9.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.13.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.11.10 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash v1.1.0 // indirect
	github.com/containerd/containerd v1.5.13 // indirect
	github.com/cyphar/filepath-securejoin v0.2.3 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgraph-io/ristretto v0.1.0 // indirect
	github.com/dimchansky/utfbom v1.1.0 // indirect
	github.com/dnephin/pflag v1.0.7 // indirect
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/eapache/go-xerial-snappy v0.0.0-20180814174437-776d5712da21 // indirect
	github.com/eapache/queue v1.1.0 // indirect
	github.com/elastic/elastic-transport-go/v8 v8.1.0 // indirect
	github.com/elastic/go-windows v1.0.1 // indirect
	github.com/evanphx/json-patch v4.12.0+incompatible // indirect
	github.com/fearful-symmetry/gomsr v0.0.1 // indirect
	github.com/felixge/httpsnoop v1.0.1 // indirect
	github.com/form3tech-oss/jwt-go v3.2.3+incompatible // indirect
	github.com/go-logfmt/logfmt v0.5.1 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/gobuffalo/here v0.6.0 // indirect
	github.com/godror/knownpb v0.1.0 // indirect
	github.com/golang-sql/civil v0.0.0-20190719163853-cb61b32ac6fe // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/google/licenseclassifier v0.0.0-20200402202327-879cb1424de0 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.1.0 // indirect
	github.com/googleapis/gnostic v0.5.5 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/hashicorp/cronexpr v1.1.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-uuid v1.0.2 // indirect
	github.com/hashicorp/go-version v1.2.0 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/jcchavezs/porto v0.1.0 // indirect
	github.com/jcmturner/aescts/v2 v2.0.0 // indirect
	github.com/jcmturner/dnsutils/v2 v2.0.0 // indirect
	github.com/jcmturner/gofork v1.0.0 // indirect
	github.com/jcmturner/gokrb5/v8 v8.4.2 // indirect
	github.com/jcmturner/rpc/v2 v2.0.3 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/karrick/godirwalk v1.15.8 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/klauspost/compress v1.15.9 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/mailru/easyjson v0.7.6 // indirect
	github.com/markbates/pkger v0.17.0 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/iochan v1.0.0 // indirect
	github.com/moby/spdystream v0.2.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.0.2 // indirect
	github.com/pierrec/lz4 v2.6.0+incompatible // indirect
	github.com/pkg/browser v0.0.0-20210911075715-681adbf594b8 // indirect
	github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
	github.com/prometheus/client_golang v1.11.0 // indirect
	github.com/sanathkr/go-yaml v0.0.0-20170819195128-ed9d249f429b // indirect
	github.com/santhosh-tekuri/jsonschema v1.2.4 // indirect
	github.com/sergi/go-diff v1.1.0 // indirect
	github.com/shirou/gopsutil v3.21.11+incompatible // indirect
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/stoewer/go-strcase v1.2.0 // indirect
	github.com/stretchr/objx v0.5.0 // indirect
	github.com/tklauser/go-sysconf v0.3.9 // indirect
	github.com/tklauser/numcpus v0.3.0 // indirect
	github.com/urso/diag v0.0.0-20200210123136-21b3cc8eb797 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.0.2 // indirect
	github.com/xdg-go/stringprep v1.0.2 // indirect
	github.com/xdg/stringprep v1.0.3 // indirect
	github.com/youmark/pkcs8 v0.0.0-20181117223130-1be2e3e5546d // indirect
	github.com/yusufpapurcu/wmi v1.2.2 // indirect
	go.elastic.co/fastjson v1.1.0 // indirect
	go.opencensus.io v0.23.0 // indirect
	golang.org/x/exp v0.0.0-20220921023135-46d9e7742f1e // indirect
	golang.org/x/term v0.2.0 // indirect
	golang.org/x/xerrors v0.0.0-20220609144429-65e65417b02f // indirect
	google.golang.org/appengine v1.6.7 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/klog/v2 v2.30.0 // indirect
	k8s.io/kube-openapi v0.0.0-20211115234752-e816edb12b65 // indirect
	k8s.io/utils v0.0.0-20211116205334-6203023598ed // indirect
	kernel.org/pub/linux/libs/security/libcap/psx v1.2.57 // indirect
	mvdan.cc/garble v0.7.1 // indirect
	sigs.k8s.io/json v0.0.0-20211020170558-c049b76a60c6 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.1 // indirect
	sigs.k8s.io/yaml v1.2.0 // indirect
)

require (
	cloud.google.com/go/storage v1.26.0
	github.com/dlclark/regexp2 v1.4.0 // indirect
	github.com/docker/distribution v2.8.1+incompatible // indirect
	github.com/godbus/dbus/v5 v5.0.6 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/moby/term v0.0.0-20210610120745-9d4ed1856297 // indirect
	github.com/yuin/gopher-lua v0.0.0-20170403160031-b402f3114ec7 // indirect
	gopkg.in/jcmturner/rpc.v1 v1.1.0 // indirect
)

replace (
	github.com/Azure/azure-sdk-for-go => github.com/elastic/azure-sdk-for-go v59.0.0-elastic-1+incompatible
	github.com/Microsoft/go-winio => github.com/bi-zone/go-winio v0.4.15
	github.com/Shopify/sarama => github.com/elastic/sarama v1.19.1-0.20220310193331-ebc2b0d8eef3
	github.com/apoydence/eachers => github.com/poy/eachers v0.0.0-20181020210610-23942921fe77 //indirect, see https://github.com/elastic/beats/pull/29780 for details.
	github.com/cucumber/godog => github.com/cucumber/godog v0.8.1
	github.com/dgraph-io/ristretto => github.com/elastic/ristretto v0.1.1-0.20220602190459-83b0895ca5b3 // Removes glog dependency. See https://github.com/elastic/beats/issues/31810.
	github.com/docker/go-plugins-helpers => github.com/elastic/go-plugins-helpers v0.0.0-20200207104224-bdf17607b79f
	github.com/dop251/goja => github.com/andrewkroh/goja v0.0.0-20190128172624-dd2ac4456e20
	github.com/dop251/goja_nodejs => github.com/dop251/goja_nodejs v0.0.0-20171011081505-adff31b136e6
	github.com/fsnotify/fsevents => github.com/elastic/fsevents v0.0.0-20181029231046-e1d381a4d270
	github.com/fsnotify/fsnotify => github.com/adriansr/fsnotify v1.4.8-0.20211018144411-a81f2b630e7c
	github.com/godror/godror => github.com/godror/godror v0.33.2 // updating to v0.24.2 caused a breaking change
	github.com/golang/glog => github.com/elastic/glog v1.0.1-0.20210831205241-7d8b5c89dfc4
	github.com/google/gopacket => github.com/elastic/gopacket v1.1.20-0.20211202005954-d412fca7f83a
	github.com/insomniacslk/dhcp => github.com/elastic/dhcp v0.0.0-20200227161230-57ec251c7eb3 // indirect
	github.com/tonistiigi/fifo => github.com/containerd/fifo v0.0.0-20190816180239-bda0ff6ed73c
)

// Exclude this version because the version has an invalid checksum.
exclude github.com/docker/distribution v2.8.0+incompatible
