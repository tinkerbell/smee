package iso

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	diskfs "github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/filesystem/iso9660"
	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
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

func TestCreateISO(t *testing.T) {
	t.Skip("Unskip this test to create a new ISO file")
	grubCfg := `set timeout=0
set gfxpayload=text
menuentry 'LinuxKit ISO Image' {
        linuxefi /kernel 464vn90e7rbj08xbwdjejmdf4it17c5zfzjyfhthbh19eij201hjgit021bmpdb9ctrc87x2ymc8e7icu4ffi15x1hah9iyaiz38ckyap8hwx2vt5rm44ixv4hau8iw718q5yd019um5dt2xpqqa2rjtdypzr5v1gun8un110hhwp8cex7pqrh2ivh0ynpm4zkkwc8wcn367zyethzy7q8hzudyeyzx3cgmxqbkh825gcak7kxzjbgjajwizryv7ec1xm2h0hh7pz29qmvtgfjj1vphpgq1zcbiiehv52wrjy9yq473d9t1rvryy6929nk435hfx55du3ih05kn5tju3vijreru1p6knc988d4gfdz28eragvryq5x8aibe5trxd0t6t7jwxkde34v6pj1khmp50k6qqj3nzgcfzabtgqkmeqhdedbvwf3byfdma4nkv3rcxugaj2d0ru30pa2fqadjqrtjnv8bu52xzxv7irbhyvygygxu1nt5z4fh9w1vwbdcmagep26d298zknykf2e88kumt59ab7nq79d8amnhhvbexgh48e8qc61vq2e9qkihzt1twk1ijfgw70nwizai15iqyted2dt9gfmf2gg7amzufre79hwqkddc1cd935ywacnkrnak6r7xzcz7zbmq3kt04u2hg1iuupid8rt4nyrju51e6uejb2ruu36g9aibmz3hnmvazptu8x5tyxk820g2cdpxjdij766bt2n3djur7v623a2v44juyfgz80ekgfb9hkibpxh3zgknw8a34t4jifhf116x15cei9hwch0fye3xyq0acuym8uhitu5evc4rag3ui0fny3qg4kju7zkfyy8hwh537urd5uixkzwu5bdvafz4jmv7imypj543xg5em8jk8cgk7c4504xdd5e4e71ihaumt6u5u2t1w7um92fepzae8p0vq93wdrd1756npu1pziiur1payc7kmdwyxg3hj5n4phxbc29x0tcddamjrwt260b0w text
        initrdefi /initrd.img
}
`
	if err := os.Remove("testdata/output.iso"); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	var diskSize int64 = 51200 // 50Kb
	mydisk, err := diskfs.Create("./testdata/output.iso", diskSize, diskfs.Raw, diskfs.SectorSizeDefault)
	if err != nil {
		t.Fatal(err)
	}
	defer mydisk.Close()

	// the following line is required for an ISO, which may have logical block sizes
	// only of 2048, 4096, 8192
	mydisk.LogicalBlocksize = 2048
	fspec := disk.FilesystemSpec{Partition: 0, FSType: filesystem.TypeISO9660, VolumeLabel: "label"}
	fs, err := mydisk.CreateFilesystem(fspec)
	if err != nil {
		t.Fatal(err)
	}
	if err := fs.Mkdir("EFI/BOOT"); err != nil {
		t.Fatal(err)
	}
	rw, err := fs.OpenFile("EFI/BOOT/grub.cfg", os.O_CREATE|os.O_RDWR)
	if err != nil {
		t.Fatal(err)
	}
	content := []byte(grubCfg)
	_, err = rw.Write(content)
	if err != nil {
		t.Fatal(err)
	}
	iso, ok := fs.(*iso9660.FileSystem)
	if !ok {
		t.Fatal(fmt.Errorf("not an iso9660 filesystem"))
	}
	err = iso.Finalize(iso9660.FinalizeOptions{})
	if err != nil {
		t.Fatal(err)
	}
}

func TestPatching(t *testing.T) {
	// create a small ISO file with the magic string
	// serve ISO with a http server
	// patch the ISO file
	// mount the ISO file and check if the magic string was patched

	wantGrubCfg := `set timeout=0
set gfxpayload=text
menuentry 'LinuxKit ISO Image' {
        linuxefi /kernel  facility=test console=ttyAMA0 console=ttyS0 console=tty0 console=tty1 console=ttyS1 syslog_host=127.0.0.1:514 grpc_authority=127.0.0.1:42113 tinkerbell_tls=false worker_id=de:ed:be:ef:fe:ed                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   text
        initrdefi /initrd.img
}
`
	// This expects that testdata/output.iso exists. Run the TestCreateISO test to create it.

	// serve it with a http server
	hs := httptest.NewServer(http.FileServer(http.Dir("./testdata")))
	defer hs.Close()

	// patch the ISO file
	u := hs.URL + "/output.iso"
	parsedURL, err := url.Parse(u)
	if err != nil {
		t.Fatal(err)
	}

	h := &Handler{
		Logger:             logr.Discard(),
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
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("got status code: %d, want status code: %d", res.StatusCode, http.StatusOK)
	}

	isoContents, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("patched.iso", isoContents, 0o644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove("patched.iso")

	dd, err := diskfs.Open("patched.iso", diskfs.WithOpenMode(diskfs.ReadOnly))
	if err != nil {
		t.Fatal(err)
	}
	defer dd.Close()
	fs, err := dd.GetFilesystem(0)
	if err != nil {
		t.Fatal(err)
	}
	ff, err := fs.OpenFile("EFI/BOOT/GRUB.CFG", os.O_RDONLY)
	if err != nil {
		t.Fatal(err)
	}
	defer ff.Close()
	grubCfgFile, err := io.ReadAll(ff)
	if err != nil {
		t.Fatal(err)
	}

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
