package tftp

import (
	"context"
	"io"
	"net"
	"time"

	"github.com/packethost/pkg/log"
	"github.com/tinkerbell/boots/ipxe"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type tftpTransfer struct {
	log.Logger
	unread []byte
	start  time.Time
}

// Open sets up a tftp transfer object that implements tftpgo.ReadCloser
func Open(ctx context.Context, mac net.HardwareAddr, filename, client string) (*tftpTransfer, error) {
	l := tftplog.With("mac", mac, "client", client, "filename", filename)
	span := trace.SpanFromContext(ctx)

	content, err := ipxe.Files.ReadFile(filename)
	if err != nil {
		l.With("event", "open", "error", err).Info()
		span.SetStatus(codes.Error, "unknown file")

		return nil, err
	}

	span.SetAttributes(attribute.Int("bytes", len(content)))

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
