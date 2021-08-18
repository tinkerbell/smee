#!/bin/sh
# the docker-compose overrides the boots container's ENTRYPOINT
# with this script so it's a little easier to debug things
#
# configuration environment variables are provided by docker-compose

# for example, to see the DHCP packets coming from the DHCP client
# container, uncomment these.
# apk update && apk add --no-cache tcpdump
# tcpdump -nvvei eth0 port 67 or port 68 &
# or just apk add tcpdump then run this in another terminal:
#     docker exec -ti boots_boots_1 tcpdump -nvvei eth0 port 67 or port 68

# start boots and explicitly bind DHCP to broadcast address otherwise
# boots will start up fine but not see the DHCP requests
# TODO: probably move boots to just use the envvars for otel
/usr/bin/boots --dhcp-addr 0.0.0.0:67 &

sleep 100000
