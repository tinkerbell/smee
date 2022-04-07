module github.com/tinkerbell/boots

go 1.18

require (
	github.com/andreyvit/diff v0.0.0-20170406064948-c7f18ee00883
	github.com/avast/retry-go v2.2.0+incompatible
	github.com/davecgh/go-spew v1.1.1
	github.com/equinix-labs/otel-init-go v0.0.5
	github.com/gammazero/workerpool v0.0.0-20200311205957-7b00833861c6
	github.com/go-logr/logr v1.2.2
	github.com/go-logr/zapr v1.2.2
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da
	github.com/golang/mock v1.5.0
	github.com/google/go-cmp v0.5.6
	github.com/google/uuid v1.1.5
	github.com/hexops/gotextdiff v1.0.3
	github.com/packethost/cacher v0.0.0-20200825140532-0b62e6726807
	github.com/packethost/dhcp4-go v0.0.0-20190402165401-39c137f31ad3
	github.com/packethost/pkg v0.0.0-20210325161133-868299771ae0
	github.com/peterbourgon/ff/v3 v3.1.2
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0
	github.com/sebest/xff v0.0.0-20160910043805-6c115e0ffa35
	github.com/stretchr/testify v1.7.0
	github.com/tinkerbell/ipxedust v0.0.0-20220406180840-46f16b8d8fb0
	github.com/tinkerbell/tink v0.0.0-20201109122352-0e8e57332303
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.21.0
	go.opentelemetry.io/otel v1.3.0
	go.opentelemetry.io/otel/trace v1.3.0
	go.uber.org/zap v1.19.1
	golang.org/x/crypto v0.0.0-20210921155107-089bfa567519
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/tools v0.1.10
	google.golang.org/genproto v0.0.0-20211021150943-2b146023228c // indirect
	google.golang.org/grpc v1.44.0
	inet.af/netaddr v0.0.0-20211027220019-c74959edd3b6
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff/v4 v4.1.2 // indirect
	github.com/cespare/xxhash/v2 v2.1.1 // indirect
	github.com/felixge/httpsnoop v1.0.2 // indirect
	github.com/gammazero/deque v0.0.0-20200227231300-1e9af0e52b46 // indirect
	github.com/go-logr/stdr v1.2.0 // indirect
	github.com/go-logr/zerologr v1.2.1 // indirect
	github.com/go-playground/locales v0.14.0 // indirect
	github.com/go-playground/universal-translator v0.18.0 // indirect
	github.com/go-playground/validator/v10 v10.9.0 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/pin/tftp v0.0.0-20210809155059-0161c5dd2e96 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.32.1 // indirect
	github.com/prometheus/procfs v0.6.0 // indirect
	github.com/rollbar/rollbar-go v1.0.2 // indirect
	github.com/rs/zerolog v1.26.0 // indirect
	github.com/sergi/go-diff v1.2.0 // indirect
	go.opentelemetry.io/contrib v0.22.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/internal/retry v1.3.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.3.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.3.0 // indirect
	go.opentelemetry.io/otel/internal/metric v0.21.0 // indirect
	go.opentelemetry.io/otel/metric v0.21.0 // indirect
	go.opentelemetry.io/otel/oteltest v1.0.0-RC2 // indirect
	go.opentelemetry.io/otel/sdk v1.3.0 // indirect
	go.opentelemetry.io/proto/otlp v0.11.0 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.7.0 // indirect
	go4.org/intern v0.0.0-20211027215823-ae77deb06f29 // indirect
	go4.org/unsafe/assume-no-moving-gc v0.0.0-20211027215541-db492cf91b37 // indirect
	golang.org/x/mod v0.6.0-dev.0.20220106191415-9b9b3d81d5e3 // indirect
	golang.org/x/net v0.0.0-20211123203042-d83791d6bcd9 // indirect
	golang.org/x/sys v0.0.0-20211029165221-6e7872819dc8 // indirect
	golang.org/x/term v0.0.0-20210916214954-140adaaadfaf // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)

replace github.com/sebest/xff v0.0.0-20160910043805-6c115e0ffa35 => github.com/packethost/xff v0.0.0-20190305172552-d3e9190c41b3
