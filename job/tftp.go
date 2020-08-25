package job

import (
	tftpgo "github.com/packethost/tftp-go"
	"github.com/tinkerbell/boots/tftp"
)

func (j Job) ServeTFTP(filename, client string) (tftpgo.ReadCloser, error) {
	return tftp.Open(j.mac, filename, client)
}
