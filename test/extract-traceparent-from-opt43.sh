#!/bin/sh

# extract_traceparent_from_opt43 takes a hex string from busybox udhcpc's opt43
# and extracts sub-option 69 which is where we stuff the traceparent in binary,
# which busybox helpfully gives us in a hex string as $opt43
#
# PXE_DISCOVERY_CONTROL is 060108 (option 6, 1 byte long, value 8)
# traceparent is 451a (option 69, 26 bytes, value is tp)
#
# The DHCP spec says nothing about ordering and boots can be observed to serve
# options in a different order on different runs, so the option has to be fully
# parsed to get the right data.
#
# this would be way easier in perl/python but this needs to work in busybox
# msh and with busybox shell tools
#
# takes 1 argument, usually $opt43
# sets $opt43x69 to the hex traceparent
# sets $TRACEPARENT to the W3C-formatted traceparent string
# neither are exported by default
extract_traceparent_from_opt43 () {
  local hexdata=$1 ; shift
  opt43x69="" # in case the global is still set, empty it
  local strlen=$(echo -n "$hexdata" |wc -c)
  local offset=1 # cut(1) uses offsets starting at 1

  while [ $offset -lt $strlen ] ; do
    # extract the option number, 1 byte
    local opt_end=$((offset + 1))
    local hopt=$(echo -n $hexdata | cut -c "${offset}-${opt_end}")
    local opt=$(printf '%d' "0x$hopt")

    # extract the option value length, 1 byte
    local len_start=$((offset + 2))
    local len_end=$((offset + 3))
    local hlen=$(echo -n $hexdata | cut -c "${len_start}-${len_end}")
    local len=$(printf '%d' "0x$hlen")
    local bov=$((offset + 4)) # beginning of value
    local eov=$((bov + len * 2 - 1)) # end of value

    if [ $opt -eq 69 ] ; then
      # set global to the full tp hex data
      opt43x69=$(echo -n $hexdata | cut -c "${bov}-${eov}")

      # break out the sections of the traceparent to make a proper W3C tp string
      local ver=$(echo -n $opt43x69 | cut -c "1-2")       # 1 byte
      local trace_id=$(echo -n $opt43x69 | cut -c "3-34") # 16 bytes
      local span_id=$(echo -n $opt43x69 | cut -c "35-50") # 8 bytes
      local flags=$(echo -n $opt43x69 | cut -c "51-53")   # 1 byte

      # set TRACEPARENT to the W3C-formatted string
      TRACEPARENT="${ver}-${trace_id}-${span_id}-${flags}"
    fi

    # add to the offset:
    # 4 characters for opt and len e.g. 0601 (option 6, length 1)
    # len (is bytes) * 2 (bc hex) = chars of offset e.g. 08 (value is 8, 2 chars in hex)
    offset=$((4 + offset + len * 2))
    local next=$(echo -n $hexdata |cut -c "${offset}-$((offset + 1))")

    # opt43 always ends with 0xff so if the next byte is ff it's the end for sure
    if [ "x$next" == "xff" ] ; then
      break
    fi
  done
}

