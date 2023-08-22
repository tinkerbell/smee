#!/bin/sh
# the docker-compose overrides the smee container's ENTRYPOINT
# with this script so it's a little easier to debug things
#
# configuration environment variables are provided by docker-compose

# for example, to see the DHCP packets coming from the DHCP client
# container, uncomment these.
# apk update && apk add --no-cache tcpdump
# tcpdump -nvvei eth0 port 67 or port 68 &
# or just apk add tcpdump then run this in another terminal:
#     docker exec -ti smee_smee_1 tcpdump -nvvei eth0 port 67 or port 68

# start smee and explicitly bind DHCP to broadcast address otherwise
# smee will start up fine but not see the DHCP requests
# TODO: probably move smee to just use the envvars for otel
/usr/bin/smee &

sleep 100000
