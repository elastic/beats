module github.com/elastic/beats/v7/x-pack/otel/extension/beatsauthextension

go 1.25.8

require (
	github.com/elastic/beats/v7 v7.0.0-alpha
	github.com/elastic/elastic-agent-libs v0.33.3
	github.com/elastic/gokrb5/v8 v8.0.0-20251105095404-23cc45e6a102
	github.com/stretchr/testify v1.11.1
	go.elastic.co/apm/module/apmelasticsearch/v2 v2.7.2
	go.opentelemetry.io/collector/component v1.54.0
	go.opentelemetry.io/collector/component/componentstatus v0.148.0
	go.opentelemetry.io/collector/component/componenttest v0.148.0
	go.opentelemetry.io/collector/config/configauth v1.54.0
	go.opentelemetry.io/collector/config/confighttp v0.148.0
	go.opentelemetry.io/collector/config/configoptional v1.54.0
	go.opentelemetry.io/collector/extension v1.54.0
	go.opentelemetry.io/collector/extension/extensionauth v1.54.0
	go.uber.org/goleak v1.3.0
	go.uber.org/zap v1.27.1
	google.golang.org/grpc v1.79.3
)

require (
	github.com/armon/go-radix v1.0.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/elastic/go-sysinfo v1.15.3 // indirect
	github.com/elastic/go-ucfg v0.8.8 // indirect
	github.com/elastic/go-windows v1.0.2 // indirect
	github.com/fatih/color v1.16.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/foxboron/go-tpm-keyfiles v0.0.0-20251226215517-609e4778396f // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/google/go-tpm v0.9.8 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/hashicorp/go-version v1.8.0 // indirect
	github.com/jcmturner/aescts/v2 v2.0.0 // indirect
	github.com/jcmturner/dnsutils/v2 v2.0.0 // indirect
	github.com/jcmturner/gofork v1.7.6 // indirect
	github.com/jcmturner/goidentity/v6 v6.0.1 // indirect
	github.com/jcmturner/rpc/v2 v2.0.3 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.18.4 // indirect
	github.com/knadh/koanf/maps v0.1.2 // indirect
	github.com/knadh/koanf/providers/confmap v1.0.0 // indirect
	github.com/knadh/koanf/v2 v2.3.3 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/pierrec/lz4/v4 v4.1.26 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/procfs v0.20.1 // indirect
	github.com/rs/cors v1.11.1 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect
	go.elastic.co/apm/module/apmhttp/v2 v2.7.2 // indirect
	go.elastic.co/apm/v2 v2.7.2 // indirect
	go.elastic.co/ecszap v1.0.2 // indirect
	go.elastic.co/fastjson v1.5.1 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/collector/client v1.54.0 // indirect
	go.opentelemetry.io/collector/config/configcompression v1.54.0 // indirect
	go.opentelemetry.io/collector/config/configmiddleware v1.54.0 // indirect
	go.opentelemetry.io/collector/config/confignet v1.54.0 // indirect
	go.opentelemetry.io/collector/config/configopaque v1.54.0 // indirect
	go.opentelemetry.io/collector/config/configtls v1.54.0 // indirect
	go.opentelemetry.io/collector/confmap v1.54.0 // indirect
	go.opentelemetry.io/collector/confmap/xconfmap v0.148.0 // indirect
	go.opentelemetry.io/collector/extension/extensionmiddleware v0.148.0 // indirect
	go.opentelemetry.io/collector/featuregate v1.54.0 // indirect
	go.opentelemetry.io/collector/internal/componentalias v0.148.0 // indirect
	go.opentelemetry.io/collector/pdata v1.54.0 // indirect
	go.opentelemetry.io/collector/pipeline v1.54.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.67.0 // indirect
	go.opentelemetry.io/otel v1.42.0 // indirect
	go.opentelemetry.io/otel/metric v1.42.0 // indirect
	go.opentelemetry.io/otel/sdk v1.42.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.42.0 // indirect
	go.opentelemetry.io/otel/trace v1.42.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.48.0 // indirect
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260226221140-a57be14db171 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	howett.net/plist v1.0.1 // indirect
)

replace github.com/elastic/beats/v7 => ../../../..
