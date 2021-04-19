package vmware

import (
	"io/ioutil"
	"net"
	"strings"
	"testing"

	"github.com/andreyvit/diff"
	"github.com/stretchr/testify/require"
	"github.com/tinkerbell/boots/conf"
	"github.com/tinkerbell/boots/job"
)

func TestDetermineDisk(t *testing.T) {
	assert := require.New(t)
	for typ, disk := range kickstartTypes {
		t.Run(typ, func(t *testing.T) {
			m := job.NewMock(t, typ, facility)
			gotDisk := determineDisk(m.Job())
			assert.Equal(disk, gotDisk)
		})
	}

}

func TestScriptKickstart(t *testing.T) {
	manufacturers := []string{"supermicro", "dell"}
	versions := []string{"vmware_esxi_6_0", "vmware_esxi_6_5", "vmware_esxi_6_7", "vmware_esxi_7_0"}
	assert := require.New(t)
	conf.PublicIPv4 = net.ParseIP("127.0.0.1")
	conf.PublicFQDN = "boots-test.example.com"

	for _, man := range manufacturers {
		t.Run(man, func(t *testing.T) {
			for _, ver := range versions {
				t.Run(ver, func(t *testing.T) {
					for typ, disk := range kickstartTypes {
						t.Run(typ, func(t *testing.T) {
							m := job.NewMock(t, typ, facility)
							m.SetManufacturer(man)
							m.SetOSSlug(ver)
							m.SetIP(net.ParseIP("127.0.0.1"))
							m.SetPassword("password")
							m.SetMAC("00:00:ba:dd:be:ef")

							var w strings.Builder
							genKickstart(m.Job(), &w)
							got := w.String()
							script := loadKickstart(disk, assert)
							assert.Equal(script, got, diff.LineDiff(script, got))
						})
					}
				})
			}
		})
	}
}

func loadKickstart(disk string, assert *require.Assertions) string {
	data, err := ioutil.ReadFile("testdata/vmware_base.txt")
	assert.Nil(err)
	return strings.Replace(string(data), "<DISK>", disk, 1)
}

var kickstartTypes = map[string]string{
	"baremetal_5":                  "--firstdisk",
	"c1.small.x86":                 "--firstdisk=vmw_ahci",
	"c1.xlarge.x86":                "--firstdisk=lsi_mr3,vmw_ahci",
	"c2.medium.x86":                "--firstdisk=vmw_ahci,lsi_mr3,lsi_msgpt3",
	"g2.large.x86":                 "--firstdisk=vmw_ahci,lsi_mr3,lsi_msgpt3",
	"m1.xlarge.x86":                "--firstdisk=lsi_mr3,lsi_msgpt3,vmw_ahci",
	"m1.xlarge.x86:baremetal_2_04": "--firstdisk=vmw_ahci",
	"m2.xlarge.x86":                "--firstdisk=vmw_ahci,lsi_mr3,lsi_msgpt3",
	"n2.xlarge.x86":                "--firstdisk=vmw_ahci,lsi_mr3,lsi_msgpt3",
	"n2.xlarge.google":             "--firstdisk=vmw_ahci,lsi_mr3,lsi_msgpt3",
	"s1.large.x86":                 "--firstdisk=vmw_ahci",
	"t1.small.x86":                 "--firstdisk=vmw_ahci",
	"x1.small.x86":                 "--firstdisk=vmw_ahci",
	"x2.xlarge.x86":                "--firstdisk=vmw_ahci,lsi_mr3,lsi_msgpt3",
}
