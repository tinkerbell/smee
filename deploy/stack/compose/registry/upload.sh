#!/usr/bin/env bash
# shellcheck disable=SC2001,SC2155,SC2046

set -xo pipefail

main() {
	local reg_user="$1"
	local reg_pw="$2"
	local reg_url="$3"
	local images_file="$4"
	# this confusing IFS= and the || is to capture the last line of the file if there is no newline at the end
	while IFS= read -r img || [ -n "${img}" ]; do
		# trim trailing whitespace
		local imgr="$(echo "${img}" | sed 's/ *$//g')"
		skopeo copy --all --dest-tls-verify=false --dest-creds="${reg_user}":"${reg_pw}" docker://"${imgr}" docker://"${reg_url}"/$(basename "${imgr}")
	done <"${images_file}"
}

main "$1" "$2" "$3" "$4"
