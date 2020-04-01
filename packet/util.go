package packet

import (
	"bytes"
	"net"
	"net/http"

	"github.com/pkg/errors"
)

var (
	ZeroMAC = MACAddr{}
	OnesMAC = MACAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
)

type Getter interface {
	Get() (*http.Request, error)
}

// MACAddr is an IEEE 802 MAC-48 hardware address.
type MACAddr [6]byte

func (m MACAddr) HardwareAddr() net.HardwareAddr {
	return net.HardwareAddr(m[:])
}

func (m MACAddr) String() string {
	return net.HardwareAddr(m[:]).String()
}

func (m *MACAddr) UnmarshalText(text []byte) error {
	const example = "00:00:00:00:00:00"
	if len(text) != len(example) {
		return errors.Errorf("expected a 48-bit hardware address, got %q", text)
	}
	*m = ZeroMAC

	mac, err := net.ParseMAC(string(text))
	if err != nil {
		return errors.Wrap(err, "parsing mac address")
	}
	copy(m[:], mac)
	return nil
}

func (m MACAddr) IsZero() bool {
	return bytes.Equal(m[:], ZeroMAC[:])
}

func (m MACAddr) IsOnes() bool {
	return bytes.Equal(m[:], OnesMAC[:])
}

// golangci-lint: unused
// withLSB sets the least significant bit to 1 if val == true or 0 if val == false.
//func (m MACAddr) withLSB(val bool) MACAddr {
//	if val {
//		m[5] |= 1 // set the last bit
//	} else {
//		m[5] &= ^byte(1) // clear the last bit
//	}
//	return m
//}
//
//type ref struct {
//	HRef string `json:"href"`
//}
//
//func (r *ref) Get() (*http.Request, error) {
//	if r.HRef == "" || r.HRef[0] == '#' {
//		return nil, errors.New("URL not available")
//	}
//	req, err := http.NewRequest("GET", r.HRef, nil)
//	return req, errors.Wrap(err, "fetching ref")
//}
