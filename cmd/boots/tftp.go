package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"regexp"

	"github.com/avast/retry-go"
	"github.com/packethost/pkg/log"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/tinkerbell/boots/conf"
	"github.com/tinkerbell/boots/job"
	"github.com/tinkerbell/boots/metrics"
	tftp "github.com/tinkerbell/tftp-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var tftpAddr = conf.TFTPBind

func init() {
	flag.StringVar(&tftpAddr, "tftp-addr", tftpAddr, "IP and port to listen on for TFTP.")
}

// ServeTFTP is a useless comment
func ServeTFTP() {
	err := retry.Do(
		func() error {
			return errors.Wrap(tftp.ListenAndServe(tftpAddr, tftpHandler{}), "serving tftp")
		},
	)
	if err != nil {
		mainlog.Fatal(errors.Wrap(err, "retry tftp serve"))
	}
}

type tftpHandler struct{}

func (t tftpHandler) ReadFile(c tftp.Conn, filename string) (tftp.ReadCloser, error) {
	labels := prometheus.Labels{"from": "tftp", "op": "read"}
	metrics.JobsTotal.With(labels).Inc()
	metrics.JobsInProgress.With(labels).Inc()
	timer := prometheus.NewTimer(metrics.JobDuration.With(labels))
	defer timer.ObserveDuration()
	defer metrics.JobsInProgress.With(labels).Dec()

	ip := tftpClientIP(c.RemoteAddr())
	filename = path.Base(filename)
	l := mainlog.With("client", ip.String(), "event", "open", "filename", filename)

	// clients can send traceparent over TFTP by appending the traceparent string
	// to the end of the filename they really want
	longfile := filename // hang onto this to report in traces
	ctx, shortfile, err := extractTraceparentFromFilename(context.Background(), filename)
	if err != nil {
		l.Info(err)
	}
	if shortfile != filename {
		l = l.With("filename", shortfile) // flip to the short filename in logs
		l.Info("client requested filename '", filename, "' with a traceparent attached and has been shortened to '", shortfile, "'")
		filename = shortfile
	}
	tracer := otel.Tracer("TFTP")
	ctx, span := tracer.Start(ctx, "TFTP get",
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(attribute.String("filename", filename)),
		trace.WithAttributes(attribute.String("requested-filename", longfile)),
		trace.WithAttributes(attribute.String("IP", ip.String())),
	)

	span.AddEvent("job.CreateFromIP")

	j, err := job.CreateFromIP(ctx, ip)
	if err != nil {
		l.With("error", errors.WithMessage(err, "retrieved job is empty")).Info()
		span.SetStatus(codes.Error, "no existing job: "+err.Error())
		span.End()

		return serveFakeReader(l, filename)
	}

	// This gates serving PXE file by
	// 1. the existence of a hardware record in tink server
	// AND
	// 2. the network.interfaces[].netboot.allow_pxe value, in the tink server hardware record, equal to true
	// This allows serving custom ipxe scripts, starting up into OSIE or other installation environments
	// without a tink workflow present.
	if !j.AllowPxe() {
		l.Info("the hardware data for this machine, or lack there of, does not allow it to pxe; allow_pxe: false")
		span.SetStatus(codes.Error, "allow_pxe is false")
		span.End()

		return serveFakeReader(l, filename)
	}

	span.SetStatus(codes.Ok, filename)
	span.End()

	return j.ServeTFTP(filename, ip.String())
}

// extractTraceparentFromFilename takes a context and filename and checks the filename for
// a traceparent tacked onto the end of it. If there is a match, the traceparent is extracted
// and a new SpanContext is contstructed and added to the context.Context that is returned.
// The filename is shortened to just the original filename so the rest of boots tftp can
// carry on as usual.
func extractTraceparentFromFilename(ctx context.Context, filename string) (context.Context, string, error) {
	// traceparentRe captures 4 items, the original filename, the trace id, span id, and trace flags
	traceparentRe := regexp.MustCompile("^(.*)-[[:xdigit:]]{2}-([[:xdigit:]]{32})-([[:xdigit:]]{16})-([[:xdigit:]]{2})")
	parts := traceparentRe.FindStringSubmatch(filename)
	if len(parts) == 5 {
		traceId, err := trace.TraceIDFromHex(parts[2])
		if err != nil {
			return ctx, filename, fmt.Errorf("parsing OpenTelemetry trace id %q failed: %s", parts[2], err)
		}

		spanId, err := trace.SpanIDFromHex(parts[3])
		if err != nil {
			return ctx, filename, fmt.Errorf("parsing OpenTelemetry span id %q failed: %s", parts[3], err)
		}

		// create a span context with the parent trace id & span id
		spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    traceId,
			SpanID:     spanId,
			Remote:     true,
			TraceFlags: trace.FlagsSampled, // TODO: use the parts[4] value instead
		})

		// inject it into the context.Context and return it along with the original filename
		return trace.ContextWithSpanContext(ctx, spanCtx), parts[1], nil
	} else {
		// no traceparent found, return everything as it was
		return ctx, filename, nil
	}
}

func serveFakeReader(l log.Logger, filename string) (tftp.ReadCloser, error) {
	switch filename {
	case "test.1mb":
		l.With("tftp_fake_read", true).Info()

		return &fakeReader{1 * 1024 * 1024}, nil
	case "test.8mb":
		l.With("tftp_fake_read", true).Info()

		return &fakeReader{8 * 1024 * 1024}, nil
	}
	l.With("error", errors.Wrap(os.ErrPermission, "access_violation")).Info()

	return nil, os.ErrPermission
}

func (tftpHandler) WriteFile(c tftp.Conn, filename string) (tftp.WriteCloser, error) {
	ip := tftpClientIP(c.RemoteAddr())
	err := errors.Wrap(os.ErrPermission, "access_violation")
	mainlog.With("client", ip, "event", "create", "filename", filename, "error", err).Info()

	return nil, os.ErrPermission
}

func tftpClientIP(addr net.Addr) net.IP {
	switch a := addr.(type) {
	case *net.IPAddr:
		return a.IP
	case *net.UDPAddr:
		return a.IP
	case *net.TCPAddr:
		return a.IP
	}

	host, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		err = errors.Wrap(err, "parse host:port")
		mainlog.Error(err)

		return nil
	}
	l := mainlog.With("host", host)

	if ip := net.ParseIP(host); ip != nil {
		l.With("ip", ip).Info()
		if v4 := ip.To4(); v4 != nil {
			ip = v4
		}

		return ip
	}
	l.Info("returning nil")

	return nil
}

var zeros = make([]byte, 1456)

type fakeReader struct {
	N int
}

func (r *fakeReader) Close() error {
	return nil
}

func (r *fakeReader) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return
	}
	if len(p) > r.N {
		p = p[:r.N]
	}

	for len(p) > 0 {
		n = copy(p, zeros)
		r.N -= n
		p = p[n:]
	}

	if r.N == 0 {
		err = io.EOF
	}

	return
}
