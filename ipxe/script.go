package ipxe

import "fmt"

type Script struct {
	buf []byte
}

func NewScript() *Script {
	return new(Script).Reset()
}

func (s *Script) Args(args ...string) *Script {
	s.buf = s.buf[:len(s.buf)-1]

	for _, arg := range args {
		s.buf = append(append(s.buf, ' '), arg...)
	}

	s.buf = append(s.buf, '\n')

	return s
}

// AppendString takes a string and appends it to the current Script
func (s *Script) AppendString(s_script string) *Script {
	s.buf = append(s.buf, s_script...)
	s.buf = append(s.buf, '\n')

	return s
}

// PhoneHome takes a type and will post boots device connected to dhcp event
func (s *Script) PhoneHome(typ string) *Script {
	s.buf = append(s.buf, `
params
param body Device connected to DHCP system
param type `+typ+`
imgfetch ${tinkerbell}/phone-home##params
imgfree

`...)

	return s
}

// Chain - Chainload another iPXE script
func (s *Script) Chain(uri string) *Script {
	s.buf = append(append(s.buf, "chain --autofree "...), uri...)
	s.buf = append(s.buf, '\n')

	return s
}

func (s *Script) DHCP() *Script {
	s.buf = append(s.buf, "dhcp\n"...)

	return s
}

func (s *Script) Boot() *Script {
	s.buf = append(s.buf, "boot\n"...)

	return s
}

func (s *Script) Bytes() []byte {
	return s.buf
}

func (s *Script) Initrd(uri string, args ...string) *Script {
	s.buf = append(append(s.buf, "initrd "...), uri...)

	for _, arg := range args {
		s.buf = append(append(s.buf, ' '), arg...)
	}

	s.buf = append(s.buf, '\n')

	return s
}

func (s *Script) Kernel(uri string, args ...string) *Script {
	s.buf = append(append(s.buf, "kernel "...), uri...)

	for _, arg := range args {
		s.buf = append(append(s.buf, ' '), arg...)
	}

	s.buf = append(s.buf, '\n')

	return s
}

func (s *Script) Or(line string) *Script {
	s.buf = append(s.buf[:len(s.buf)-1], " || "...)
	s.buf = append(s.buf, line...)
	s.buf = append(s.buf, '\n')

	return s
}

func (s *Script) Reset() *Script {
	s.buf = append(s.buf[:0], "#!ipxe\n\n"...)

	return s
}

// Echo outputs a string to console
func (s *Script) Echo(message string) *Script {
	s.buf = append(append(s.buf, "echo "...), message...)
	s.buf = append(s.buf, '\n')

	return s
}

func (s *Script) Set(name, value string) *Script {
	s.buf = append(append(s.buf, "set "...), name...)
	s.buf = append(append(s.buf, ' '), value...)
	s.buf = append(s.buf, '\n')

	return s
}

func (s *Script) Shell() *Script {
	s.buf = append(s.buf, "shell\n"...)

	return s
}

func (s *Script) Sleep(value int) *Script {
	s.buf = append(s.buf, fmt.Sprintf("sleep %d\n", value)...)

	return s
}
