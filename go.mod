module github.com/elastic/beats/v7

go 1.24.10

require (
	cloud.google.com/go/bigquery v1.69.0
	cloud.google.com/go/monitoring v1.24.2
	cloud.google.com/go/pubsub v1.49.0
	code.cloudfoundry.org/go-diodes v0.0.0-20190809170250-f77fb823c7ee // indirect
	code.cloudfoundry.org/go-loggregator v7.4.0+incompatible
	code.cloudfoundry.org/rfc5424 v0.0.0-20180905210152-236a6d29298a // indirect
	github.com/Azure/azure-event-hubs-go/v3 v3.6.1
	github.com/Azure/azure-sdk-for-go v68.0.0+incompatible
	github.com/Azure/go-ansiterm v0.0.0-20230124172434-306776ec8161 // indirect
	github.com/Azure/go-autorest/autorest v0.11.30
	github.com/Azure/go-autorest/autorest/date v0.3.1
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Microsoft/go-winio v0.6.2
	github.com/PaesslerAG/gval v1.2.2
	github.com/PaesslerAG/jsonpath v0.1.1
	github.com/StackExchange/wmi v1.2.1
	github.com/akavel/rsrc v0.10.2 // indirect
	github.com/apoydence/eachers v0.0.0-20181020210610-23942921fe77 // indirect
	github.com/armon/go-socks5 v0.0.0-20160902184237-e75332964ef5
	github.com/aws/aws-sdk-go-v2 v1.39.2
	github.com/aws/aws-sdk-go-v2/config v1.31.12
	github.com/aws/aws-sdk-go-v2/credentials v1.18.16
	github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.45.2
	github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs v1.50.2
	github.com/aws/aws-sdk-go-v2/service/costexplorer v1.51.1
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.254.1
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2 v1.45.4
	github.com/aws/aws-sdk-go-v2/service/iam v1.42.1
	github.com/aws/aws-sdk-go-v2/service/organizations v1.38.4
	github.com/aws/aws-sdk-go-v2/service/rds v1.97.2
	github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi v1.26.5
	github.com/aws/aws-sdk-go-v2/service/s3 v1.88.3
	github.com/aws/aws-sdk-go-v2/service/sqs v1.38.7
	github.com/aws/aws-sdk-go-v2/service/sts v1.38.6
	github.com/blakesmith/ar v0.0.0-20150311145944-8bd4349a67f2
	github.com/cavaliergopher/rpm v1.2.0
	github.com/cespare/xxhash/v2 v2.3.0
	github.com/cloudfoundry-community/go-cfclient v0.0.0-20190808214049-35bcce23fc5f
	github.com/cloudfoundry/noaa v2.1.0+incompatible
	github.com/cloudfoundry/sonde-go v0.0.0-20171206171820-b33733203bb4
	github.com/containerd/fifo v1.1.0
	github.com/coreos/go-systemd/v22 v22.6.0
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f
	github.com/devigned/tab v0.1.2-0.20190607222403-0c15cf42f9a2
	github.com/digitalocean/go-libvirt v0.0.0-20240709142323-d8406205c752
	github.com/docker/docker v28.4.0+incompatible
	github.com/docker/go-connections v0.6.0
	github.com/docker/go-plugins-helpers v0.0.0-20181025120712-1e6269c305b8
	github.com/docker/go-units v0.5.0
	github.com/dop251/goja v0.0.0-20200831102558-9af81ddcf0e1
	github.com/dop251/goja_nodejs v0.0.0-20171011081505-adff31b136e6
	github.com/dustin/go-humanize v1.0.1
	github.com/eapache/go-resiliency v1.7.0
	github.com/eclipse/paho.mqtt.golang v1.3.5
	github.com/elastic/elastic-agent-client/v7 v7.15.0
	github.com/elastic/go-concert v0.3.0
	github.com/elastic/go-libaudit/v2 v2.6.2
	github.com/elastic/go-licenser v0.4.2
	github.com/elastic/go-lookslike v1.0.1
	github.com/elastic/go-lumber v0.1.2-0.20220819171948-335fde24ea0f
	github.com/elastic/go-perf v0.0.0-20241029065020-30bec95324b8
	github.com/elastic/go-seccomp-bpf v1.5.0
	github.com/elastic/go-structform v0.0.12
	github.com/elastic/go-sysinfo v1.15.3
	github.com/elastic/go-ucfg v0.8.8
	github.com/elastic/gosigar v0.14.3
	github.com/elastic/pkcs8 v1.0.0
	github.com/fatih/color v1.16.0 // indirect
	github.com/fearful-symmetry/gorapl v0.0.4
	github.com/fsnotify/fsevents v0.1.1
	github.com/fsnotify/fsnotify v1.9.0
	github.com/go-sourcemap/sourcemap v2.1.2+incompatible // indirect
	github.com/go-sql-driver/mysql v1.9.3
	github.com/go-viper/mapstructure/v2 v2.4.0
	github.com/gocarina/gocsv v0.0.0-20170324095351-ffef3ffc77be
	github.com/godbus/dbus/v5 v5.1.0
	github.com/godror/godror v0.49.3
	github.com/gofrs/flock v0.8.1
	github.com/gogo/protobuf v1.3.2
	github.com/gohugoio/hashstructure v0.5.0
	github.com/golang/snappy v1.0.0
	github.com/gomodule/redigo v1.9.2
	github.com/google/flatbuffers v25.2.10+incompatible
	github.com/google/go-cmp v0.7.0
	github.com/google/gopacket v1.1.19
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorhill/cronexpr v0.0.0-20180427100037-88b0669f7d75
	github.com/h2non/filetype v1.1.1
	github.com/hashicorp/go-retryablehttp v0.7.7
	github.com/hashicorp/nomad/api v0.0.0-20250930071859-eaa0fe0e27af
	github.com/hectane/go-acl v0.0.0-20190604041725-da78bae5fc95
	github.com/insomniacslk/dhcp v0.0.0-20220119180841-3c283ff8b7dd
	github.com/jonboulle/clockwork v0.2.2
	github.com/josephspurrier/goversioninfo v1.5.0
	github.com/jpillora/backoff v1.0.0 // indirect
	github.com/lib/pq v1.10.9
	github.com/magefile/mage v1.15.0
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/miekg/dns v1.1.68
	github.com/osquery/osquery-go v0.0.0-20231108163517-e3cde127e724
	github.com/pkg/errors v0.9.1
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.67.1
	github.com/prometheus/procfs v0.17.0
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475
	github.com/samuel/go-parser v0.0.0-20130731160455-ca8abbf65d0e // indirect
	github.com/samuel/go-thrift v0.0.0-20140522043831-2187045faa54
	github.com/shopspring/decimal v1.3.1 // indirect
	github.com/spf13/cobra v1.10.1
	github.com/spf13/pflag v1.0.9
	github.com/stretchr/testify v1.11.1
	github.com/ugorji/go/codec v1.1.8
	github.com/vmware/govmomi v0.52.0
	go.elastic.co/ecszap v1.0.2
	go.elastic.co/go-licence-detector v0.7.0
	go.etcd.io/bbolt v1.4.0
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0
	golang.org/x/crypto v0.44.0
	golang.org/x/mod v0.29.0
	golang.org/x/net v0.47.0
	golang.org/x/oauth2 v0.31.0
	golang.org/x/sync v0.18.0
	golang.org/x/sys v0.38.0
	golang.org/x/text v0.31.0
	golang.org/x/time v0.13.0
	golang.org/x/tools v0.38.0
	google.golang.org/api v0.250.0
	google.golang.org/genproto v0.0.0-20250603155806-513f23925822 // indirect
	google.golang.org/grpc v1.76.0
	google.golang.org/protobuf v1.36.10
	gopkg.in/inf.v0 v0.9.1
	gopkg.in/yaml.v2 v2.4.0
	gotest.tools/gotestsum v1.7.0
	howett.net/plist v1.0.1
	k8s.io/api v0.34.1
	k8s.io/apimachinery v0.34.1
	k8s.io/client-go v0.34.1
	kernel.org/pub/linux/libs/security/libcap/cap v1.2.57
)

require (
	cloud.google.com/go v0.121.0
	cloud.google.com/go/compute v1.38.0
	cloud.google.com/go/redis v1.18.2
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.20.0
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.13.1
	github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs v1.3.1
	github.com/Azure/azure-sdk-for-go/sdk/monitor/query/azmetrics v1.1.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/consumption/armconsumption v1.1.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v4 v4.8.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/costmanagement/armcostmanagement v1.1.1
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor v0.8.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources v1.2.0
	github.com/Azure/azure-sdk-for-go/sdk/storage/azblob v1.6.0
	github.com/Azure/azure-storage-blob-go v0.15.0
	github.com/aerospike/aerospike-client-go/v7 v7.7.1
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.9
	github.com/aws/aws-sdk-go-v2/feature/s3/manager v1.17.79
	github.com/aws/aws-sdk-go-v2/service/apigateway v1.31.3
	github.com/aws/aws-sdk-go-v2/service/apigatewayv2 v1.28.3
	github.com/aws/aws-sdk-go-v2/service/health v1.30.3
	github.com/aws/smithy-go v1.23.0
	github.com/beevik/ntp v1.4.3
	github.com/brianvoe/gofakeit v3.18.0+incompatible
	github.com/dgraph-io/badger/v4 v4.6.0
	github.com/elastic/bayeux v1.0.5
	github.com/elastic/ebpfevents v0.8.0
	github.com/elastic/elastic-agent-autodiscover v0.10.0
	github.com/elastic/elastic-agent-libs v0.26.0
	github.com/elastic/elastic-agent-system-metrics v0.13.4
	github.com/elastic/go-elasticsearch/v8 v8.19.0
	github.com/elastic/go-freelru v0.16.0
	github.com/elastic/go-quark v0.3.0
	github.com/elastic/go-sfdc v0.0.0-20241010131323-8e176480d727
	github.com/elastic/mito v1.23.0
	github.com/elastic/mock-es v0.0.0-20250530054253-8c3b6053f9b6
	github.com/elastic/sarama v1.19.1-0.20250603175145-7672917f26b6
	github.com/elastic/tk-btf v0.2.0
	github.com/elastic/toutoumomoma v0.0.0-20240626215117-76e39db18dfb
	github.com/go-ldap/ldap/v3 v3.4.6
	github.com/go-ole/go-ole v1.3.0
	github.com/go-resty/resty/v2 v2.16.5
	github.com/gofrs/uuid/v5 v5.3.2
	github.com/golang-jwt/jwt/v5 v5.3.0
	github.com/google/cel-go v0.25.0
	github.com/googleapis/gax-go/v2 v2.15.0
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/websocket v1.5.4-0.20250319132907-e064f32e3674
	github.com/hashicorp/golang-lru/v2 v2.0.7
	github.com/icholy/digest v0.1.22
	github.com/klauspost/compress v1.18.1
	github.com/meraki/dashboard-api-go/v3 v3.0.9
	github.com/microsoft/go-mssqldb v1.9.3
	github.com/microsoft/wmi v0.34.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/elasticsearchexporter v0.139.0
	github.com/pierrec/lz4/v4 v4.1.22
	github.com/pkg/xattr v0.4.9
	github.com/prometheus/prometheus v0.307.1
	github.com/shirou/gopsutil/v4 v4.25.9
	github.com/teambition/rrule-go v1.8.2
	github.com/tklauser/go-sysconf v0.3.15
	github.com/tomnomnom/linkheader v0.0.0-20180905144013-02ca5825eb80
	github.com/xdg-go/scram v1.1.2
	github.com/zyedidia/generic v1.2.1
	go.elastic.co/apm/module/apmelasticsearch/v2 v2.7.1
	go.elastic.co/apm/module/apmhttp/v2 v2.7.1
	go.elastic.co/apm/v2 v2.7.1
	go.mongodb.org/mongo-driver v1.17.4
	go.opentelemetry.io/collector/component v1.45.0
	go.opentelemetry.io/collector/component/componentstatus v0.139.0
	go.opentelemetry.io/collector/config/configtls v1.45.0
	go.opentelemetry.io/collector/confmap v1.45.0
	go.opentelemetry.io/collector/confmap/provider/fileprovider v1.45.0
	go.opentelemetry.io/collector/consumer v1.45.0
	go.opentelemetry.io/collector/consumer/consumererror v0.139.0
	go.opentelemetry.io/collector/exporter/debugexporter v0.139.0
	go.opentelemetry.io/collector/otelcol v0.139.0
	go.opentelemetry.io/collector/pdata v1.45.0
	go.opentelemetry.io/collector/receiver v1.45.0
	go.uber.org/mock v0.5.0
	golang.org/x/term v0.37.0
	google.golang.org/genproto/googleapis/api v0.0.0-20250929231259-57b25ae835d4
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/apache/arrow-go/v18 v18.4.1
	github.com/cilium/ebpf v0.19.0
	github.com/elastic/gokrb5/v8 v8.0.0-20251105095404-23cc45e6a102
	github.com/forensicanalysis/fslib v0.15.2
	github.com/mattn/go-sqlite3 v1.14.32
	go.opentelemetry.io/collector/client v1.45.0
	go.opentelemetry.io/collector/component/componenttest v0.139.0
	go.opentelemetry.io/collector/config/configauth v1.45.0
	go.opentelemetry.io/collector/config/confighttp v0.139.0
	go.opentelemetry.io/collector/config/configoptional v1.45.0
	go.opentelemetry.io/collector/confmap/xconfmap v0.139.0
	go.opentelemetry.io/collector/consumer/consumertest v0.139.0
	go.opentelemetry.io/collector/exporter v1.45.0
	go.opentelemetry.io/collector/exporter/exportertest v0.139.0
	go.opentelemetry.io/collector/extension v1.45.0
	go.opentelemetry.io/collector/extension/extensionauth v1.45.0
	go.opentelemetry.io/collector/extension/extensiontest v0.139.0
	go.opentelemetry.io/collector/pipeline v1.45.0
	go.opentelemetry.io/collector/processor v1.45.0
	go.opentelemetry.io/collector/processor/processorhelper v0.139.0
	go.opentelemetry.io/collector/receiver/receivertest v0.139.0
	go.opentelemetry.io/otel/sdk/metric v1.38.0
	go.uber.org/goleak v1.3.0
	sigs.k8s.io/kind v0.29.0
	www.velocidex.com/golang/regparser v0.0.0-20250203141505-31e704a67ef7
)

require (
	al.essio.dev/pkg/shellescape v1.5.1 // indirect
	aqwari.net/xml v0.0.0-20210331023308-d9421b293817 // indirect
	cel.dev/expr v0.24.0 // indirect
	cloud.google.com/go/auth v0.16.5 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/compute/metadata v0.8.4 // indirect
	cloud.google.com/go/iam v1.5.2 // indirect
	cloud.google.com/go/longrunning v0.6.7 // indirect
	code.cloudfoundry.org/gofileutils v0.0.0-20170111115228-4d0c80011a0f // indirect
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/Azure/azure-amqp-common-go/v4 v4.2.0 // indirect
	github.com/Azure/azure-pipeline-go v0.2.3 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.11.2 // indirect
	github.com/Azure/go-amqp v1.3.0 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.24 // indirect
	github.com/Azure/go-autorest/autorest/to v0.4.1 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.3.2 // indirect
	github.com/Azure/go-autorest/logger v0.2.2 // indirect
	github.com/Azure/go-autorest/tracing v0.6.1 // indirect
	github.com/Azure/go-ntlmssp v0.0.0-20221128193559-754e69321358 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.6.0 // indirect
	github.com/BurntSushi/toml v1.4.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp v1.29.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric v0.51.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/internal/resourcemapping v0.51.0 // indirect
	github.com/VictoriaMetrics/easyproto v0.1.4 // indirect
	github.com/andybalholm/brotli v1.2.0 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/apache/arrow/go/v15 v15.0.2 // indirect
	github.com/apache/thrift v0.22.0 // indirect
	github.com/armon/go-radix v1.0.0 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.9 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.9 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.3 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.8.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.29.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.1 // indirect
	github.com/benbjohnson/clock v1.3.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bluekeyes/go-gitdiff v0.7.1 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cncf/xds/go v0.0.0-20250501225837-2ac532fd4443 // indirect
	github.com/containerd/containerd/v2 v2.1.0 // indirect
	github.com/containerd/errdefs v1.0.0 // indirect
	github.com/containerd/errdefs/pkg v0.3.0 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/coreos/go-systemd v0.0.0-20180511133405-39ca1b05acc7 // indirect
	github.com/cyphar/filepath-securejoin v0.2.5 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dgraph-io/ristretto/v2 v2.1.0 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/djherbis/times v1.5.0 // indirect
	github.com/dnephin/pflag v1.0.7 // indirect
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/eapache/go-xerial-snappy v0.0.0-20230731223053-c322873962e3 // indirect
	github.com/eapache/queue v1.1.0 // indirect
	github.com/ebitengine/purego v0.9.0 // indirect
	github.com/elastic/elastic-transport-go/v8 v8.7.0 // indirect
	github.com/elastic/go-docappender/v2 v2.11.3 // indirect
	github.com/elastic/go-windows v1.0.2 // indirect
	github.com/elazarl/goproxy v0.0.0-20240909085733-6741dbfc16a1 // indirect
	github.com/elazarl/goproxy/ext v0.0.0-20240909085733-6741dbfc16a1 // indirect
	github.com/emicklei/go-restful/v3 v3.12.2 // indirect
	github.com/envoyproxy/go-control-plane/envoy v1.35.0 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.2.1 // indirect
	github.com/evanphx/json-patch/v5 v5.6.0 // indirect
	github.com/fearful-symmetry/gomsr v0.0.1 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/foxboron/go-tpm-keyfiles v0.0.0-20250903184740-5d135037bd4d // indirect
	github.com/fxamacker/cbor/v2 v2.9.0 // indirect
	github.com/go-asn1-ber/asn1-ber v1.5.5 // indirect
	github.com/go-jose/go-jose/v4 v4.1.2 // indirect
	github.com/go-logfmt/logfmt v0.6.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/godror/knownpb v0.3.0 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.2 // indirect
	github.com/golang-sql/civil v0.0.0-20220223132316-b832511892a9 // indirect
	github.com/golang-sql/sqlexp v0.1.0 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/gnostic-models v0.7.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/go-tpm v0.9.6 // indirect
	github.com/google/licenseclassifier v0.0.0-20221004142553-c1ed8fcf4bab // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.6 // indirect
	github.com/grafana/regexp v0.0.0-20250905093917-f7b3be9d1853 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.2 // indirect
	github.com/hashicorp/cronexpr v1.1.3 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	github.com/hashicorp/golang-lru v0.6.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jcmturner/aescts/v2 v2.0.0 // indirect
	github.com/jcmturner/dnsutils/v2 v2.0.0 // indirect
	github.com/jcmturner/gofork v1.7.6 // indirect
	github.com/jcmturner/goidentity/v6 v6.0.1 // indirect
	github.com/jcmturner/gokrb5/v8 v8.4.4 // indirect
	github.com/jcmturner/rpc/v2 v2.0.3 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/klauspost/asmfmt v1.3.2 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/knadh/koanf/maps v0.1.2 // indirect
	github.com/knadh/koanf/providers/confmap v1.0.0 // indirect
	github.com/knadh/koanf/v2 v2.3.0 // indirect
	github.com/kortschak/utter v1.5.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/lestrrat-go/strftime v1.1.1 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-ieproxy v0.0.1 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mileusna/useragent v1.3.5 // indirect
	github.com/minio/asm2plan9s v0.0.0-20200509001527-cdd76441f9d8 // indirect
	github.com/minio/c2goasm v0.0.0-20190812172519-36a3d3bbc4f3 // indirect
	github.com/minio/sha256-simd v1.0.1 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/spdystream v0.5.0 // indirect
	github.com/moby/sys/atomicwriter v0.1.0 // indirect
	github.com/moby/sys/sequential v0.6.0 // indirect
	github.com/moby/sys/userns v0.1.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/montanaflynn/stats v0.7.1 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/mxk/go-flowrate v0.0.0-20140419014527-cca7078d478f // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/common v0.139.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal v0.139.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/prometheus/client_golang v1.23.2 // indirect
	github.com/prometheus/otlptranslator v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/rs/cors v1.11.1 // indirect
	github.com/segmentio/fasthash v1.0.3 // indirect
	github.com/sergi/go-diff v1.3.1 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/spiffe/go-spiffe/v2 v2.5.0 // indirect
	github.com/stoewer/go-strcase v1.3.1 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/tklauser/numcpus v0.10.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	github.com/zeebo/errs v1.4.0 // indirect
	github.com/zeebo/xxh3 v1.0.2 // indirect
	go.elastic.co/fastjson v1.5.1 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.0 // indirect
	go.opentelemetry.io/collector/config/configcompression v1.45.0 // indirect
	go.opentelemetry.io/collector/config/configmiddleware v1.45.0 // indirect
	go.opentelemetry.io/collector/config/configopaque v1.45.0 // indirect
	go.opentelemetry.io/collector/config/configretry v1.45.0 // indirect
	go.opentelemetry.io/collector/config/configtelemetry v0.139.0 // indirect
	go.opentelemetry.io/collector/connector v0.139.0 // indirect
	go.opentelemetry.io/collector/connector/connectortest v0.139.0 // indirect
	go.opentelemetry.io/collector/connector/xconnector v0.139.0 // indirect
	go.opentelemetry.io/collector/consumer/consumererror/xconsumererror v0.139.0 // indirect
	go.opentelemetry.io/collector/consumer/xconsumer v0.139.0 // indirect
	go.opentelemetry.io/collector/exporter/exporterhelper v0.139.0 // indirect
	go.opentelemetry.io/collector/exporter/exporterhelper/xexporterhelper v0.139.0 // indirect
	go.opentelemetry.io/collector/exporter/xexporter v0.139.0 // indirect
	go.opentelemetry.io/collector/extension/extensioncapabilities v0.139.0 // indirect
	go.opentelemetry.io/collector/extension/extensionmiddleware v0.139.0 // indirect
	go.opentelemetry.io/collector/extension/xextension v0.139.0 // indirect
	go.opentelemetry.io/collector/featuregate v1.45.0 // indirect
	go.opentelemetry.io/collector/internal/fanoutconsumer v0.139.0 // indirect
	go.opentelemetry.io/collector/internal/telemetry v0.139.0 // indirect
	go.opentelemetry.io/collector/pdata/pprofile v0.139.0 // indirect
	go.opentelemetry.io/collector/pdata/testdata v0.139.0 // indirect
	go.opentelemetry.io/collector/pdata/xpdata v0.139.0 // indirect
	go.opentelemetry.io/collector/pipeline/xpipeline v0.139.0 // indirect
	go.opentelemetry.io/collector/processor/processortest v0.139.0 // indirect
	go.opentelemetry.io/collector/processor/xprocessor v0.139.0 // indirect
	go.opentelemetry.io/collector/receiver/xreceiver v0.139.0 // indirect
	go.opentelemetry.io/collector/service v0.139.0 // indirect
	go.opentelemetry.io/collector/service/hostcapabilities v0.139.0 // indirect
	go.opentelemetry.io/contrib/bridges/otelzap v0.13.0 // indirect
	go.opentelemetry.io/contrib/detectors/gcp v1.36.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.61.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.63.0 // indirect
	go.opentelemetry.io/contrib/otelconf v0.18.0 // indirect
	go.opentelemetry.io/contrib/propagators/b3 v1.38.0 // indirect
	go.opentelemetry.io/ebpf-profiler v0.0.202540 // indirect
	go.opentelemetry.io/otel v1.38.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc v0.14.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp v0.14.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.38.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.38.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.38.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.38.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.38.0 // indirect
	go.opentelemetry.io/otel/exporters/prometheus v0.60.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutlog v0.14.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.38.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.38.0 // indirect
	go.opentelemetry.io/otel/log v0.14.0 // indirect
	go.opentelemetry.io/otel/metric v1.38.0 // indirect
	go.opentelemetry.io/otel/sdk v1.38.0 // indirect
	go.opentelemetry.io/otel/sdk/log v0.14.0 // indirect
	go.opentelemetry.io/otel/trace v1.38.0 // indirect
	go.opentelemetry.io/proto/otlp v1.7.1 // indirect
	go.uber.org/ratelimit v0.3.1 // indirect
	go.yaml.in/yaml/v2 v2.4.3 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/exp v0.0.0-20250911091902-df9299821621 // indirect
	golang.org/x/telemetry v0.0.0-20251008203120-078029d740a8 // indirect
	golang.org/x/xerrors v0.0.0-20240903120638-7835f813f4da // indirect
	gonum.org/v1/gonum v0.16.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251007200510-49b9836ed3ff // indirect
	gopkg.in/evanphx/json-patch.v4 v4.12.0 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
	k8s.io/kube-openapi v0.0.0-20250710124328-f3f2b991d03b // indirect
	k8s.io/utils v0.0.0-20250604170112-4c0f3b243397 // indirect
	kernel.org/pub/linux/libs/security/libcap/psx v1.2.57 // indirect
	mvdan.cc/garble v0.12.1 // indirect
	sigs.k8s.io/json v0.0.0-20241014173422-cfa47c3a1cc8 // indirect
	sigs.k8s.io/randfill v1.0.0 // indirect
	sigs.k8s.io/structured-merge-diff/v6 v6.3.0 // indirect
	sigs.k8s.io/yaml v1.6.0 // indirect
	www.velocidex.com/golang/go-ntfs v0.1.1 // indirect
)

require (
	cloud.google.com/go/storage v1.53.0
	github.com/PaloAltoNetworks/pango v0.10.2
	github.com/dlclark/regexp2 v1.4.0 // indirect
	github.com/moby/term v0.5.0 // indirect
	github.com/yuin/gopher-lua v1.1.1 // indirect
)

replace (
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/consumption/armconsumption => github.com/elastic/azure-sdk-for-go/sdk/resourcemanager/consumption/armconsumption v1.1.0-elastic
	github.com/apoydence/eachers => github.com/poy/eachers v0.0.0-20181020210610-23942921fe77 //indirect, see https://github.com/elastic/beats/pull/29780 for details.
	github.com/dop251/goja => github.com/elastic/goja v0.0.0-20190128172624-dd2ac4456e20
	github.com/fsnotify/fsevents => github.com/elastic/fsevents v0.0.0-20181029231046-e1d381a4d270
	github.com/fsnotify/fsnotify => github.com/elastic/fsnotify v1.6.1-0.20240920222514-49f82bdbc9e3
	github.com/google/gopacket => github.com/elastic/gopacket v1.1.20-0.20241002174017-e8c5fda595e6
	github.com/insomniacslk/dhcp => github.com/elastic/dhcp v0.0.0-20200227161230-57ec251c7eb3 // indirect
	github.com/meraki/dashboard-api-go/v3 => github.com/tommyers-elastic/dashboard-api-go/v3 v3.0.0-20250616163611-a325b49669a4
)
