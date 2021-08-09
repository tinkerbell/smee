#!/usr/bin/env bash

set -xo pipefail

# update_csr will add the sans_ip to the csr
update_csr() {
	local sans_ip="$1"
	local csr_file="$2"
	sed -i "/\"hosts\".*/a \    \"${sans_ip}\"," "${csr_file}"
}

# cleanup will remove unneeded files
cleanup() {
	rm -rf ca-key.pem ca.csr ca.pem server.csr server.pem
}

# gen will generate the key and bundle
gen() {
	local bundle_destination="$1"
	local key_destination="$2"
	cfssl gencert -initca /code/tls/csr.json | cfssljson -bare ca -
	cfssl gencert -config /code/tls/ca-config.json -ca ca.pem -ca-key ca-key.pem -profile server /code/tls/csr.json | cfssljson -bare server
	cat server.pem ca.pem >"${bundle_destination}"
	mv server-key.pem "${key_destination}"
}

# main orchestrates the process
main() {
	local sans_ip="$1"
	local csr_file="/code/tls/csr.json"
	local bundle_file="/certs/${FACILITY:-onprem}/bundle.pem"
	local server_key_file="/certs/${FACILITY:-onprem}/server-key.pem"

	if ! grep -q "${sans_ip}" "${csr_file}"; then
		update_csr "${sans_ip}" "${csr_file}"
	else
		echo "IP ${sans_ip} already in ${csr_file}"
	fi
	if [ ! -f "${bundle_file}" ] && [ ! -f "${server_key_file}" ]; then
		gen "${bundle_file}" "${server_key_file}"
	else
		echo "Files [${bundle_file}, ${server_key_file}] already exist"
	fi
	cleanup
}

main "$1"
