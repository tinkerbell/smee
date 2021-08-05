#!/usr/bin/env bash

set -xo pipefail

install_deps() {
	apt -y update
	DEBIAN_FRONTEND=noninteractive apt -y install qemu-utils wget gzip
}

download_image() {
	local url="$1"
	wget "${url}"
}

img_to_raw() {
	local img_file="$1"
	local raw_file="$2"
	qemu-img convert "${img_file}" -O raw "${raw_file}"
}

compress_raw() {
	local raw_file="$1"
	gzip "${raw_file}"
}

cleanup() {
	local img_file="$1"
	rm -rf "${img_file}"
}

main() {
	local image_url="$1"
	local img_file="$2"
	local raw_file="$3"

	if [ ! -f "${raw_file}.gz" ]; then
		install_deps
		download_image "${image_url}"
		img_to_raw "${img_file}" "${raw_file}"
		compress_raw "${raw_file}"
		cleanup "${img_file}"
	fi
}

main "$1" "$2" "$3"
