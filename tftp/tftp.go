package tftp

import (
	_ "embed"
	"io"
	"net"
	"os"
	"time"

	"github.com/packethost/pkg/log"
	"github.com/pkg/errors"
)

//go:embed ipxe/ipxe.efi
var ipxeEFI []byte

//go:embed ipxe/undionly.kpxe
var undionly []byte

//go:embed ipxe/snp-nolacp.efi
var snpNolacp []byte

//go:embed ipxe/snp-hua.efi
var snpHua []byte

var tftpFiles = map[string][]byte{
	"undionly.kpxe":  undionly,
	"snp-nolacp.efi": snpNolacp,
	"ipxe.efi":       ipxeEFI,
	"snp-hua.efi":    snpHua,
}

type tftpTransfer struct {
	log.Logger
	unread []byte
	start  time.Time
}

// Open sets up a tftp transfer object that implements tftpgo.ReadCloser
func Open(mac net.HardwareAddr, filename, client string) (*tftpTransfer, error) {
	l := tftplog.With("mac", mac, "client", client, "filename", filename)

	content, ok := tftpFiles[filename]
	if !ok {
		err := errors.Wrap(os.ErrNotExist, "unknown file")
		l.With("event", "open", "error", err).Info()

		return nil, err
	}

	t := &tftpTransfer{
		Logger: l,
		unread: content,
		start:  time.Now(),
	}

	t.With("event", "open").Debug()

	return t, nil
}

func (t *tftpTransfer) Close() error {
	d := time.Since(t.start)
	n := len(t.unread)

	t.With("event", "close", "duration", d, "unread", n).Info()

	t.unread = nil

	return nil
}

func (t *tftpTransfer) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		t.With("event", "read", "read", 0, "unread", len(t.unread)).Info()

		return
	}

	n = copy(p, t.unread)
	t.unread = t.unread[n:]

	if len(t.unread) == 0 {
		err = io.EOF
	}

	t.With("event", "read", "read", n, "unread", len(t.unread)).Debug()

	return
}

func (t *tftpTransfer) Size() int {
	return len(t.unread)
}
