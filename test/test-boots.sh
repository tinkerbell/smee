#!/bin/sh

sleep_at_start=3

echo "starting DHCP in $sleep_at_start seconds"
set -x

# useful for debugging sometimes
#tcpdump -ni eth0 &

sleep $sleep_at_start

# run a mainstream DHCP client in debug mode
#dhclient -4 -d
dhcpcd -d -4 --nobackground --noipv4ll -T

# get the OpenTelemetry traceparent via HTTP header
TRACEPARENT=`curl -sSIX GET http://192.168.99.42 2>/dev/null |awk '/Traceparent:/{print $2}'`
export TRACEPARENT

# over time we can add some tests here, including stepping through tftp and http
# requests to boots

# sleep a long time so you can enter the container with
# docker exec -ti boots_client_1 /bin/sh
sleep 30000
