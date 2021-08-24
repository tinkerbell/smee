package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"reflect"
	"testing"

	"bou.ke/monkey"
	"github.com/google/go-cmp/cmp"
	"github.com/tinkerbell/boots/job"
	"github.com/tinkerbell/tftp-go"
	"go.opentelemetry.io/otel/trace"
)

type tftpConnTester struct{}

func (t tftpConnTester) LocalAddr() net.Addr {
	return &net.IPAddr{IP: []byte("127.0.0.1")}
}

func (t tftpConnTester) RemoteAddr() net.Addr {
	return &net.IPAddr{IP: []byte("127.0.0.1")}
}

func TestReadFile(t *testing.T) {
	fileContents := "you downloaded a tftp file"
	tests := map[string]struct {
		expectedResp     string
		allowPxe         bool
		failCreateFromIP bool
		err              error
	}{
		"success":                {expectedResp: fileContents, allowPxe: true},
		"fail failCreateFromIP":  {failCreateFromIP: true, err: fmt.Errorf("permission denied")},
		"fail allowPxe is false": {err: fmt.Errorf("permission denied")},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			monkey.Patch(job.CreateFromIP, func(_ context.Context, _ net.IP) (job.Job, error) {
				var err error
				if tc.failCreateFromIP {
					err = tc.err
				}

				return job.Job{}, err
			})
			var jb job.Job
			monkey.PatchInstanceMethod(reflect.TypeOf(jb), "AllowPxe", func(_ job.Job) bool {
				return tc.allowPxe
			})
			monkey.PatchInstanceMethod(reflect.TypeOf(jb), "ServeTFTP", func(_ job.Job, _, _ string) (tftp.ReadCloser, error) {
				r := ioutil.NopCloser(bytes.NewReader([]byte(fileContents)))

				return r, nil
			})

			tf := tftpHandler{}
			tftpConn := tftpConnTester{}
			f, err := tf.ReadFile(tftpConn, "nofile")
			if err != nil {
				if tc.err != nil {
					if diff := cmp.Diff(tc.err.Error(), err.Error()); diff != "" {
						t.Fatal(diff)
					}
				} else {
					t.Fatal(err)
				}
			} else {
				buf := new(bytes.Buffer)
				buf.ReadFrom(f)
				if diff := cmp.Diff(tc.expectedResp, buf.String()); diff != "" {
					t.Fatal(diff)
				}
			}
		})
	}
}

func TestExtractTraceparentFromFilename(t *testing.T) {
	tests := map[string]struct {
		fileIn  string
		fileOut string
		err     error
		spanId  string
		traceId string
	}{
		"do nothing when no tp": {fileIn: "undionly.ipxe", fileOut: "undionly.ipxe", err: nil},
		"ignore bad filename": {
			fileIn:  "undionly.ipxe-00-0000-0000-00",
			fileOut: "undionly.ipxe-00-0000-0000-00",
			err:     nil,
		},
		"ignore corrupt tp": {
			fileIn:  "undionly.ipxe-00-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-abcdefghijklmnop-01",
			fileOut: "undionly.ipxe-00-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-abcdefghijklmnop-01",
			err:     nil,
		},
		"extract tp": {
			fileIn:  "undionly.ipxe-00-23b1e307bb35484f535a1f772c06910e-d887dc3912240434-01",
			fileOut: "undionly.ipxe",
			err:     nil,
			spanId:  "d887dc3912240434",
			traceId: "23b1e307bb35484f535a1f772c06910e",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			ctx, outfile, err := extractTraceparentFromFilename(ctx, tc.fileIn)
			if err != tc.err {
				t.Errorf("filename %q should have resulted in error %q but got %q", tc.fileIn, tc.err, err)
			}
			if outfile != tc.fileOut {
				t.Errorf("filename %q should have resulted in %q but got %q", tc.fileIn, tc.fileOut, outfile)
			}

			if tc.spanId != "" {
				sc := trace.SpanContextFromContext(ctx)
				got := sc.SpanID().String()
				if tc.spanId != got {
					t.Errorf("got incorrect span id from context, expected %q but got %q", tc.spanId, got)
				}
			}

			if tc.traceId != "" {
				sc := trace.SpanContextFromContext(ctx)
				got := sc.TraceID().String()
				if tc.traceId != got {
					t.Errorf("got incorrect trace id from context, expected %q but got %q", tc.traceId, got)
				}
			}
		})
	}
}
