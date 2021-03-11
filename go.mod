module github.com/tinkerbell/boots

go 1.13

require (
	bou.ke/monkey v1.0.2
	github.com/andreyvit/diff v0.0.0-20170406064948-c7f18ee00883
	github.com/avast/retry-go v2.2.0+incompatible
	github.com/davecgh/go-spew v1.1.1
	github.com/gammazero/workerpool v0.0.0-20200311205957-7b00833861c6
	github.com/golang/groupcache v0.0.0-20190702054246-869f871628b6
	github.com/golang/mock v1.4.4
	github.com/google/go-cmp v0.5.2
	github.com/google/uuid v1.1.2
	github.com/packethost/cacher v0.0.0-20200825140532-0b62e6726807
	github.com/packethost/dhcp4-go v0.0.0-20190402165401-39c137f31ad3
	github.com/packethost/pkg v0.0.0-20200903155310-0433e0605550
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v1.6.0
	github.com/sebest/xff v0.0.0-20160910043805-6c115e0ffa35
	github.com/stretchr/testify v1.6.1
	github.com/tinkerbell/tftp-go v0.0.0-20200825172122-d9200358b6cd
	github.com/tinkerbell/tink v0.0.0-20201109122352-0e8e57332303
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.16.0
	golang.org/x/crypto v0.0.0-20200820211705-5c72a883971a
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b // indirect
	golang.org/x/net v0.0.0-20200904194848-62affa334b73 // indirect
	golang.org/x/sys v0.0.0-20200826173525-f9321e4c35a6 // indirect
	golang.org/x/text v0.3.3 // indirect
	golang.org/x/tools v0.0.0-20200921190806-0f52b63a40e8 // indirect
	google.golang.org/genproto v0.0.0-20200921165018-b9da36f5f452 // indirect
	google.golang.org/grpc v1.32.0
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776 // indirect
	gotest.tools v2.2.0+incompatible
)

replace github.com/sebest/xff v0.0.0-20160910043805-6c115e0ffa35 => github.com/packethost/xff v0.0.0-20190305172552-d3e9190c41b3
