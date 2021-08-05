#!/usr/bin/env sh
# shellcheck disable=SC2039,SC2155,SC2086

set -xo pipefail

# update_hw_ip_addr the hardware json with a specified IP address
update_hw_ip_addr() {
	local ip_address="$1"
	local hw_file="$2"
	sed -i "s/\"address\":.*,/\"address\": \"${ip_address}\",/" "${hw_file}"
}

# update_hw_mac_addr the hardware json with a specified MAC address
update_hw_mac_addr() {
	local mac_address="$1"
	local hw_file="$2"
	sed -i "s/\"mac\":.*,/\"mac\": \"${mac_address}\",/" "${hw_file}"
}

# hardware creates a hardware record in tink from the file_loc provided
hardware() {
	local file_loc="$1"
	tink hardware push --file "${file_loc}"
}

# update_template_img_ip the template yaml with a specified IP address
update_template_img_ip() {
	local ip_address="$1"
	local template_file="$2"
	sed -i "s,IMG_URL: \"http://.*,IMG_URL: \"http://${ip_address}:8080/focal-server-cloudimg-amd64.raw.gz\"," "${template_file}"
}

# template create a template record in tink from the file_loc provided
template() {
	local file_loc="$1"
	tink template create --file "${file_loc}"
}

# workflow creates a workflow record in tink from the hardware and template records
workflow() {
	local workflow_dir="$1"
	local mac_address="$2"
	local mac=$(echo "${mac_address}" | tr '[:upper:]' '[:lower:]')
	local template_id=$(tink template get --no-headers 2>/dev/null | grep -v "+" | cut -d" " -f2 | xargs)
	tink workflow create --template "${template_id}" --hardware "{\"device_1\":\"${mac}\"}" | tee "${workflow_dir}"/workflow_id.txt
	# write just the workflow id to a file. `|| true` is a failsafe in case the workflow creation fails
	sed -i 's/Created Workflow:  //g' ${workflow_dir}/workflow_id.txt || true
}

# workflow_exists checks if a workflow record exists in tink before creating a new one
workflow_exists() {
	local workflow_dir="$1"
	local mac_address="$2"
	if [ ! -f "${workflow_dir}"/workflow_id.txt ]; then
		workflow "${workflow_dir}" "${mac_address}"
		return 0
	fi
	local workflow_id=$(cat "${workflow_dir}"/workflow_id.txt)
	tink workflow get | grep -q "${workflow_id}"
	local result=$?
	if [ "${result}" -ne 0 ]; then
		workflow "${workflow_dir}" "${mac_address}"
	else
		echo "Workflow [$(cat "${workflow_dir}"/workflow_id.txt)] already exists"
	fi
}

# main runs the creation functions in order
main() {
	local hw_file="$1"
	local template_file="$2"
	local workflow_dir="$3"
	local ip_address="$4"
	local client_ip_address="$5"
	local client_mac_address="$6"

	update_hw_ip_addr "${client_ip_address}" "${hw_file}"
	update_hw_mac_addr "${client_mac_address}" "${hw_file}"
	hardware "${hw_file}"
	update_template_img_ip "${ip_address}" "${template_file}"
	template "${template_file}"
	workflow_exists "${workflow_dir}" "${client_mac_address}"
}

main "$1" "$2" "$3" "$4" "$5" "$6"
