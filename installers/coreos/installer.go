package coreos

import (
	"strings"

	"github.com/tinkerbell/boots/env"
	"github.com/tinkerbell/boots/files/ignition"
	"github.com/tinkerbell/boots/job"
)

func getInstallOpts(j job.Job, channel, facilityCode string) string {
	distro := j.OperatingSystem().Distro
	base := map[bool]string{
		true:  "http://install." + facilityCode + ".packet.net/" + distro + "/arm64-usr/" + channel,
		false: "http://install." + facilityCode + ".packet.net/" + distro + "/amd64-usr/" + channel,
	}
	args := []string{
		"-V current",
		"-C " + channel,
		"-b " + base[j.IsARM()],
	}

	if !j.IsARM() {
		args = append(args, "-o packet")
	}

	if strings.HasPrefix(distro, "flatcar") {
		args = append(args, "-s")
	} else {
		disk := "/dev/sda"
		if strings.Contains(j.PlanSlug(), "s1.large") {
			disk = "/dev/sdo"
		}
		args = append(args, "-d "+disk)
	}

	return strings.Join(args, " ")
}

func configureInstaller(j job.Job, u *ignition.SystemdUnit) {
	distro := j.OperatingSystem().Distro
	u.AddSection("Unit", "Requires=network-online.target", "After=network-online.target")

	var channel string
	var facilityCode string
	if os := j.OperatingSystem(); os != nil {
		channel = os.Version
	}
	if channel == "" {
		channel = "alpha"
	}
	facilityCode = j.FacilityCode()
	if facilityCode == "" {
		facilityCode = env.FacilityCode
	}

	var console string
	if j.IsARM() {
		console = "console=ttyAMA0,115200"
	} else {
		console = "console=tty0 console=ttyS1,115200n8"
	}

	installOpts := getInstallOpts(j, channel, facilityCode)
	lines := []string{
		// Install to disk:
		`/usr/bin/curl -H "Content-Type: application/json" -X POST -d '{"type":"provisioning.106"}' ${phone_home_url}`,
		"/usr/bin/" + distro + "-install " + installOpts,
		"/usr/bin/udevadm settle",
		"/usr/bin/mkdir -p /oemmnt",
		"/usr/bin/mount /dev/disk/by-label/OEM /oemmnt",
		`/usr/bin/bash -c "/usr/bin/echo \"set linux_console=\\\"` + console + `\\\"\" >> /oemmnt/grub.cfg"`,
		`/usr/bin/curl -H "Content-Type: application/json" -X POST -d '{"type":"provisioning.109"}' ${phone_home_url}`,
		"/usr/bin/systemctl reboot",
	}

	s := u.AddSection("Service", "Type=oneshot")
	for _, line := range lines {
		s.Add("ExecStart", line)
	}

	u.AddSection("Install", "WantedBy=multi-user.target")
	u.Enable()
}
