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
