package vmware

import (
	"io"
	"net/http"
	"text/template"

	"github.com/pkg/errors"
	"github.com/tinkerbell/boots/conf"
	"github.com/tinkerbell/boots/installers"
	"github.com/tinkerbell/boots/job"
)

func ServeKickstart() func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		j, err := job.CreateFromRemoteAddr(req.Context(), req.RemoteAddr)
		if err != nil {
			installers.Logger("vmware").With("client", req.RemoteAddr).Error(err, "retrieved job is empty")
			w.WriteHeader(http.StatusNotFound)

			return
		}
		if err := genKickstart(j, w); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			j.Error(err)
		}
	}
}

func genKickstart(j job.Job, writer io.Writer) error {
	return errors.Wrap(tmpl.Execute(writer, j), "generating kickstart template")
}

func mustParseNew(name, text string) *template.Template {
	return template.Must(template.New(name).Funcs(helpers).Parse(text))
}

var tmpl = mustParseNew("kickstart", `
# Accept the VMware End User License Agreement
vmaccepteula
# Set the root password for the DCUI and Tech Support Mode
rootpw --iscrypted {{ rootpw . }}
# The install media is in the CD-ROM drive
{{- if (firstDisk .) }}
install --firstdisk={{ firstDisk . }} --overwritevmfs
{{- else }}
install --firstdisk --overwritevmfs
{{- end }}
# Set the network to DHCP on the proper network adapter based on its type
network --bootproto=dhcp --device={{ vmnic . }}
reboot

%firstboot --interpreter=busybox
echo "Packet firstboot executed" > /packet-firstboot.log
echo "Packet firstboot executed" > /var/log/packet-firstboot.log
# Fetch packet MD
wget http://metadata.packet.net/metadata -O /tmp/metadata
uuid=$(cat /tmp/metadata | python -c "import sys, json; print(json.load(sys.stdin)['id'])")
hostname=$(cat /tmp/metadata | python -c "import sys, json; print(json.load(sys.stdin)['hostname'])")
# Set hostname
esxcli system hostname set --fqdn=$hostname
# Enable shell
vim-cmd hostsvc/enable_esx_shell
vim-cmd hostsvc/start_esx_shell
# Add private network interface
esxcli network vswitch standard portgroup add --portgroup-name='Private Network' --vswitch-name=vSwitch0
esxcli network ip interface add --interface-name=vmk1 --portgroup-name='Private Network'
# Set the iSCSI IQN
iqn=$(cat /tmp/metadata | python -c "import sys, json; print(json.load(sys.stdin)['iqn'])")
esxcli iscsi software set --enabled=true
esxcli iscsi adapter set -A vmhba64 -n $iqn
esxcli iscsi networkportal add -n vmk1 -A vmhba64
# Configure IP addresses statically from metadata using python
cat >> /tmp/netcfg.py <<EOF
import sys
import json
import subprocess
def exec(cmd):
  print(cmd + "\n")
  subprocess.call(cmd, shell=True)
with open('/tmp/metadata', 'r') as json_file:
  packet_metadata = json.load(json_file)

try:
  netcfg_mgmt_type = packet_metadata['customdata']['network']['management_ip_type']
except:
  netcfg_mgmt_type = "Null"

private_subnets = packet_metadata.get('private_subnets')
if private_subnets is None:
  private_subnets = ['10.0.0.0/8']

if netcfg_mgmt_type == "public":
  print ("Custom data management IP type is set private for this instance...\n")

  for addr in packet_metadata['network']['addresses']:
    if addr['public'] == True:
      interface = "vmk0"
    else:
      next
    if addr['address_family'] == 4:
      exec("esxcli network ip interface ipv4 set -i " + interface + " -t static -I " + addr['address'] + " -N " + addr['netmask'] + " -g " + addr['gateway'])
    elif addr['address_family'] == 6:
      exec("esxcli network ip interface ipv6 set -i " + interface + " -e true")
      exec("esxcli network ip interface ipv6 address add -i " + interface + " -I " + addr['address'] + "/" + str(addr['cidr']))
      exec("esxcli network ip interface ipv6 set -i " + interface + " -g " + addr['gateway'])
    else:
      print("Skipping unknown address_family [" + addr['address_family'] +"]\n")

elif netcfg_mgmt_type == "private":
  print ("Custom data management IP type is set private for this instance...\n")

  for addr in packet_metadata['network']['addresses']:
    if addr['public'] == True:
      next
    else:
      interface = "vmk0"
    if addr['address_family'] == 4:
      interface = "vmk0"
      exec("esxcli network ip interface ipv4 set -i " + interface + " -t static -I " + addr['address'] + " -N " + addr['netmask'] + " -g " + addr['gateway'])
    elif addr['address_family'] == 6:
      exec("esxcli network ip interface ipv6 set -i " + interface + " -e true")
      exec("esxcli network ip interface ipv6 address add -i " + interface + " -I " + addr['address'] + "/" + str(addr['cidr']))
      exec("esxcli network ip interface ipv6 set -i " + interface + " -g " + addr['gateway'])
    else:
      print("Skipping unknown address_family [" + addr['address_family'] +"]\n")

    exec("esxcli network ip set --ipv6-enabled=false")

elif netcfg_mgmt_type == "both" or netcfg_mgmt_type == "Null":
  print ("Custom data management IP type is set " + netcfg_mgmt_type + ". Configuring both as default for this instance...\n")

  for addr in packet_metadata['network']['addresses']:
    if addr['public'] == True:
      interface = "vmk0"
    else:
      interface = "vmk1"
    if addr['address_family'] == 4:
      if interface == "vmk1":
        exec("esxcli network ip interface ipv4 set -i " + interface + " -t static -I " + addr['address'] + " -N " + addr['netmask'])
        for route in private_subnets:
          exec("esxcli network ip route ipv4 add --gateway " + addr['gateway'] + " --network " + route)
      else:
        exec("esxcli network ip interface ipv4 set -i " + interface + " -t static -I " + addr['address'] + " -N " + addr['netmask'] + " -g " + addr['gateway'])
    elif addr['address_family'] == 6:
      exec("esxcli network ip interface ipv6 set -i " + interface + " -e true")
      exec("esxcli network ip interface ipv6 address add -i " + interface + " -I " + addr['address'] + "/" + str(addr['cidr']))
      exec("esxcli network ip interface ipv6 set -i " + interface + " -g " + addr['gateway'])
    else:
      print("Skipping unknown address_family [" + addr['address_family'] +"]\n")

if netcfg_mgmt_type == "manual":
  print ("Custom data management IP type manual set for this instance...\n")
  network = packet_metadata['customdata']['network']
  exec("esxcli network ip interface ipv4 set -i " + network['interface'] + " -t static -I " + network['ipv4_address'] + " -N " + network['ipv4_netmask'] + " -g " + network['ipv4_gateway'])
  exec("esxcli network vswitch standard portgroup set -p=\"Management Network\" -v=" + network['ipv4_vlan'])
  exec("esxcli network ip set --ipv6-enabled=false")

if netcfg_mgmt_type == "dhcp":
  print ("Custom data management IP type DHCP set for this instance. Nothing to do...\n")

else:
  print ("Custom data management IP type NOT set for this instance...\n")
EOF
cat << 'EOF' > /tmp/customize.sh
#/bin/sh
metadata=/tmp/metadata

custom_data () {
        python -c "import json; print(json.load(open('$metadata'))['customdata']$1)" 2>/dev/null
        RESULT=$?
        if [ $RESULT -eq 0 ]; then
                return
        else
                echo "null"
        fi
}

set_root_pw() {
	echo "Setting rootpw"
	sed -i "s|^root:[^:]*|root:$1|" "$2"
}

## TODO: Consider validating customdata, but maybe the API is a better place for that

sshset=$(custom_data "['sshd']['enabled']")
sshpwauth=$(custom_data "['sshd']['pwauth']")
rootpwcrypt=$(custom_data "['rootpwcrypt']")
esxishellset=$(custom_data "['esxishell']['enabled']")
kickstartfburl=$(custom_data "['kickstart']['firstboot_url']")
kickstartfbshell=$(custom_data "['kickstart']['firstboot_shell']")
kickstartfbshellcmd=$(custom_data "['kickstart']['firstboot_shell_cmd']")

# SSHd config
if [ "$sshset" == "true" ]; then
	echo "Enabling SSHd"
	vim-cmd hostsvc/enable_ssh
	wget -q http://metadata.packet.net/2009-04-04/meta-data/public-keys -O /etc/ssh/keys-root/authorized_keys
elif [ "$sshset" == "false" ]; then
	echo "Disabling SSHd"
	vim-cmd hostsvc/disable_ssh
else
	echo "Skipping SSHd config"
fi

# SSHd pass auth config
if [ "$sshpwauth" == "true" ]; then
	echo "Enabling SSHd password auth"
	sed -i 's/ChallengeResponseAuthentication no/ChallengeResponseAuthentication yes/g' /etc/ssh/sshd_config
elif [ "$sshpwauth" == "false" ]; then
	echo "Disabling SSHd password auth and force keys (default)"
	echo 'ChallengeResponseAuthentication no' >> /etc/ssh/sshd_config
else
        echo "Skipping SSHd password auth config"
fi

# ESXishell config
if [ "$esxishellset" == "true" ]; then
	echo "Enabling ESXishell"
	vim-cmd hostsvc/enable_esx_shell
	vim-cmd hostsvc/start_esx_shell
elif [ "$esxishellset" == "false" ]; then
	echo "Disabling ESXishell"
	vim-cmd hostsvc/disable_esx_shell
	vim-cmd hostsvc/stop_esx_shell
else
	echo "Skipping ESXishell config"
fi

# Custom root pass
if [ "$rootpwcrypt" != "null" ]; then
	echo "Using custom root pass"
	set_root_pw "$rootpwcrypt" /etc/shadow
else
	echo "Skipping custom root pass"
fi

# Kickstart firstboot supplemental config URL
if [ "$kickstartfburl" != "null" ]; then
	echo "Using supplemental kickstart firstboot URL: $kickstartfburl"
	if wget -q "$kickstartfburl" -O /tmp/ks-firstboot-sup.sh; then
		echo "========Begin execution of supplemental firstboot kickstart"
		chmod +x /tmp/ks-firstboot-sup.sh && /tmp/ks-firstboot-sup.sh
		echo "========End execution of supplemental firstboot kickstart"
	else
		echo "ERROR: Custom kickstart firstboot URL '$kickstartfburl' is NOT accessible!"
		exit 1
	fi
else
	echo "Skipping supplemental kickstart firstboot URL"
fi

# Kickstart firstboot supplemental shell commands
if [ "$kickstartfbshellcmd" != "null" ]; then
        echo "Using kickstart firstboot shell command(s)"
	if [ "$kickstartfbshell" != "null" ]; then
		cmdshell="$kickstartfbshell"
#		echo "Shell kickstartfbshell is: $kickstartfbshell"
	else
		cmdshell = "/bin/sh -C"
	fi

	echo "$kickstartfbshellcmd" > /tmp/fbshell.sh
	chmod +x /tmp/fbshell.sh
	echo "========Begin execution of supplemental firstboot shell commands"
	cmdoutput=$($cmdshell /tmp/fbshell.sh)
	echo "${cmdoutput}"
	echo "========End execution of supplemental firstboot shell commands"
else
	echo "Skipping kickstart firstboot shell command(s)"
fi
EOF
python /tmp/netcfg.py
# Setup public SSH key auth for root
wget http://metadata.packet.net/2009-04-04/meta-data/public-keys -O /etc/ssh/keys-root/authorized_keys
# Disable SSH password auth and force public key auth
echo 'ChallengeResponseAuthentication no' >> /etc/ssh/sshd_config
# Enable ssh
vim-cmd hostsvc/enable_ssh
# Ensure serial port is activated
esxcli system settings kernel set -s logPort -v none
esxcli system settings kernel set -s gdbPort -v none
esxcli system settings kernel set -s tty2Port -v com2
# Execute customization script after the above vim-cmds, etc run as default
chmod +x /tmp/customize.sh
sh /tmp/customize.sh > /var/log/firstboot-customize.log
# Phone home to Packet for device activation
echo "Tinkerbell: {{ tink_host }}" > /tmp/firstboot-packet.log
echo "UUID: $uuid" >> /tmp/firstboot-packet.log
BODY='{"instance_id":"$uuid"}'
BODY_LEN=$( echo -n ${BODY} | wc -c )
echo -ne "POST /phone-home HTTP/1.0\r\nHost: {{ tink_host }}\r\nContent-Type: application/json\r\nContent-Length: ${BODY_LEN}\r\n\r\n${BODY}" | nc -i 3 {{ tink_host }} 80 > /tmp/firstboot-phone-home.log
reboot

%post --interpreter=busybox
cat << 'EOF' > /tmp/customize-pi.sh
#/bin/sh
metadata=/tmp/metadata
wget http://metadata.packet.net/metadata -O $metadata

custom_data () {
        python -c "import json; print(json.load(open('$metadata'))['customdata']$1)" 2>/dev/null
        RESULT=$?
        if [ $RESULT -eq 0 ]; then
                return
        else
                echo "null"
        fi
}

kickstartpiurl=$(custom_data "['kickstart']['postinstall_url']")
kickstartpishell=$(custom_data "['kickstart']['postinstall_shell']")
kickstartpishellcmd=$(custom_data "['kickstart']['postinstall_shell_cmd']")

# Kickstart postinstall supplemental config URL
if [ "$kickstartpiurl" != "null" ]; then
	echo "Using supplemental kickstart postinstall URL: $kickstartpiurl"
	if wget -q "$kickstartpiurl" -O /tmp/ks-postinstall-sup.sh; then
		echo "========Begin execution of supplemental postinstall kickstart"
		chmod +x /tmp/ks-postinstall-sup.sh && /tmp/ks-postinstall-sup.sh
		echo "========End execution of supplemental postinstall kickstart"
	else
		echo "ERROR: Custom kickstart postinstall URL '$kickstartpiurl' is NOT accessible!"
		exit 1
	fi
else
	echo "Skipping supplemental kickstart postinstall URL"
fi

# Kickstart postinstall supplemental shell commands
if [ "$kickstartpishellcmd" != "null" ]; then
        echo "Using kickstart postinstall shell command(s)"
        if [ "$kickstartpishell" != "null" ]; then
                cmdshell="$kickstartpishell"
        else
                cmdshell = "/bin/sh -C"
        fi

        echo "$kickstartpishellcmd" > /tmp/customize-pi-cmd.sh
        echo "========Begin execution of supplemental postinstall shell commands"
        $cmdshell /tmp/customize-pi-cmd.sh
        echo "========End execution of supplemental postinstall shell commands"
else
        echo "Skipping kickstart postinstall shell command(s)"
fi
EOF
esxcli system settings kernel set -s logPort -v none
esxcli system settings kernel set -s gdbPort -v none
esxcli system settings kernel set -s tty2Port -v com2
echo "nameserver 147.75.207.207" > /etc/resolv.conf
chmod +x /tmp/customize-pi.sh
sh /tmp/customize-pi.sh > /tmp/customize-pi.log
sleep 60
echo "Tinkerbell: {{ tink_host }}" > /tmp/post-packet.log
BODY='{"type":"provisioning.109"}'
BODY_LEN=$( echo -n ${BODY} | wc -c )
echo -ne "POST /phone-home HTTP/1.0\r\nHost: {{ tink_host }}\r\nContent-Type: application/json\r\nContent-Length: ${BODY_LEN}\r\n\r\n${BODY}" | nc -i 3 {{ tink_host }} 80 > /tmp/post-phone-home.log

%post --interpreter=busybox --ignorefailure=true
echo "Packet installation postinstall executed" > /packet-pi-ks.log
sleep 20

%post --interpreter=busybox --ignorefailure=true
echo "Packet installation postinstall executed" > /packet-pi-ks-nc.log
sleep 20

%pre --interpreter=busybox
BOOTOPTIONS=$(/sbin/bootOption -o)
echo $BOOTOPTIONS > /cmdline-bootoption
echo $BOOTOPTIONS > /tmp/pre-bootoptions
sleep 30
`)

var helpers = template.FuncMap{
	"vmnic":     vmnic,
	"rootpw":    rootpw,
	"firstDisk": firstDisk,
	"tink_host": func() string { return conf.PublicFQDN },
}

func vmnic(j job.Job) string {
	return j.PrimaryNIC().String()
}

func rootpw(j job.Job) string {
	return j.PasswordHash()
}

// firstDisk returns which disk to install onto - normally provided via metadata.
func firstDisk(j job.Job) string {
	// The metadata service did not return a boot drive, so use the hard-coded version
	if j.BootDriveHint() == "" {
		return equinixPlanDisk(j.PlanSlug(), j.PlanVersionSlug())
	}

	if j.PlanSlug() == "" {
		return j.BootDriveHint()
	}

	// TODO: Remove temporary Equinix-specific logic for whitelisting which plans to respect boot drive hints for
	switch j.PlanSlug()[0:1] {
	case "s", "w":
		return j.BootDriveHint()
	default:
		disk := equinixPlanDisk(j.PlanSlug(), j.PlanVersionSlug())
		if disk != "" {
			return disk
		}

		// Plan did not match our hard-coded list, let's try the boot drive hint (probably empty)
		return j.BootDriveHint()
	}
}

// equinixPlanDisk is an Equinix-specific fallback used to return the first disk if it wasn't provided via metadata
// TODO: Remove this function once the metadata is plumbed through everywhere.
func equinixPlanDisk(slug string, version string) string {
	switch slug {
	case "c1.small.x86", "s1.large.x86", "t1.small.x86", "x1.small.x86":
		return "vmw_ahci"
	case "c2.medium.x86", "g2.large.x86", "m2.xlarge.x86", "n2.xlarge.x86", "n2.xlarge.google", "x2.xlarge.x86":
		return "vmw_ahci,lsi_mr3,lsi_msgpt3"
	case "c3.medium.x86", "c3.small.x86", "m3.large.x86", "s3.xlarge.x86":
		switch version {
		case "c3.medium.x86.01":
			return "Micron_5100_MTFD,vmw_ahci"
		case "s3.xlarge.x86.01":
			return "KXG50ZNV256G_TOSHIBA,vmw_ahci"
		default:
			return "vmw_ahci,lsi_mr3,lsi_msgpt3"
		}
	case "m1.xlarge.x86":
		if version == "baremetal_2_04" {
			return "vmw_ahci"
		}

		return "lsi_mr3,lsi_msgpt3,vmw_ahci"

	case "c1.xlarge.x86":
		return "lsi_mr3,vmw_ahci"
	default:
		return ""
	}
}
