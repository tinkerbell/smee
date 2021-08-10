package custom_ipxe

import (
	"os"
	"regexp"
	"testing"

	l "github.com/packethost/pkg/log"
	"github.com/stretchr/testify/require"
	"github.com/tinkerbell/boots/installers"
	"github.com/tinkerbell/boots/ipxe"
	"github.com/tinkerbell/boots/job"
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
		installerData string
		want          string
	}{
		{
			"invalid config",
			"",
			`#!ipxe

			echo Failed to decode installer data
			shell
			`,
		},
		{
			"valid config",
			`{"chain": "http://url/path.ipxe"}`,
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := require.New(t)
			mockJob := job.NewMock(t, "test.slug", "test.facility")
			script := ipxe.NewScript()

			mockJob.SetOSInstaller("ipxe")
			mockJob.SetOSInstallerData(tc.installerData)

			ipxeScript(mockJob.Job(), script)

			assert.Equal(dedent(tc.want), string(script.Bytes()))
		})
	}
}

func TestIpxeScriptFromConfig(t *testing.T) {
	var testCases = []struct {
		name   string
		config *Config
		want   string
	}{
		{
			"invalid config",
			&Config{},
			`#!ipxe

			echo Invalid ipxe configuration
			shell
			`,
		},
		{
			"valid chain",
			&Config{Chain: "http://url/path.ipxe"},
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
			&Config{Script: "echo my test script"},
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
			&Config{Script: "#!ipxe\necho my test script"},
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

			IpxeScriptFromConfig(testLogger, tc.config, mockJob.Job(), script)

			assert.Equal(dedent(tc.want), string(script.Bytes()))
		})
	}
}

func TestIpxeConfigFromJob(t *testing.T) {
	var testCases = []struct {
		name          string
		installerData string
		want          *Config
		expectError   string
	}{
		{
			"valid chain",
			`{"chain": "http://url/path.ipxe"}`,
			&Config{Chain: "http://url/path.ipxe"},
			"",
		},
		{
			"valid script",
			`{"script": "echo script"}`,
			&Config{Script: "echo script"},
			"",
		},
		{
			"empty json error",
			``,
			nil,
			"EOF",
		},
		{
			"invalid json error",
			`{"error"`,
			nil,
			"unexpected EOF",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := require.New(t)

			mockJob := job.NewMock(t, "test.slug", "test.facility")

			mockJob.SetOSInstallerData(tc.installerData)

			cfg, err := ipxeConfigFromJob(mockJob.Job())

			if tc.expectError == "" {
				assert.Nil(err)
			} else {
				assert.EqualError(err, tc.expectError)
			}

			assert.Equal(tc.want, cfg)
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

			cfg := &Config{
				Chain:  tc.chain,
				Script: tc.script,
			}

			got := cfg.validate()

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
