package job

import (
	"github.com/tinkerbell/boots/tftp"
	tftpgo "github.com/tinkerbell/tftp-go"
)

func (j Job) ServeTFTP(filename, client string) (tftpgo.ReadCloser, error) {
	return tftp.Open(j.mac, filename, client)
}
