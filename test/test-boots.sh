#!/bin/sh

sleep_at_start=3

echo "starting DHCP in $sleep_at_start seconds"
set -x

# useful for debugging sometimes
# tcpdump -ni eth0 &
# alternatively, only show DHCP and pretty print the packets
# tcpdump -nvvei eth0 port 67 or port 68 &

sleep $sleep_at_start

# busybox udhcpc will happily set arbitrary DHCP options and is easy
# to configure with a custom setup script to call on DHCPOFFER
#
# dummy setup script for -s is copied in by Dockerfile
# -q tells udhcpc to exit after getting a lease, otherwise it will keep generating new traces
# opt60 (-V PXEClient) pretend to be an Intel PXE client. required to be noticed by boots
# opt93 (-x 0x5d) set to 0 for "Intel x86PC" platform, required by boots
# opt97 (-x 0x61) sets the client guid (https://datatracker.ietf.org/doc/html/rfc4578#section-2.3)
#              first 8 octets should be zeroes to make boots happy (Intel PXE does this)
#              ID: 4a525bd43517df7f8b4799c18d (randomly generated and hard-coded here)
# [WIP] opt43 (-O 0x67) request option 43, which will contain the traceparent
busybox udhcpc \
    -q \
    -s /busybox-udhcpc-script.sh \
    -V PXEClient \
    -x 0x5d:0000 \
    -x 0x61:000000004a525bd43517df7f8b4799c18d \
    -O 0x67

# the busybox script writes the DHCP variables to /tmp/dhcpoffer-vars.sh
. /tmp/dhcpoffer-vars.sh

# extrace the traceparent from the boot_file name, which is a regular
# boot file with -$TRACEPARENT appended
TRACEPARENT=$(echo $boot_file |sed 's/^.*xe-//')
echo "export TRACEPARENT=$TRACEPARENT" > /etc/profile.d/boots-traceparent.sh
export TRACEPARENT

# fetch / from the server with the traceparent set
tp_header="Traceparent: $TRACEPARENT"
curl -H "$tp_header" http://192.168.99.42/install.ipxe
# TODO: test opportunity here: validate the returned traceparent matches the one in boot_file

# boot_file is set by the DHCP envvars
# boot_file should already have the traceparent appended
tftp 192.168.99.42 -c get $boot_file

# sleep a long time so you can enter the container with
# docker exec -ti boots_client_1 /bin/sh
sleep 30000
