package iso

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"github.com/kdomanski/iso9660"
	"github.com/kdomanski/iso9660/util"
	"github.com/tinkerbell/smee/internal/dhcp/data"
)

const magicString = `464vn90e7rbj08xbwdjejmdf4it17c5zfzjyfhthbh19eij201hjgit021bmpdb9ctrc87x2ymc8e7icu4ffi15x1hah9iyaiz38ckyap8hwx2vt5rm44ixv4hau8iw718q5yd019um5dt2xpqqa2rjtdypzr5v1gun8un110hhwp8cex7pqrh2ivh0ynpm4zkkwc8wcn367zyethzy7q8hzudyeyzx3cgmxqbkh825gcak7kxzjbgjajwizryv7ec1xm2h0hh7pz29qmvtgfjj1vphpgq1zcbiiehv52wrjy9yq473d9t1rvryy6929nk435hfx55du3ih05kn5tju3vijreru1p6knc988d4gfdz28eragvryq5x8aibe5trxd0t6t7jwxkde34v6pj1khmp50k6qqj3nzgcfzabtgqkmeqhdedbvwf3byfdma4nkv3rcxugaj2d0ru30pa2fqadjqrtjnv8bu52xzxv7irbhyvygygxu1nt5z4fh9w1vwbdcmagep26d298zknykf2e88kumt59ab7nq79d8amnhhvbexgh48e8qc61vq2e9qkihzt1twk1ijfgw70nwizai15iqyted2dt9gfmf2gg7amzufre79hwqkddc1cd935ywacnkrnak6r7xzcz7zbmq3kt04u2hg1iuupid8rt4nyrju51e6uejb2ruu36g9aibmz3hnmvazptu8x5tyxk820g2cdpxjdij766bt2n3djur7v623a2v44juyfgz80ekgfb9hkibpxh3zgknw8a34t4jifhf116x15cei9hwch0fye3xyq0acuym8uhitu5evc4rag3ui0fny3qg4kju7zkfyy8hwh537urd5uixkzwu5bdvafz4jmv7imypj543xg5em8jk8cgk7c4504xdd5e4e71ihaumt6u5u2t1w7um92fepzae8p0vq93wdrd1756npu1pziiur1payc7kmdwyxg3hj5n4phxbc29x0tcddamjrwt260b0w`

func TestReqPathInvalid(t *testing.T) {
	tests := map[string]struct {
		isoURL     string
		statusCode int
	}{
		"invalid URL prefix": {isoURL: "invalid", statusCode: http.StatusNotFound},
		"invalid URL":        {isoURL: "http://invalid.:123/hook.iso", statusCode: http.StatusBadRequest},
		"no script or url":   {isoURL: "http://10.10.10.10:8080/aa:aa:aa:aa:aa:aa/invalid.iso", statusCode: http.StatusInternalServerError},
	}
	for name, tt := range tests {
		u, _ := url.Parse(tt.isoURL)
		t.Run(name, func(t *testing.T) {
			h := &Handler{
				parsedURL: u,
			}
			req := http.Request{
				Method: http.MethodGet,
				URL:    u,
			}

			got, err := h.RoundTrip(&req)
			got.Body.Close()
			if err != nil {
				t.Fatal(err)
			}
			if got.StatusCode != tt.statusCode {
				t.Fatalf("got response status code: %d, want status code: %d", got.StatusCode, tt.statusCode)
			}
		})
	}
}

func TestPatching(t *testing.T) {
	// create a small ISO file with the magic string
	// serve it with a http server
	// patch the ISO file
	// mount the ISO file and check if the magic string was patched

	wantGrubCfg := `set timeout=0
set gfxpayload=text
menuentry 'LinuxKit ISO Image' {
        linuxefi /kernel  facility=test console=ttyAMA0 console=ttyS0 console=tty0 console=tty1 console=ttyS1 syslog_host=127.0.0.1:514 grpc_authority=127.0.0.1:42113 tinkerbell_tls=false worker_id=de:ed:be:ef:fe:ed                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   text
        initrdefi /initrd.img
}
`

	// create a small ISO file with the magic string
	grubCfg := `set timeout=0
set gfxpayload=text
menuentry 'LinuxKit ISO Image' {
        linuxefi /kernel 464vn90e7rbj08xbwdjejmdf4it17c5zfzjyfhthbh19eij201hjgit021bmpdb9ctrc87x2ymc8e7icu4ffi15x1hah9iyaiz38ckyap8hwx2vt5rm44ixv4hau8iw718q5yd019um5dt2xpqqa2rjtdypzr5v1gun8un110hhwp8cex7pqrh2ivh0ynpm4zkkwc8wcn367zyethzy7q8hzudyeyzx3cgmxqbkh825gcak7kxzjbgjajwizryv7ec1xm2h0hh7pz29qmvtgfjj1vphpgq1zcbiiehv52wrjy9yq473d9t1rvryy6929nk435hfx55du3ih05kn5tju3vijreru1p6knc988d4gfdz28eragvryq5x8aibe5trxd0t6t7jwxkde34v6pj1khmp50k6qqj3nzgcfzabtgqkmeqhdedbvwf3byfdma4nkv3rcxugaj2d0ru30pa2fqadjqrtjnv8bu52xzxv7irbhyvygygxu1nt5z4fh9w1vwbdcmagep26d298zknykf2e88kumt59ab7nq79d8amnhhvbexgh48e8qc61vq2e9qkihzt1twk1ijfgw70nwizai15iqyted2dt9gfmf2gg7amzufre79hwqkddc1cd935ywacnkrnak6r7xzcz7zbmq3kt04u2hg1iuupid8rt4nyrju51e6uejb2ruu36g9aibmz3hnmvazptu8x5tyxk820g2cdpxjdij766bt2n3djur7v623a2v44juyfgz80ekgfb9hkibpxh3zgknw8a34t4jifhf116x15cei9hwch0fye3xyq0acuym8uhitu5evc4rag3ui0fny3qg4kju7zkfyy8hwh537urd5uixkzwu5bdvafz4jmv7imypj543xg5em8jk8cgk7c4504xdd5e4e71ihaumt6u5u2t1w7um92fepzae8p0vq93wdrd1756npu1pziiur1payc7kmdwyxg3hj5n4phxbc29x0tcddamjrwt260b0w text
        initrdefi /initrd.img
}
`
	writer, err := iso9660.NewWriter()
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Cleanup()

	f := strings.NewReader(grubCfg)

	fileToPatch := "EFI/BOOT/grub.cfg"
	if err := writer.AddFile(f, fileToPatch); err != nil {
		t.Fatal(err)
	}

	outputFile, err := os.OpenFile("output.iso", os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		t.Fatalf("failed to create file: %s", err)
	}

	defer os.Remove("output.iso")

	if err := writer.WriteTo(outputFile, "testvol"); err != nil {
		t.Fatalf("failed to write ISO image: %s", err)
	}
	if err := outputFile.Close(); err != nil {
		t.Fatalf("failed to close output file: %s", err)
	}

	// serve it with a http server
	hs := httptest.NewServer(http.FileServer(http.Dir(".")))
	defer hs.Close()

	// patch the ISO file
	u := hs.URL + "/output.iso"
	parsedURL, err := url.Parse(u)
	if err != nil {
		t.Fatal(err)
	}

	h := &Handler{
		Logger:             logr.FromSlogHandler(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{AddSource: true, Level: slog.LevelDebug})),
		Backend:            &mockBackend{},
		SourceISO:          u,
		ExtraKernelParams:  []string{},
		Syslog:             "127.0.0.1:514",
		TinkServerTLS:      false,
		TinkServerGRPCAddr: "127.0.0.1:42113",
		parsedURL:          parsedURL,
		MagicString:        magicString,
	}

	rurl := hs.URL + "/iso/de:ed:be:ef:fe:ed/output.iso"
	purl, _ := url.Parse(rurl)
	req := http.Request{
		Header: http.Header{},
		Method: http.MethodGet,
		URL:    purl,
	}
	res, err := h.RoundTrip(&req)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("got status code: %d, want status code: %d", res.StatusCode, http.StatusOK)
	}

	isoContents, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("patched.iso", isoContents, 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove("patched.iso")

	op, err := os.Open("patched.iso")
	if err != nil {
		t.Fatal(err)
	}
	defer op.Close()
	os.Mkdir("mnt", 0755)
	defer os.RemoveAll("mnt")

	// mount the ISO file and check if the magic string was patched
	if err := util.ExtractImageToDirectory(op, "./mnt"); err != nil {
		t.Fatal(err)
	}

	grubCfgFile, err := os.ReadFile("./mnt/efi/boot/grub.cfg")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(grubCfgFile))
	if diff := cmp.Diff(wantGrubCfg, string(grubCfgFile)); diff != "" {
		t.Fatalf("unexpected grub.cfg file: %s", diff)
	}
}

type mockBackend struct{}

func (m *mockBackend) GetByMac(context.Context, net.HardwareAddr) (*data.DHCP, *data.Netboot, error) {
	d := &data.DHCP{}
	n := &data.Netboot{
		Facility: "test",
	}
	return d, n, nil
}

func (m *mockBackend) GetByIP(context.Context, net.IP) (*data.DHCP, *data.Netboot, error) {
	d := &data.DHCP{}
	n := &data.Netboot{
		Facility: "test",
	}
	return d, n, nil
}
