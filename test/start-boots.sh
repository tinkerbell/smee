#!/bin/sh
# the docker-compose overrides the boots container's ENTRYPOINT
# with this script so it's a little easier to debug things
#
# configuration environment variables are provided by docker-compose

# for example, to see the DHCP packets coming from the DHCP client
# container, uncomment these.
#apk update && apk add --no-cache tcpdump
#tcpdump -ni eth0 &

# start boots and explicitly bind DHCP to broadcast address otherwise
# boots will start up fine but not see the DHCP requests
/usr/bin/boots --dhcp-addr 0.0.0.0:67
