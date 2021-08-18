#!/bin/sh
# instead of messing with the actual interface configuration
# this just dumps the environment variables to a file and stdout

env | grep -v '^[A-Z]' | sort | tee /tmp/dhcpoffer-vars.sh
