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
# opt60 (-V PXEClient) pretend to be an Intel PXE client. required to be noticed by boots
# opt93 (-x 0x5d) set to 0 for "Intel x86PC" platform, required by boots
# opt97 (-x 0x61) sets the client guid (https://datatracker.ietf.org/doc/html/rfc4578#section-2.3)
#              first 8 octets should be zeroes to make boots happy (Intel PXE does this)
#              ID: 4a525bd43517df7f8b4799c18d (randomly generated and hard-coded here)
busybox udhcpc \
    -s /busybox-udhcpc-script.sh \
    -V PXEClient \
    -x 0x5d:0000 \
    -x 0x61:000000004a525bd43517df7f8b4799c18d

# the busybox script writes the DHCP variables to /tmp/dhcpoffer-vars.sh
. /tmp/dhcpoffer-vars.sh

# get the OpenTelemetry traceparent via HTTP header
# TODO(@tobert): replace this with the DHCP traceparent once I reimplement that
TRACEPARENT=$(curl -sSIX GET http://192.168.99.42 2>/dev/null | awk '/Traceparent:/{print $2}')
export TRACEPARENT

# boot_file is set by the DHCP envvars
# try to fetch the boot file with traceparent
# TODO(@tobert) coming soon...
#echo "fetching ${boot_file}-${TRACEPARENT} over tftp..."
#tftp 192.168.99.42 -c get "${boot_file}-${TRACEPARENT}"
tftp 192.168.99.42 -c get $boot_file

# sleep a long time so you can enter the container with
# docker exec -ti boots_client_1 /bin/sh
sleep 30000
