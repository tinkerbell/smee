// Package file watches a file for changes and updates the in memory DHCP data.
package file

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ccoveille/go-safecast"
	"github.com/fsnotify/fsnotify"
	"github.com/ghodss/yaml"
	"github.com/go-logr/logr"
	"github.com/tinkerbell/smee/internal/dhcp/data"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

const tracerName = "github.com/tinkerbell/smee/dhcp"

// Errors used by the file watcher.
var (
	// errFileFormat is returned when the file is not in the correct format, e.g. not valid YAML.
	errFileFormat     = fmt.Errorf("invalid file format")
	errRecordNotFound = fmt.Errorf("record not found")
	errParseIP        = fmt.Errorf("failed to parse IP from File")
	errParseSubnet    = fmt.Errorf("failed to parse subnet mask from File")
	errParseURL       = fmt.Errorf("failed to parse URL")
)

// netboot is the structure for the data expected in a file.
type netboot struct {
	AllowPXE      bool   `yaml:"allowPxe"`      // If true, the client will be provided netboot options in the DHCP offer/ack.
	IPXEScriptURL string `yaml:"ipxeScriptUrl"` // Overrides default value of that is passed into DHCP on startup.
	IPXEScript    string `yaml:"ipxeScript"`    // Overrides a default value that is passed into DHCP on startup.
	Console       string `yaml:"console"`
	Facility      string `yaml:"facility"`
}

// dhcp is the structure for the data expected in a file.
type dhcp struct {
	MACAddress       net.HardwareAddr // The MAC address of the client.
	IPAddress        string           `yaml:"ipAddress"`        // yiaddr DHCP header.
	SubnetMask       string           `yaml:"subnetMask"`       // DHCP option 1.
	DefaultGateway   string           `yaml:"defaultGateway"`   // DHCP option 3.
	NameServers      []string         `yaml:"nameServers"`      // DHCP option 6.
	Hostname         string           `yaml:"hostname"`         // DHCP option 12.
	DomainName       string           `yaml:"domainName"`       // DHCP option 15.
	BroadcastAddress string           `yaml:"broadcastAddress"` // DHCP option 28.
	NTPServers       []string         `yaml:"ntpServers"`       // DHCP option 42.
	VLANID           string           `yaml:"vlanID"`           // DHCP option 43.116.
	LeaseTime        int              `yaml:"leaseTime"`        // DHCP option 51.
	Arch             string           `yaml:"arch"`             // DHCP option 93.
	DomainSearch     []string         `yaml:"domainSearch"`     // DHCP option 119.
	Netboot          netboot          `yaml:"netboot"`
}

// Watcher represents the backend for watching a file for changes and updating the in memory DHCP data.
type Watcher struct {
	fileMu sync.RWMutex // protects FilePath for reads

	// FilePath is the path to the file to watch.
	FilePath string

	// Log is the logger to be used in the File backend.
	Log     logr.Logger
	dataMu  sync.RWMutex // protects data
	data    []byte       // data from file
	watcher *fsnotify.Watcher
}

// NewWatcher creates a new file watcher.
func NewWatcher(l logr.Logger, f string) (*Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	if err := watcher.Add(f); err != nil {
		return nil, err
	}

	w := &Watcher{
		FilePath: f,
		watcher:  watcher,
		Log:      l,
	}

	w.fileMu.RLock()
	w.data, err = os.ReadFile(filepath.Clean(f))
	w.fileMu.RUnlock()
	if err != nil {
		return nil, err
	}

	return w, nil
}

// GetByMac is the implementation of the Backend interface.
// It reads a given file from the in memory data (w.data).
func (w *Watcher) GetByMac(ctx context.Context, mac net.HardwareAddr) (*data.DHCP, *data.Netboot, error) {
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "backend.file.GetByMac")
	defer span.End()

	// get data from file, translate it, then pass it into setDHCPOpts and setNetworkBootOpts
	w.dataMu.RLock()
	d := w.data
	w.dataMu.RUnlock()
	r := make(map[string]dhcp)
	if err := yaml.Unmarshal(d, &r); err != nil {
		err := fmt.Errorf("%w: %w", err, errFileFormat)
		w.Log.Error(err, "failed to unmarshal file data")
		span.SetStatus(codes.Error, err.Error())

		return nil, nil, err
	}
	for k, v := range r {
		if strings.EqualFold(k, mac.String()) {
			// found a record for this mac address
			v.MACAddress = mac
			d, n, err := w.translate(v)
			if err != nil {
				span.SetStatus(codes.Error, err.Error())

				return nil, nil, err
			}
			span.SetAttributes(d.EncodeToAttributes()...)
			span.SetAttributes(n.EncodeToAttributes()...)
			span.SetStatus(codes.Ok, "")

			return d, n, nil
		}
	}

	err := fmt.Errorf("%w: %s", errRecordNotFound, mac.String())
	span.SetStatus(codes.Error, err.Error())

	return nil, nil, err
}

// GetByIP is the implementation of the Backend interface.
// It reads a given file from the in memory data (w.data).
func (w *Watcher) GetByIP(ctx context.Context, ip net.IP) (*data.DHCP, *data.Netboot, error) {
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "backend.file.GetByIP")
	defer span.End()

	// get data from file, translate it, then pass it into setDHCPOpts and setNetworkBootOpts
	w.dataMu.RLock()
	d := w.data
	w.dataMu.RUnlock()
	r := make(map[string]dhcp)
	if err := yaml.Unmarshal(d, &r); err != nil {
		err := fmt.Errorf("%w: %w", err, errFileFormat)
		w.Log.Error(err, "failed to unmarshal file data")
		span.SetStatus(codes.Error, err.Error())

		return nil, nil, err
	}
	for k, v := range r {
		if v.IPAddress == ip.String() {
			// found a record for this ip address
			v.IPAddress = ip.String()
			mac, err := net.ParseMAC(k)
			if err != nil {
				err := fmt.Errorf("%w: %w", err, errFileFormat)
				w.Log.Error(err, "failed to parse mac address")
				span.SetStatus(codes.Error, err.Error())

				return nil, nil, err
			}
			v.MACAddress = mac
			d, n, err := w.translate(v)
			if err != nil {
				span.SetStatus(codes.Error, err.Error())

				return nil, nil, err
			}
			span.SetAttributes(d.EncodeToAttributes()...)
			span.SetAttributes(n.EncodeToAttributes()...)
			span.SetStatus(codes.Ok, "")

			return d, n, nil
		}
	}

	err := fmt.Errorf("%w: %s", errRecordNotFound, ip.String())
	span.SetStatus(codes.Error, err.Error())

	return nil, nil, err
}

// Start starts watching a file for changes and updates the in memory data (w.data) on changes.
// Start is a blocking method. Use a context cancellation to exit.
func (w *Watcher) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			w.Log.Info("stopping watcher")
			return
		case event, ok := <-w.watcher.Events:
			if !ok {
				continue
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				w.Log.Info("file changed, updating cache")
				w.fileMu.RLock()
				d, err := os.ReadFile(w.FilePath)
				w.fileMu.RUnlock()
				if err != nil {
					w.Log.Error(err, "failed to read file", "file", w.FilePath)
					break
				}
				w.dataMu.Lock()
				w.data = d
				w.dataMu.Unlock()
			}
		case err, ok := <-w.watcher.Errors:
			if !ok {
				continue
			}
			w.Log.Info("error watching file", "err", err)
		}
	}
}

// translate converts the data from the file into a data.DHCP and data.Netboot structs.
func (w *Watcher) translate(r dhcp) (*data.DHCP, *data.Netboot, error) {
	d := new(data.DHCP)
	n := new(data.Netboot)

	d.MACAddress = r.MACAddress
	// ip address, required
	ip, err := netip.ParseAddr(r.IPAddress)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %w", err, errParseIP)
	}
	d.IPAddress = ip

	// subnet mask, required
	sm := net.ParseIP(r.SubnetMask)
	if sm == nil {
		return nil, nil, errParseSubnet
	}
	d.SubnetMask = net.IPMask(sm.To4())

	// default gateway, optional
	if dg, err := netip.ParseAddr(r.DefaultGateway); err != nil {
		w.Log.Info("failed to parse default gateway", "defaultGateway", r.DefaultGateway, "err", err)
	} else {
		d.DefaultGateway = dg
	}

	// name servers, optional
	for _, s := range r.NameServers {
		ip := net.ParseIP(s)
		if ip == nil {
			w.Log.Info("failed to parse name server", "nameServer", s)
			break
		}
		d.NameServers = append(d.NameServers, ip)
	}

	// hostname, optional
	d.Hostname = r.Hostname

	// domain name, optional
	d.DomainName = r.DomainName

	// broadcast address, optional
	if ba, err := netip.ParseAddr(r.BroadcastAddress); err != nil {
		w.Log.Info("failed to parse broadcast address", "broadcastAddress", r.BroadcastAddress, "err", err)
	} else {
		d.BroadcastAddress = ba
	}

	// ntp servers, optional
	for _, s := range r.NTPServers {
		ip := net.ParseIP(s)
		if ip == nil {
			w.Log.Info("failed to parse ntp server", "ntpServer", s)
			break
		}
		d.NTPServers = append(d.NTPServers, ip)
	}

	// vlanid
	d.VLANID = r.VLANID

	// lease time
	// Default to one week
	d.LeaseTime = 604800
	if v, err := safecast.ToUint32(r.LeaseTime); err == nil {
		d.LeaseTime = v
	}

	// arch
	d.Arch = r.Arch

	// domain search
	d.DomainSearch = r.DomainSearch

	// allow machine to netboot
	n.AllowNetboot = r.Netboot.AllowPXE

	// ipxe script url is optional but if provided, it must be a valid url
	if r.Netboot.IPXEScriptURL != "" {
		u, err := url.Parse(r.Netboot.IPXEScriptURL)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: %w", err, errParseURL)
		}
		n.IPXEScriptURL = u
	}

	// ipxe script
	if r.Netboot.IPXEScript != "" {
		n.IPXEScript = r.Netboot.IPXEScript
	}

	// console
	if r.Netboot.Console != "" {
		n.Console = r.Netboot.Console
	}

	// facility
	if r.Netboot.Facility != "" {
		n.Facility = r.Netboot.Facility
	}

	return d, n, nil
}
