#!/bin/sh

echo "starting DHCP in 5 seconds"

# useful for debugging sometimes
#tcpdump -ni eth0 &

sleep 5

# run a mainstream DHCP client in debug mode
#dhclient -4 -d
dhcpcd -d -4 --nobackground --noipv4ll -T

# over time we can add some tests here, including stepping through tftp and http
# requests to boots

