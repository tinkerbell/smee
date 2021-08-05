package coreos

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tinkerbell/boots/files/ignition"
	"github.com/tinkerbell/boots/job"
)

func assertLines(t *testing.T, m job.Mock, execLines []string) {
	assert := require.New(t)
	su := ignition.SystemdUnits{}
	configureInstaller(m.Job(), su.Add("install.service"))
	_, err := su.MarshalJSON()
	assert.Nil(err)
	assert.Equal(len(su), 1)
	unit := *su[0]
	assert.Equal(unit.Name, "install.service")
	assert.True(unit.Enabled)
	bytes, err := unit.Contents.MarshalText()
	assert.Nil(err)
	verifyLines := append(baseStart, execLines...)
	verifyLines = append(verifyLines, baseEnd...)
	assert.Equal(strings.Join(verifyLines, "\n"), string(bytes))
}

func TestInstaller(t *testing.T) {
	for _, distro := range []string{"coreos", "flatcar"} {
		for typ, execLines := range script {
			if distro == "flatcar" {
				execLines = replacer(execLines, "-d /dev/sda", "-s", "-d /dev/sdo", "-s")
			}
			t.Run(distro+"-"+typ, func(t *testing.T) {
				m := job.NewMock(t, typ, facility)
				m.SetOSDistro(distro)
				m.SetOSSlug(distro + "_alpha")
				m.SetOSVersion("alpha")
				for i := range execLines {
					execLines[i] = strings.Replace(execLines[i], "coreos", distro, -1)
				}
				assertLines(t, m, execLines)
			})
		}
	}
}

// this is the base set of starter commands for coreos installs
var baseStart = []string{
	"[Unit]",
	"Requires=systemd-networkd-wait-online.service",
	"After=systemd-networkd-wait-online.service",
	"",
	"[Service]",
	"Type=oneshot",
}

// this is the end of every coreos install
var baseEnd = []string{
	"ExecStart=/usr/bin/systemctl reboot",
	"",
	"[Install]",
	"WantedBy=multi-user.target",
	"",
}

var Exec = []string{
	`ExecStart=/usr/bin/curl --retry 10 -H "Content-Type: application/json" -X POST -d '{"type":"provisioning.106"}' ${phone_home_url}`,
	"ExecStart=/usr/bin/coreos-install -V current -C alpha -b http://install." + facility + ".packet.net/coreos/amd64-usr/alpha -o packet -d /dev/sda",
	"ExecStart=/usr/bin/udevadm settle",
	"ExecStart=/usr/bin/mkdir -p /oemmnt",
	"ExecStart=/usr/bin/mount /dev/disk/by-label/OEM /oemmnt",
	`ExecStart=/usr/bin/bash -c "/usr/bin/echo \"set linux_console=\\\"console=tty0 console=ttyS1,115200n8\\\"\" >> /oemmnt/grub.cfg"`,
	`ExecStart=/usr/bin/curl -H "Content-Type: application/json" -X POST -d '{"type":"provisioning.109"}' ${phone_home_url}`,
}

func replacer(l []string, replacements ...string) []string {
	if len(replacements)%2 != 0 {
		panic("replacements count must be even multiple of 2")
	}
	script := strings.Join(l, "\n")
	for i := 0; i < len(replacements); i = i + 2 {
		script = strings.ReplaceAll(script, replacements[i], replacements[i+1])
	}

	return strings.Split(script, "\n")
}

var script = map[string][]string{
	"baremetal_0":  Exec,
	"baremetal_1":  Exec,
	"s1.large.x86": replacer(Exec, "/dev/sda", "/dev/sdo"),
	"baremetal_2a": replacer(Exec, " -o packet", "", "tty0 console=ttyS1,115200n8", "ttyAMA0,115200", "amd64", "arm64"),
}
