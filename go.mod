module github.com/tinkerbell/boots

go 1.16

require (
	github.com/andreyvit/diff v0.0.0-20170406064948-c7f18ee00883
	github.com/avast/retry-go v2.2.0+incompatible
	github.com/davecgh/go-spew v1.1.1
	github.com/equinix-labs/otel-init-go v0.0.1
	github.com/gammazero/workerpool v0.0.0-20200311205957-7b00833861c6
	github.com/golang/groupcache v0.0.0-20190702054246-869f871628b6
	github.com/golang/mock v1.5.0
	github.com/google/go-cmp v0.5.6
	github.com/google/uuid v1.1.5
	github.com/hexops/gotextdiff v1.0.3
	github.com/packethost/cacher v0.0.0-20200825140532-0b62e6726807
	github.com/packethost/dhcp4-go v0.0.0-20190402165401-39c137f31ad3
	github.com/packethost/pkg v0.0.0-20210325161133-868299771ae0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.6.0
	github.com/sebest/xff v0.0.0-20160910043805-6c115e0ffa35
	github.com/stretchr/testify v1.7.0
	github.com/tinkerbell/tftp-go v0.0.0-20200825172122-d9200358b6cd
	github.com/tinkerbell/tink v0.0.0-20201109122352-0e8e57332303
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.21.0
	go.opentelemetry.io/otel v1.0.0-RC2
	go.opentelemetry.io/otel/trace v1.0.0-RC2
	go.uber.org/zap v1.16.0
	golang.org/x/crypto v0.0.0-20200820211705-5c72a883971a
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b // indirect
	golang.org/x/text v0.3.5 // indirect
	golang.org/x/tools v0.1.0
	google.golang.org/genproto v0.0.0-20200921165018-b9da36f5f452 // indirect
	google.golang.org/grpc v1.39.0
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776 // indirect
)

replace github.com/sebest/xff v0.0.0-20160910043805-6c115e0ffa35 => github.com/packethost/xff v0.0.0-20190305172552-d3e9190c41b3
