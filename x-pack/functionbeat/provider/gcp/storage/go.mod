module github.com/elastic/beats/x-pack/functionbeat/provider/gcp/storage

go 1.16

require (
	cloud.google.com/go/functions v1.0.0 // indirect
	cloud.google.com/go/pubsub v1.17.0 // indirect
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/libtrust v0.0.0-20160708172513-aabc10ec26b7 // indirect
	github.com/elastic/beats/v7 v7.15.0
	gopkg.in/jcmturner/aescts.v1 v1.0.1 // indirect
	gopkg.in/jcmturner/dnsutils.v1 v1.0.1 // indirect
	gopkg.in/jcmturner/goidentity.v3 v3.0.0 // indirect
	gopkg.in/jcmturner/rpc.v1 v1.1.0 // indirect
)

replace github.com/Shopify/sarama => github.com/elastic/sarama v1.19.1-0.20210823122811-11c3ef800752
