#!/usr/bin/env sh
# shellcheck disable=SC2039

set -xo pipefail

# osie_download from url and save it to directory
osie_download() {
	local url="$1"
	local directory="$2"
	wget "${url}" -O "${directory}"/osie.tar.gz
}

# osie_extract from tarball and save it to directory
osie_extract() {
	local source_dir="$1"
	local dest_dir="$2"
	tar -zxvf "${source_dir}"/osie.tar.gz -C "${dest_dir}" --strip-components 1
}

# osie_move_helper_scripts moves workflow helper scripts to the workflow directory
osie_move_helper_scripts() {
	local source_dir="$1"
	local dest_dir="$2"
	cp "${source_dir}"/workflow-helper.sh "${source_dir}"/workflow-helper-rc "${dest_dir}"/
}

# main runs the functions in order to download, extract, and move helper scripts
main() {
	local url="$1"
	local extract_dir="$2"
	local source_dir="$3"
	local dest_dir="$4"

	if [ ! -f "${extract_dir}"/osie.tar.gz ]; then
		echo "downloading osie..."
		osie_download "${url}" "${extract_dir}"
	else
		echo "osie already downloaded"
	fi
	if [ ! -f "${source_dir}"/workflow-helper.sh ] && [ ! -f "${source_dir}"/workflow-helper-rc ]; then
		echo "extracting osie..."
		osie_extract "${extract_dir}" "${source_dir}"
	else
		echo "osie files already exist, not extracting"
	fi
	osie_move_helper_scripts "${source_dir}" "${dest_dir}"
}

main "$1" "$2" "$3" "$4"
