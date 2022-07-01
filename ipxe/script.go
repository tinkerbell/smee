package ipxe

import "fmt"

type Script struct {
	buf []byte
}

func NewScript() *Script {
	s := &Script{}
	s.Reset()

	return s
}

func (s *Script) Args(args ...string) {
	s.buf = s.buf[:len(s.buf)-1]

	for _, arg := range args {
		s.buf = append(append(s.buf, ' '), arg...)
	}

	s.buf = append(s.buf, '\n')
}

// AppendString takes a string and appends it to the current Script.
func (s *Script) AppendString(str string) {
	s.buf = append(s.buf, str...)
	s.buf = append(s.buf, '\n')
}

// PhoneHome takes a type and will post boots device connected to dhcp event.
func (s *Script) PhoneHome(typ string) {
	s.buf = append(s.buf, `
params
param body Device connected to DHCP system
param type `+typ+`
imgfetch ${tinkerbell}/phone-home##params
imgfree

`...)
}

// Chain - Chainload another iPXE script.
func (s *Script) Chain(uri string) {
	s.buf = append(append(s.buf, "chain --autofree "...), uri...)
	s.buf = append(s.buf, '\n')
}

func (s *Script) DHCP() {
	s.buf = append(s.buf, "dhcp\n"...)
}

func (s *Script) Boot() {
	s.buf = append(s.buf, "boot\n"...)
}

func (s *Script) Bytes() []byte {
	return s.buf
}

func (s *Script) Initrd(uri string, args ...string) {
	s.buf = append(append(s.buf, "initrd "...), uri...)

	for _, arg := range args {
		s.buf = append(append(s.buf, ' '), arg...)
	}

	s.buf = append(s.buf, '\n')
}

func (s *Script) Kernel(uri string, args ...string) {
	s.buf = append(append(s.buf, "kernel "...), uri...)

	for _, arg := range args {
		s.buf = append(append(s.buf, ' '), arg...)
	}

	s.buf = append(s.buf, '\n')
}

func (s *Script) Or(line string) {
	s.buf = append(s.buf[:len(s.buf)-1], " || "...)
	s.buf = append(s.buf, line...)
	s.buf = append(s.buf, '\n')
}

func (s *Script) Reset() {
	s.buf = append(s.buf[:0], "#!ipxe\n\n"...)
	s.Echo("Tinkerbell Boots iPXE")
}

// Echo outputs a string to console.
func (s *Script) Echo(message string) {
	s.buf = append(append(s.buf, "echo "...), message...)
	s.buf = append(s.buf, '\n')
}

func (s *Script) Set(name, value string) {
	s.buf = append(append(s.buf, "set "...), name...)
	s.buf = append(append(s.buf, ' '), value...)
	s.buf = append(s.buf, '\n')
}

func (s *Script) Shell() {
	s.buf = append(s.buf, "shell\n"...)
}

func (s *Script) Sleep(value int) {
	s.buf = append(s.buf, fmt.Sprintf("sleep %d\n", value)...)
}
