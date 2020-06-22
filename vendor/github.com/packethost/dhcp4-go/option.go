package dhcp4

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"reflect"
	"sort"
	"strconv"
	"time"
)

// MessageType is the type for the various DHCP messages defined in RFC2132.
type MessageType byte

const (
	MessageTypeDiscover = MessageType(1)
	MessageTypeOffer    = MessageType(2)
	MessageTypeRequest  = MessageType(3)
	MessageTypeDecline  = MessageType(4)
	MessageTypeAck      = MessageType(5)
	MessageTypeNak      = MessageType(6)
	MessageTypeRelease  = MessageType(7)
	MessageTypeInform   = MessageType(8)
)

var messageTypeStrings = map[MessageType]string{
	MessageTypeDiscover: "DHCPDISCOVER",
	MessageTypeOffer:    "DHCPOFFER",
	MessageTypeRequest:  "DHCPREQUEST",
	MessageTypeDecline:  "DHCPDECLINE",
	MessageTypeAck:      "DHCPACK",
	MessageTypeNak:      "DHCPNAK",
	MessageTypeRelease:  "DHCPRELEASE",
	MessageTypeInform:   "DHCPINFORM",
}

func (t MessageType) String() string {
	if s, ok := messageTypeStrings[t]; ok {
		return s
	}
	return fmt.Sprintf("DHCP(%d)", t)
}

// OptionGetter defines a bag of functions that can be used to get options.
type OptionGetter interface {
	GetSortedOptions() []Option
	GetOption(Option) ([]byte, bool)
	GetMessageType() MessageType
	GetUint8(Option) (uint8, bool)
	GetUint16(Option) (uint16, bool)
	GetUint32(Option) (uint32, bool)
	GetString(Option) (string, bool)
	GetIP(Option) (net.IP, bool)
	GetDuration(Option) (time.Duration, bool)
}

// OptionSetter defines a bag of functions that can be used to set options.
type OptionSetter interface {
	SetOption(Option, []byte)
	SetMessageType(MessageType)
	SetUint8(Option, uint8)
	SetUint16(Option, uint16)
	SetUint32(Option, uint32)
	SetString(Option, string)
	SetIP(Option, net.IP)
	SetDuration(Option, time.Duration)
}

// Option is the type for DHCP option tags.
type Option byte

// OptionMap maps DHCP option tags to their values.
type OptionMap map[Option][]byte

// optionSlice defines a sortable array of options.
type optionSlice []Option

func (o optionSlice) Len() int {
	return len(o)
}

func (o optionSlice) Less(i, j int) bool {
	return o[i] < o[j]
}

func (o optionSlice) Swap(i, j int) {
	x := o[i]
	o[i] = o[j]
	o[j] = x
}

// GetSortedOptions gets all the options (keys) in sorted order.
func (om OptionMap) GetSortedOptions() []Option {
	ks := make(optionSlice, 0, len(om))
	for k := range om {
		ks = append(ks, k)
	}
	sort.Sort(ks)
	return []Option(ks)
}

// GetOption gets the []byte value of an option.
func (om OptionMap) GetOption(o Option) ([]byte, bool) {
	v, ok := om[o]
	return v, ok
}

// SetOption sets the []byte value of an option.
func (om OptionMap) SetOption(o Option, v []byte) {
	om[o] = v
}

// GetMessageType gets the message type from the DHCPMsgType option field.
func (om OptionMap) GetMessageType() MessageType {
	v, ok := om.GetOption(OptionDHCPMsgType)
	if !ok || len(v) != 1 {
		return MessageType(0)
	}

	return MessageType(v[0])
}

// SetMessageType sets the message type in the DHCPMsgType option field.
func (om OptionMap) SetMessageType(m MessageType) {
	om.SetOption(OptionDHCPMsgType, []byte{byte(m)})
}

// GetUint8 gets the 8 bit unsigned integer value of an option.
func (om OptionMap) GetUint8(o Option) (uint8, bool) {
	if v, ok := om.GetOption(o); ok && len(v) == 1 {
		return uint8(v[0]), true
	}

	return uint8(0), false
}

// SetUint8 sets the 8 bit unsigned integer value of an option.
func (om OptionMap) SetUint8(o Option, v uint8) {
	b := make([]byte, 1)
	b[0] = uint8(v)
	om.SetOption(o, b)
}

// GetUint16 gets the 16 bit unsigned integer value of an option.
func (om OptionMap) GetUint16(o Option) (uint16, bool) {
	if v, ok := om.GetOption(o); ok && len(v) == 2 {
		return binary.BigEndian.Uint16(v), true
	}

	return uint16(0), false
}

// SetUint16 sets the 16 bit unsigned integer value of an option.
func (om OptionMap) SetUint16(o Option, v uint16) {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, v)
	om.SetOption(o, b)
}

// GetUint32 gets the 32 bit unsigned integer value of an option.
func (om OptionMap) GetUint32(o Option) (uint32, bool) {
	if v, ok := om.GetOption(o); ok && len(v) == 4 {
		return binary.BigEndian.Uint32(v), true
	}

	return uint32(0), false
}

// SetUint32 sets the 32 bit unsigned integer value of an option.
func (om OptionMap) SetUint32(o Option, v uint32) {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, v)
	om.SetOption(o, b)
}

// GetString gets the string value of an option.
func (om OptionMap) GetString(o Option) (string, bool) {
	if v, ok := om.GetOption(o); ok {
		return string(v), true
	}

	return "", false
}

// SetString sets the string value of an option.
func (om OptionMap) SetString(o Option, v string) {
	om.SetOption(o, []byte(v))
}

// GetIP gets the IP value of an option.
func (om OptionMap) GetIP(o Option) (net.IP, bool) {
	if v, ok := om.GetOption(o); ok && len(v) == 4 {
		return net.IPv4(v[0], v[1], v[2], v[3]), true
	}

	return nil, false
}

// GetIP sets the IP value of an option.
func (om OptionMap) SetIP(o Option, v net.IP) {
	om.SetOption(o, []byte(v.To4()))
}

// GetDuration gets the duration value of an option, stored as a 32 bit unsigned integer.
func (om OptionMap) GetDuration(o Option) (time.Duration, bool) {
	if v, ok := om.GetUint32(o); ok {
		return time.Duration(v) * time.Second, true
	}

	return time.Duration(0), false
}

// SetDuration sets the duration value of an option, stored as a 32 bit unsigned integer.
func (om OptionMap) SetDuration(o Option, v time.Duration) {
	om.SetUint32(o, uint32(v.Seconds()))
}

type OptionMapDeserializeOptions struct {
	IgnoreMissingEndTag bool
}

// Deserialize reads options from the []byte into the option map.
func (om OptionMap) Deserialize(x []byte, opts *OptionMapDeserializeOptions) error {
	for {
		if len(x) == 0 {
			if opts != nil && opts.IgnoreMissingEndTag {
				return nil
			}

			return ErrShortPacket
		}

		tag := Option(x[0])
		x = x[1:]
		if tag == OptionEnd {
			break
		}

		// Padding tag
		if tag == OptionPad {
			continue
		}

		// Read length octet
		if len(x) == 0 {
			return ErrShortPacket
		}

		length := int(x[0])
		x = x[1:]
		if len(x) < length {
			return ErrShortPacket
		}

		_, ok := om[tag]
		if ok {
			// We've got a bad client here; duplicate options are not allowed.
			// Let it slide instead of throwing a fit, for the sake of robustness.
		}

		// Capture option and move to the next one
		om[tag] = x[0:length]
		x = x[length:]
	}

	return nil
}

// Serialize writes the contents of the option map to a byte slice.
func (om OptionMap) Serialize() []byte {
	b := bytes.Buffer{}

	for k, v := range om {
		if len(v) > 255 {
			continue
		}

		if err := b.WriteByte(byte(k)); err != nil {
			panic(err)
		}

		if err := b.WriteByte(byte(len(v))); err != nil {
			panic(err)
		}

		if _, err := b.Write(v); err != nil {
			panic(err)
		}
	}

	if err := b.WriteByte(byte(OptionEnd)); err != nil {
		panic(err)
	}

	return b.Bytes()
}

func (om OptionMap) decodeValue(code int, dv reflect.Value) {
	var rv reflect.Value

	dt := dv.Type()
	n := 0

	// Indirect the pointer type as far as needed
	for dt.Kind() == reflect.Ptr {
		dt = dt.Elem()
		n++
	}

	switch dt.Kind() {
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Int8, reflect.Int16, reflect.Int32:
		if v, ok := om.GetOption(Option(code)); ok && len(v) > 0 {
			rv = reflect.New(dt)
			binary.Read(bytes.NewReader(v), binary.BigEndian, rv.Interface())

			// Dereference pointer so that the underlying value can be assigned to
			// the destination reflect.Value if it is not a pointer.
			rv = rv.Elem()
		}
	case reflect.String:
		if v, ok := om.GetString(Option(code)); ok {
			rv = reflect.ValueOf(v)
		}
	case reflect.Bool:
		if v, ok := om.GetUint8(Option(code)); ok {
			rv = reflect.ValueOf(v > 0)
		}
	}

	// Abort if the result value wasn't set
	if !rv.IsValid() {
		return
	}

	// Get the value to the pointer depth to the level we can set it at
	for ; n > 0; n-- {
		if rv.CanAddr() {
			rv = rv.Addr()
		} else {
			// Make addressable
			x := reflect.New(rv.Type())
			x.Elem().Set(rv)
			rv = x
		}
	}

	dv.Set(rv)
}

func (om OptionMap) Decode(dst interface{}) {
	sv := reflect.ValueOf(dst)
	st := sv.Type()

	// Expect a pointer
	if st.Kind() != reflect.Ptr {
		panic("dhcp4: expected a *struct")
	}

	sv = sv.Elem()
	st = sv.Type()

	// Expect a struct
	if st.Kind() != reflect.Struct {
		panic("dhcp4: expected a *struct")
	}

	// Walk the fields of this struct
	n := sv.NumField()
	for i := 0; i < n; i++ {
		fv := sv.Field(i)
		ft := st.Field(i)

		s := ft.Tag.Get("code")
		if s == "" {
			continue
		}

		code, err := strconv.Atoi(s)
		if err != nil {
			continue
		}

		om.decodeValue(code, fv)
	}
}

func encodeInt(v reflect.Value) []byte {
	var buffer bytes.Buffer

	err := binary.Write(&buffer, binary.BigEndian, v.Interface())
	if err != nil {
		panic(err)
	}

	return buffer.Bytes()
}

func (om OptionMap) encodeValue(code int, v reflect.Value) {
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			// Bail if there is nothing to see here...
			return
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Uint8, reflect.Uint16, reflect.Uint32:
		if v.Uint() != 0 {
			om.SetOption(Option(code), encodeInt(v))
		}
	case reflect.Int8, reflect.Int16, reflect.Int32:
		if v.Int() != 0 {
			om.SetOption(Option(code), encodeInt(v))
		}
	case reflect.String:
		if v.Len() > 0 {
			om.SetOption(Option(code), []byte(v.Interface().(string)))
		}
	case reflect.Bool:
		if v.Bool() {
			om.SetOption(Option(code), []byte{0x1})
		}
	}
}

func (om OptionMap) Encode(dst interface{}) {
	sv := reflect.ValueOf(dst)
	st := sv.Type()

	// Dereference the pointer as deep as possible
	for sv.Kind() == reflect.Ptr {
		sv = sv.Elem()
		st = sv.Type()
	}

	// Expect a struct
	if st.Kind() != reflect.Struct {
		panic("dhcp4: expected a *struct")
	}

	// Walk the fields of this struct
	n := sv.NumField()
	for i := 0; i < n; i++ {
		fv := sv.Field(i)
		ft := st.Field(i)

		s := ft.Tag.Get("code")
		if s == "" {
			continue
		}

		code, err := strconv.Atoi(s)
		if err != nil {
			continue
		}

		om.encodeValue(code, fv)
	}
}

// From RFC2132: DHCP Options and BOOTP Vendor Extensions
const (
	// RFC2132 Section 3: RFC 1497 Vendor Extensions
	OptionPad           = Option(0)
	OptionEnd           = Option(255)
	OptionSubnetMask    = Option(1)
	OptionTimeOffset    = Option(2)
	OptionRouter        = Option(3)
	OptionTimeServer    = Option(4)
	OptionNameServer    = Option(5)
	OptionDomainServer  = Option(6)
	OptionLogServer     = Option(7)
	OptionQuotesServer  = Option(8)
	OptionLPRServer     = Option(9)
	OptionImpressServer = Option(10)
	OptionRLPServer     = Option(11)
	OptionHostname      = Option(12)
	OptionBootFileSize  = Option(13)
	OptionMeritDumpFile = Option(14)
	OptionDomainName    = Option(15)
	OptionSwapServer    = Option(16)
	OptionRootPath      = Option(17)
	OptionExtensionFile = Option(18)

	// RFC2132 Section 4: IP Layer Parameters per Host
	OptionForwardOnOff  = Option(19)
	OptionSrcRteOnOff   = Option(20)
	OptionPolicyFilter  = Option(21)
	OptionMaxDGAssembly = Option(22)
	OptionDefaultIPTTL  = Option(23)
	OptionMTUTimeout    = Option(24)
	OptionMTUPlateau    = Option(25)

	// RFC2132 Section 5: IP Layer Parameters per Interface
	OptionMTUInterface     = Option(26)
	OptionMTUSubnet        = Option(27)
	OptionBroadcastAddress = Option(28)
	OptionMaskDiscovery    = Option(29)
	OptionMaskSupplier     = Option(30)
	OptionRouterDiscovery  = Option(31)
	OptionRouterRequest    = Option(32)
	OptionStaticRoute      = Option(33)

	// RFC2132 Section 6: Link Layer Parameters per Interface
	OptionTrailers   = Option(34)
	OptionARPTimeout = Option(35)
	OptionEthernet   = Option(36)

	// RFC2132 Section 7: TCP Parameters
	OptionDefaultTCPTTL = Option(37)
	OptionKeepaliveTime = Option(38)
	OptionKeepaliveData = Option(39)

	// RFC2132 Section 8: Application and Service Parameters
	OptionNISDomain        = Option(40)
	OptionNISServers       = Option(41)
	OptionNTPServers       = Option(42)
	OptionVendorSpecific   = Option(43)
	OptionNETBIOSNameSrv   = Option(44)
	OptionNETBIOSDistSrv   = Option(45)
	OptionNETBIOSNodeType  = Option(46)
	OptionNETBIOSScope     = Option(47)
	OptionXWindowFont      = Option(48)
	OptionXWindowManager   = Option(49)
	OptionNISDomainName    = Option(64)
	OptionNISServerAddr    = Option(65)
	OptionHomeAgentAddrs   = Option(68)
	OptionSMTPServer       = Option(69)
	OptionPOP3Server       = Option(70)
	OptionNNTPServer       = Option(71)
	OptionWWWServer        = Option(72)
	OptionFingerServer     = Option(73)
	OptionIRCServer        = Option(74)
	OptionStreetTalkServer = Option(75)
	OptionSTDAServer       = Option(76)

	// RFC2132 Section 9: DHCP Extensions
	OptionAddressRequest = Option(50)
	OptionAddressTime    = Option(51)
	OptionOverload       = Option(52)
	OptionServerName     = Option(66)
	OptionBootfileName   = Option(67)
	OptionDHCPMsgType    = Option(53)
	OptionDHCPServerID   = Option(54)
	OptionParameterList  = Option(55)
	OptionDHCPMessage    = Option(56)
	OptionDHCPMaxMsgSize = Option(57)
	OptionRenewalTime    = Option(58)
	OptionRebindingTime  = Option(59)
	OptionClassID        = Option(60)
	OptionClientID       = Option(61)
)

// From RFC2241: DHCP Options for Novell Directory Services
const (
	OptionNDSServers  = Option(85)
	OptionNDSTreeName = Option(86)
	OptionNDSContext  = Option(87)
)

// From RFC2242: NetWare/IP Domain Name and Information
const (
	OptionNetWareIPDomain = Option(62)
	OptionNetWareIPOption = Option(63)
)

// From RFC2485: DHCP Option for The Open Group\x27s User Authentication Protocol
const (
	OptionUserAuth = Option(98)
)

// From RFC2563: DHCP Option to Disable Stateless Auto-Configuration in IPv4 Clients
const (
	OptionAutoConfig = Option(116)
)

// From RFC2610: DHCP Options for Service Location Protocol
const (
	OptionDirectoryAgent = Option(78)
	OptionServiceScope   = Option(79)
)

// From RFC2937: The Name Service Search Option for DHCP
const (
	OptionNameServiceSearch = Option(117)
)

// From RFC3004: The User Class Option for DHCP
const (
	OptionUserClass = Option(77)
)

// From RFC3011: The IPv4 Subnet Selection Option for DHCP
const (
	OptionSubnetSelectionOption = Option(118)
)

// From RFC3046: DHCP Relay Agent Information Option
const (
	OptionRelayAgentInformation = Option(82)
)

// From RFC3118: Authentication for DHCP Messages
const (
	OptionAuthentication = Option(90)
)

// From RFC3361: Dynamic Host Configuration Protocol (DHCP-for-IPv4) Option for Session Initiation Protocol (SIP) Servers
const (
	OptionSIPServersDHCPOption = Option(120)
)

// From RFC3397: Dynamic Host Configuration Protocol (DHCP) Domain Search Option
const (
	OptionDomainSearch = Option(119)
)

// From RFC3442: The Classless Static Route Option for Dynamic Host Configuration Protocol (DHCP) version 4
const (
	OptionClasslessStaticRouteOption = Option(121)
)

// From RFC3495: Dynamic Host Configuration Protocol (DHCP) Option for CableLabs Client Configuration
const (
	OptionCCC = Option(122)
)

// From RFC3679: Unused Dynamic Host Configuration Protocol (DHCP) Option Codes
const (
	OptionLDAP           = Option(95)
	OptionNetinfoAddress = Option(112)
	OptionNetinfoTag     = Option(113)
	OptionURL            = Option(114)
)

// From RFC3925: Vendor-Identifying Vendor Options for Dynamic Host Configuration Protocol version 4 (DHCPv4)
const (
	OptionVIVendorClass               = Option(124)
	OptionVIVendorSpecificInformation = Option(125)
)

// From RFC4039: Rapid Commit Option for the Dynamic Host Configuration Protocol version 4 (DHCPv4)
const (
	OptionRapidCommit = Option(80)
)

// From RFC4174: The IPv4 Dynamic Host Configuration Protocol (DHCP) Option for the Internet Storage Name Service
const (
	OptioniSNS = Option(83)
)

// From RFC4280: Dynamic Host Configuration Protocol (DHCP) Options for Broadcast and Multicast Control Servers
const (
	OptionBCMCSControllerDomainNameList    = Option(88)
	OptionBCMCSControllerIPv4AddressOption = Option(89)
)

// From RFC4388: Dynamic Host Configuration Protocol (DHCP) Leasequery
const (
	OptionClientLastTransactionTimeOption = Option(91)
	OptionAssociatedIPOption              = Option(92)
)

// From RFC4578: Dynamic Host Configuration Protocol (DHCP) Options for the Intel Preboot eXecution Environment (PXE)
const (
	OptionClientSystem    = Option(93)
	OptionClientNDI       = Option(94)
	OptionUUIDGUID        = Option(97)
	OptionPXEUndefined128 = Option(128)
	OptionPXEUndefined129 = Option(129)
	OptionPXEUndefined130 = Option(130)
	OptionPXEUndefined131 = Option(131)
	OptionPXEUndefined132 = Option(132)
	OptionPXEUndefined133 = Option(133)
	OptionPXEUndefined134 = Option(134)
	OptionPXEUndefined135 = Option(135)
)

// From RFC4702: The Dynamic Host Configuration Protocol (DHCP) Client Fully Qualified Domain Name (FQDN) Option
const (
	OptionClientFQDN = Option(81)
)

// From RFC4776: Dynamic Host Configuration Protocol (DHCPv4 and DHCPv6) Option for Civic Addresses Configuration Information
const (
	OptionGeoConfCivic = Option(99)
)

// From RFC4833: Timezone Options for DHCP
const (
	OptionPCode = Option(100)
	OptionTCode = Option(101)
)

// From RFC6225: Dynamic Host Configuration Protocol Options for Coordinate-Based Location Configuration Information
const (
	OptionGeoConfOption = Option(123)
	OptionGeoLoc        = Option(144)
)
