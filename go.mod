module github.com/tinkerbell/boots

go 1.13

require (
	github.com/andreyvit/diff v0.0.0-20170406064948-c7f18ee00883
	github.com/avast/retry-go v2.2.0+incompatible
	github.com/betawaffle/tftp-go v0.0.0-20160921192434-dc649c1318ff
	github.com/gammazero/workerpool v0.0.0-20200311205957-7b00833861c6
	github.com/golang/groupcache v0.0.0-20180513044358-24b0969c4cb7
	github.com/google/uuid v1.1.1
	github.com/packethost/cacher v0.0.0-20200512205048-5253af131795
	github.com/packethost/dhcp4-go v0.0.0-20190402165401-39c137f31ad3
	github.com/packethost/pkg v0.0.0-20200422151836-417b049b48b1
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v1.6.0
	github.com/sebest/xff v0.0.0-20160910043805-6c115e0ffa35
	github.com/sergi/go-diff v1.0.0 // indirect
	github.com/stretchr/testify v1.4.0
	github.com/tinkerbell/tink v0.0.0-20200428163249-b654f8630288
	go.uber.org/zap v1.10.0
	golang.org/x/crypto v0.0.0-20200317142112-1b76d66859c6
	google.golang.org/grpc v1.28.0
)

replace github.com/sebest/xff v0.0.0-20160910043805-6c115e0ffa35 => github.com/packethost/xff v0.0.0-20190305172552-d3e9190c41b3

replace github.com/tinkerbell/tink v0.0.0-20200428163249-b654f8630288 => github.com/kdeng3849/tink v0.0.0-20200527155229-11965b00a6ad
