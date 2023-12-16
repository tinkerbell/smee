package file

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/netip"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/tinkerbell/smee/internal/dhcp/data"
)

func TestNewWatcher(t *testing.T) {
	tests := map[string]struct {
		createFile bool
		want       string
		wantErr    error
	}{
		"contents equal": {createFile: true, want: "test content here"},
		"file not found": {createFile: false, wantErr: &fs.PathError{}},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var name string
			if tt.createFile {
				var err error
				name, err = createFile([]byte(tt.want))
				if err != nil {
					t.Fatal(err)
				}
				defer os.Remove(name)
			}
			w, err := NewWatcher(logr.Discard(), name)
			if (err != nil) != (tt.wantErr != nil) {
				t.Fatalf("NewWatcher() error = %v; type = %[1]T, wantErr %v; type = %[2]T", err, tt.wantErr)
			}
			var got string
			if tt.wantErr != nil {
				got = ""
			} else {
				got = string(w.data)
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func createFile(content []byte) (string, error) {
	file, err := os.CreateTemp("", "prefix")
	if err != nil {
		return "", err
	}
	defer file.Close()
	if _, err := file.Write(content); err != nil {
		return "", err
	}
	return file.Name(), nil
}

type testData struct {
	initial     string
	after       string
	action      string
	expectedOut string
}

func TestStartAndStop(t *testing.T) {
	tt := &testData{action: "cancel", expectedOut: `"level"=0 "msg"="stopping watcher"` + "\n"}
	out := &bytes.Buffer{}
	l := stdr.New(log.New(out, "", 0))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatal(err)
	}
	w := &Watcher{Log: l, watcher: watcher}
	w.Start(ctx)
	if diff := cmp.Diff(out.String(), tt.expectedOut); diff != "" {
		t.Fatal(diff)
	}
}

func TestStartFileUpdateError(t *testing.T) {
	tt := &testData{expectedOut: `"level"=0 "msg"="file changed, updating cache"` + "\n" + `"msg"="failed to read file" "error"="open not-found.txt: no such file or directory" "file"="not-found.txt"` + "\n" + `"level"=0 "msg"="stopping watcher"` + "\n"}
	out := &bytes.Buffer{}
	l := stdr.New(log.New(out, "", 0))
	got, name := tt.helper(t, l)
	defer os.Remove(name)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-time.After(time.Millisecond)
		got.FilePath = "not-found.txt"
		got.watcher.Events <- fsnotify.Event{Op: fsnotify.Write}
		cancel()
	}()
	got.Start(ctx)
	time.Sleep(time.Second)
	if diff := cmp.Diff(out.String(), tt.expectedOut); diff != "" {
		t.Fatal(diff)
	}
}

func TestStartFileUpdate(t *testing.T) {
	tt := &testData{initial: "once upon a time", after: "\nhello world", expectedOut: "once upon a time\nhello world"}
	got, name := tt.helper(t, logr.Discard())
	defer os.Remove(name)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-time.After(time.Millisecond)
		got.fileMu.Lock()
		f, err := os.OpenFile(name, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			t.Log(err)
		}
		f.Write([]byte(tt.after))
		f.Close()
		got.fileMu.Unlock()
		time.Sleep(time.Millisecond)
		cancel()
	}()
	got.Start(ctx)
	got.dataMu.RLock()
	d := got.data
	got.dataMu.RUnlock()
	if diff := cmp.Diff(string(d), tt.expectedOut); diff != "" {
		t.Log(string(d))
		t.Fatal(diff)
	}
}

func TestStartFileUpdateClosedChan(t *testing.T) {
	out := &bytes.Buffer{}
	l := stdr.New(log.New(out, "", 0))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatal(err)
	}
	w := &Watcher{Log: l, watcher: watcher}
	go w.Start(ctx)
	close(w.watcher.Events)
	time.Sleep(time.Millisecond)
	if diff := cmp.Diff(out.String(), ""); diff != "" {
		t.Fatal(diff)
	}
}

func TestStartError(t *testing.T) {
	tt := &testData{expectedOut: `"level"=0 "msg"="error watching file" "err"="test error"` + "\n" + `"level"=0 "msg"="stopping watcher"` + "\n"}
	out := &bytes.Buffer{}
	l := stdr.New(log.New(out, "", 0))
	ctx, cancel := context.WithCancel(context.Background())
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatal(err)
	}
	w := &Watcher{Log: l, watcher: watcher}
	go func() {
		time.Sleep(time.Millisecond)
		w.watcher.Errors <- fmt.Errorf("test error")
		cancel()
	}()
	w.Start(ctx)
	if diff := cmp.Diff(out.String(), tt.expectedOut); diff != "" {
		t.Fatal(diff)
	}
}

func TestStartErrorContinue(t *testing.T) {
	out := &bytes.Buffer{}
	l := stdr.New(log.New(out, "", 0))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatal(err)
	}
	w := &Watcher{Log: l, watcher: watcher}
	go w.Start(ctx)
	close(w.watcher.Errors)
	time.Sleep(time.Millisecond)
	if diff := cmp.Diff(out.String(), ""); diff != "" {
		t.Fatal(diff)
	}
}

func (tt *testData) helper(t *testing.T, l logr.Logger) (*Watcher, string) {
	t.Helper()
	name, err := createFile([]byte(tt.initial))
	if err != nil {
		t.Fatal(err)
	}
	w, err := NewWatcher(l, name)
	if err != nil {
		t.Fatal(err)
	}
	w.dataMu.RLock()
	before := string(w.data)
	w.dataMu.RUnlock()
	if diff := cmp.Diff(before, tt.initial); diff != "" {
		t.Fatal("before", diff)
	}

	return w, name
}

func TestTranslate(t *testing.T) {
	input := dhcp{
		MACAddress:       []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05},
		IPAddress:        "192.168.2.150",
		SubnetMask:       "255.255.255.0",
		DefaultGateway:   "192.168.2.1",
		NameServers:      []string{"1.1.1.1", "8.8.8.8"},
		Hostname:         "test-server",
		DomainName:       "example.com",
		BroadcastAddress: "192.168.2.255",
		NTPServers:       []string{"132.163.96.2"},
		VLANID:           "100",
		LeaseTime:        86400,
		Arch:             "x86_64",
		DomainSearch:     []string{"example.com"},
		Netboot: netboot{
			AllowPXE:      true,
			IPXEScriptURL: "http://boot.netboot.xyz",
			IPXEScript:    "#!ipxe\nchain http://boot.netboot.xyz",
			Console:       "ttyS0",
			Facility:      "onprem",
		},
	}
	wantDHCP := &data.DHCP{
		MACAddress:       []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05},
		IPAddress:        netip.MustParseAddr("192.168.2.150"),
		SubnetMask:       net.IPv4Mask(255, 255, 255, 0),
		DefaultGateway:   netip.MustParseAddr("192.168.2.1"),
		NameServers:      []net.IP{{1, 1, 1, 1}, {8, 8, 8, 8}},
		Hostname:         "test-server",
		DomainName:       "example.com",
		BroadcastAddress: netip.MustParseAddr("192.168.2.255"),
		NTPServers:       []net.IP{{132, 163, 96, 2}},
		VLANID:           "100",
		LeaseTime:        86400,
		Arch:             "x86_64",
		DomainSearch:     []string{"example.com"},
	}
	wantNetboot := &data.Netboot{
		AllowNetboot:  true,
		IPXEScriptURL: &url.URL{Scheme: "http", Host: "boot.netboot.xyz"},
		IPXEScript:    "#!ipxe\nchain http://boot.netboot.xyz",
		Console:       "ttyS0",
		Facility:      "onprem",
	}
	w := &Watcher{Log: logr.Discard()}
	gotDHCP, gotNetboot, err := w.translate(input)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(gotDHCP, wantDHCP, cmpopts.IgnoreUnexported(netip.Addr{})); diff != "" {
		t.Error(diff)
	}
	if diff := cmp.Diff(gotNetboot, wantNetboot); diff != "" {
		t.Error(diff)
	}
}

func TestTranslateErrors(t *testing.T) {
	tests := map[string]struct {
		input   dhcp
		wantErr error
	}{
		"invalid IP":                {input: dhcp{IPAddress: "not an IP"}, wantErr: errParseIP},
		"invalid subnet mask":       {input: dhcp{IPAddress: "1.1.1.1", SubnetMask: "not a mask"}, wantErr: errParseSubnet},
		"invalid gateway":           {input: dhcp{IPAddress: "1.1.1.1", SubnetMask: "192.168.1.255", DefaultGateway: "not a gateway"}, wantErr: nil},
		"invalid broadcast address": {input: dhcp{IPAddress: "1.1.1.1", SubnetMask: "192.168.1.255"}, wantErr: nil},
		"invalid NameServers":       {input: dhcp{IPAddress: "1.1.1.1", SubnetMask: "192.168.1.255", NameServers: []string{"no good"}}, wantErr: nil},
		"invalid ntpservers":        {input: dhcp{IPAddress: "1.1.1.1", SubnetMask: "192.168.1.255", NTPServers: []string{"no good"}}, wantErr: nil},
		"invalid ipxe script url":   {input: dhcp{IPAddress: "1.1.1.1", SubnetMask: "255.255.255.0", Netboot: netboot{IPXEScriptURL: ":not a url"}}, wantErr: errParseURL},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			w := &Watcher{Log: stdr.New(log.New(os.Stdout, "", log.Lshortfile))}
			if _, _, err := w.translate(tt.input); !errors.Is(err, tt.wantErr) {
				t.Errorf("translate() = %T, want %T", err, tt.wantErr)
			}
		})
	}
}

func TestGetByMac(t *testing.T) {
	tests := map[string]struct {
		mac     net.HardwareAddr
		badData bool
		wantErr error
	}{
		"no record found":        {mac: net.HardwareAddr{0x00, 0x01, 0x02, 0x03, 0x04, 0x05}, wantErr: errRecordNotFound},
		"record found":           {mac: net.HardwareAddr{0x08, 0x00, 0x27, 0x29, 0x4e, 0x67}, wantErr: nil},
		"fail error translating": {mac: net.HardwareAddr{0x08, 0x00, 0x27, 0x29, 0x4e, 0x68}, wantErr: errParseIP},
		"fail parsing file":      {badData: true, wantErr: errFileFormat},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			data := "testdata/example.yaml"
			if tt.badData {
				var err error
				data, err = createFile([]byte("not a yaml file"))
				if err != nil {
					t.Fatal(err)
				}
				defer os.Remove(data)
			}
			w, err := NewWatcher(logr.Discard(), data)
			if err != nil {
				t.Fatal(err)
			}
			_, _, err = w.GetByMac(context.Background(), tt.mac)
			if !errors.Is(err, tt.wantErr) {
				t.Fatal(err)
			}
		})
	}
}

func TestGetByIP(t *testing.T) {
	tests := map[string]struct {
		ip      net.IP
		badData bool
		wantErr error
	}{
		"no record found":   {ip: net.IPv4(172, 168, 2, 1), wantErr: errRecordNotFound},
		"record found":      {ip: net.IPv4(192, 168, 2, 153), wantErr: nil},
		"fail parsing file": {badData: true, wantErr: errFileFormat},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			data := "testdata/example.yaml"
			if tt.badData {
				var err error
				data, err = createFile([]byte("not a yaml file"))
				if err != nil {
					t.Fatal(err)
				}
				defer os.Remove(data)
			}
			w, err := NewWatcher(logr.Discard(), data)
			if err != nil {
				t.Fatal(err)
			}
			_, _, err = w.GetByIP(context.Background(), tt.ip)
			if !errors.Is(err, tt.wantErr) {
				t.Fatal(err)
			}
		})
	}
}
