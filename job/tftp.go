package job

import (
	"context"

	"github.com/tinkerbell/boots/tftp"
	tftpgo "github.com/tinkerbell/tftp-go"
)

func (j Job) ServeTFTP(ctx context.Context, filename, client string) (tftpgo.ReadCloser, error) {
	return tftp.Open(ctx, j.mac, filename, client)
}
