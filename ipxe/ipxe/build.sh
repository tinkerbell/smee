#!/usr/bin/env bash

set -e

# Deps on ubuntu
# apt-get -y --no-install-recommends install build-essential gcc-aarch64-linux-gnu git liblzma-dev

name=$(basename $1)
topdir=${1%%/*}
version=$2
short_version="$(echo $version | cut -c1-5)"
sed -i '/#define OCSP_CHECK/ d' $topdir/src/config/crypto.h
case $name in
undionly.kpxe)
	cp $topdir/src/config/local/general.undionly.h $topdir/src/config/local/general.h
	make -C $topdir/src VERSION_PATCH=255 EXTRAVERSION="+ ($short_version)" bin/undionly.kpxe
	;;
ipxe.efi)
	cp $topdir/src/config/local/general.efi.h $topdir/src/config/local/general.h
	make -C $topdir/src VERSION_PATCH=255 EXTRAVERSION="+ ($short_version)" bin-x86_64-efi/ipxe.efi
	;;
snp.efi)
	cp $topdir/src/config/local/general.aarch64-snp-nolacp.h $topdir/src/config/local/general.h
	# http://lists.ipxe.org/pipermail/ipxe-devel/2018-August/006254.html
	sed -i '/^WORKAROUND_CFLAGS/ s|^|#|' $topdir/src/arch/arm64/Makefile
	CROSS_COMPILE=aarch64-unknown-linux-gnu- make -C $topdir/src VERSION_PATCH=255 EXTRAVERSION="+ ($short_version)" bin-arm64-efi/snp.efi
	;;
*) echo "unknown target: $1" && exit 1 ;;
esac
