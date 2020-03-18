module github.com/packethost/boots

go 1.13

require (
	github.com/andreyvit/diff v0.0.0-20170406064948-c7f18ee00883
	github.com/avast/retry-go v2.2.0+incompatible
	github.com/beorn7/perks v0.0.0-20180321164747-3a771d992973
	github.com/betawaffle/tftp-go v0.0.0-20160921192434-dc649c1318ff
	github.com/davecgh/go-spew v1.1.1
	github.com/golang/groupcache v0.0.0-20180513044358-24b0969c4cb7
	github.com/golang/protobuf v1.3.1
	github.com/google/uuid v0.0.0-20161128191214-064e2069ce9c
	github.com/grpc-ecosystem/go-grpc-middleware v1.0.0
	github.com/jteeuwen/go-bindata v3.0.8-0.20180305030458-6025e8de665b+incompatible
	github.com/kylelemons/godebug v0.0.0-20170820004349-d65d576e9348
	github.com/matttproud/golang_protobuf_extensions v1.0.1
	github.com/packethost/cacher v0.0.0-20190610185035-b82ef56d72a3
	github.com/packethost/dhcp4-go v0.0.0-20190402165401-39c137f31ad3
	github.com/packethost/pkg v0.0.0-20190715213007-7c3a64b4b5e3
	github.com/packethost/xff v0.0.0-20190305172552-d3e9190c41b3
	github.com/pkg/errors v0.8.1
	github.com/pmezard/go-difflib v1.0.0
	github.com/prometheus/client_golang v0.9.2
	github.com/prometheus/client_model v0.0.0-20190129233127-fd36f4220a90
	github.com/prometheus/common v0.2.0
	github.com/prometheus/procfs v0.0.0-20190306233201-d0f344d83b0c
	github.com/rollbar/rollbar-go v1.0.2
	github.com/sergi/go-diff v1.0.0
	github.com/stretchr/testify v1.3.1-0.20190219160739-3f658bd5ac42
	go.uber.org/atomic v1.3.2
	go.uber.org/multierr v1.1.0
	go.uber.org/zap v1.9.1
	golang.org/x/crypto v0.0.0-20180525160159-a3beeb748656
	golang.org/x/net v0.0.0-20180524181706-dfa909b99c79
	golang.org/x/text v0.3.0
	google.golang.org/genproto v0.0.0-20180709204101-e92b11657268
	google.golang.org/grpc v1.13.0
)

replace github.com/sebest/xff d3e9190c41b3bfb920320cff4f6db3e03bc4a232 => github.com/packethost/xff v0.0.0-20190305172552-d3e9190c41b3
