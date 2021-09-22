package custom_ipxe

import (
	"context"
	"os"
	"regexp"
	"testing"

	l "github.com/packethost/pkg/log"
	"github.com/stretchr/testify/require"
	"github.com/tinkerbell/boots/installers"
	"github.com/tinkerbell/boots/ipxe"
	"github.com/tinkerbell/boots/job"
	"github.com/tinkerbell/boots/packet"
)

var (
	testLogger l.Logger
)

func TestMain(m *testing.M) {
	os.Setenv("PACKET_ENV", "test")
	os.Setenv("PACKET_VERSION", "0")
	os.Setenv("ROLLBAR_DISABLE", "1")
	os.Setenv("ROLLBAR_TOKEN", "1")

	logger, _ := l.Init("github.com/tinkerbell/boots")
	job.Init(logger)
	installers.Init(logger)
	testLogger = logger
	os.Exit(m.Run())
}

func TestIpxeScript(t *testing.T) {
	var testCases = []struct {
		name          string
		installer     string
		installerData *packet.InstallerData
		want          string
	}{
		{
			"installer: invalid config",
			"custom_ipxe",
			nil,
			`#!ipxe

			echo Installer data not provided
			shell
			`,
		},
		{
			"valid config",
			"custom_ipxe",
			&packet.InstallerData{Chain: "http://url/path.ipxe"},
			`#!ipxe


			params
			param body Device connected to DHCP system
			param type provisioning.104.01
			imgfetch ${tinkerbell}/phone-home##params
			imgfree

			set packet_facility test.facility
			set packet_plan test.slug
			chain --autofree http://url/path.ipxe
			`,
		},
		{
			"installer: valid config",
			"",
			&packet.InstallerData{Chain: "http://url/path.ipxe"},
			`#!ipxe


			params
			param body Device connected to DHCP system
			param type provisioning.104.01
			imgfetch ${tinkerbell}/phone-home##params
			imgfree

			set packet_facility test.facility
			set packet_plan test.slug
			chain --autofree http://url/path.ipxe
			`,
		},
		{
			"instance: no config",
			"",
			nil,
			`#!ipxe

			echo Unknown ipxe configuration
			shell
			`,
		},
		{
			"instance: ipxe script url",
			"",
			&packet.InstallerData{Chain: "http://url/path.ipxe"},
			`#!ipxe


			params
			param body Device connected to DHCP system
			param type provisioning.104.01
			imgfetch ${tinkerbell}/phone-home##params
			imgfree

			set packet_facility test.facility
			set packet_plan test.slug
			chain --autofree http://url/path.ipxe
			`,
		},
		{
			"instance: userdata script",
			"",
			&packet.InstallerData{Script: "#!ipxe\necho userdata script"},
			`#!ipxe


			params
			param body Device connected to DHCP system
			param type provisioning.104.01
			imgfetch ${tinkerbell}/phone-home##params
			imgfree

			set packet_facility test.facility
			set packet_plan test.slug

			echo userdata script
			`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := require.New(t)
			mockJob := job.NewMock(t, "test.slug", "test.facility")
			script := ipxe.NewScript()

			if tc.installer == "custom_ipxe" {
				mockJob.SetOSInstaller("custom_ipxe")
				mockJob.SetOSInstallerData(tc.installerData)
			} else if tc.installerData != nil {
				mockJob.SetIPXEScriptURL(tc.installerData.Chain)
				mockJob.SetUserData(tc.installerData.Script)
			}
			i := Installer{}
			bs := i.BootScript()(context.Background(), mockJob.Job(), *script)

			assert.Equal(dedent(tc.want), string(bs.Bytes()))
		})
	}
}

func TestIpxeScriptFromConfig(t *testing.T) {
	var testCases = []struct {
		name   string
		config *packet.InstallerData
		want   string
	}{
		{
			"invalid config",
			&packet.InstallerData{},
			`#!ipxe

			echo ipxe config URL or Script must be defined
			shell
			`,
		},
		{
			"valid chain",
			&packet.InstallerData{Chain: "http://url/path.ipxe"},
			`#!ipxe


			params
			param body Device connected to DHCP system
			param type provisioning.104.01
			imgfetch ${tinkerbell}/phone-home##params
			imgfree

			set packet_facility test.facility
			set packet_plan test.slug
			chain --autofree http://url/path.ipxe
			`,
		},
		{
			"valid script",
			&packet.InstallerData{Script: "echo my test script"},
			`#!ipxe


			params
			param body Device connected to DHCP system
			param type provisioning.104.01
			imgfetch ${tinkerbell}/phone-home##params
			imgfree

			set packet_facility test.facility
			set packet_plan test.slug
			echo my test script
			`,
		},
		{
			"valid script with header",
			&packet.InstallerData{Script: "#!ipxe\necho my test script"},
			`#!ipxe


			params
			param body Device connected to DHCP system
			param type provisioning.104.01
			imgfetch ${tinkerbell}/phone-home##params
			imgfree

			set packet_facility test.facility
			set packet_plan test.slug

			echo my test script
			`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := require.New(t)
			mockJob := job.NewMock(t, "test.slug", "test.facility")
			script := ipxe.NewScript()

			bs := ipxeScriptFromConfig(testLogger, tc.config, mockJob.Job(), *script)

			assert.Equal(dedent(tc.want), string(bs.Bytes()))
		})
	}
}

func TestConfigValidate(t *testing.T) {
	var testCases = []struct {
		name   string
		chain  string
		script string
		want   string
	}{
		{"error when empty", "", "", "ipxe config URL or Script must be defined"},
		{"using chain", "http://chain.url/script.ipxe", "", ""},
		{"using script", "", "#!ipxe\necho ipxe script", ""},
		{"using both", "http://path", "ipxe script", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := require.New(t)

			cfg := &packet.InstallerData{
				Chain:  tc.chain,
				Script: tc.script,
			}

			got := validateConfig(cfg)

			if tc.want == "" {
				assert.Nil(got)
			} else {
				assert.EqualError(got, tc.want)
			}
		})
	}
}

var dedentRegexp = regexp.MustCompile(`(?m)^[^\S\n]+`)

func dedent(s string) string {
	return dedentRegexp.ReplaceAllString(s, "")
}
