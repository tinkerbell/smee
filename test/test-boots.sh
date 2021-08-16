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
# send -V PXEClient and option 93 set to 0 to get boots to accept this as a PXE
# DHCP client
busybox udhcpc -s /busybox-udhcpc-script.sh -V PXEClient -x 0x5d:0000

# set boot_file variable ahead of sourcing dhcpoffer-vars.sh to please the linter
boot_file=""

# the busybox script writes the DHCP variables to /tmp/dhcpoffer-vars.sh
# shellcheck disable=SC1091
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
tftp 192.168.99.42 -c get "$boot_file"

# sleep a long time so you can enter the container with
# docker exec -ti boots_client_1 /bin/sh
sleep 30000
