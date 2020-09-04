module github.com/tinkerbell/boots

go 1.13

require (
	github.com/andreyvit/diff v0.0.0-20170406064948-c7f18ee00883
	github.com/avast/retry-go v2.2.0+incompatible
	github.com/davecgh/go-spew v1.1.1
	github.com/gammazero/workerpool v0.0.0-20200311205957-7b00833861c6
	github.com/golang/groupcache v0.0.0-20190702054246-869f871628b6
	github.com/golang/mock v1.4.3
	github.com/google/uuid v1.1.1
	github.com/packethost/cacher v0.0.0-20200825140532-0b62e6726807
	github.com/packethost/dhcp4-go v0.0.0-20190402165401-39c137f31ad3
	github.com/packethost/pkg v0.0.0-20200903155310-0433e0605550
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v1.6.0
	github.com/sebest/xff v0.0.0-20160910043805-6c115e0ffa35
	github.com/sergi/go-diff v1.0.0 // indirect
	github.com/stretchr/testify v1.4.0
	github.com/tinkerbell/tftp-go v0.0.0-20200825172122-d9200358b6cd
	github.com/tinkerbell/tink v0.0.0-20200806104854-0f947d87c35f
	go.uber.org/zap v1.10.0
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
	golang.org/x/net v0.0.0-20200625001655-4c5254603344 // indirect
	google.golang.org/grpc v1.29.1
)

replace github.com/sebest/xff v0.0.0-20160910043805-6c115e0ffa35 => github.com/packethost/xff v0.0.0-20190305172552-d3e9190c41b3
