/*
Copyright (c) 2015 VMware, Inc. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tftp

// Serialization and deserialization of TFTP packets.

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"strings"
)

type opcode uint16
type mode string

const (
	opcodeRRQ   = opcode(1)
	opcodeWRQ   = opcode(2)
	opcodeDATA  = opcode(3)
	opcodeACK   = opcode(4)
	opcodeERROR = opcode(5)
	opcodeOACK  = opcode(6)

	modeNETASCII = mode("netascii")
	modeOCTET    = mode("octet")
)

type tftpError struct {
	Code    uint16
	Message string
}

var (
	// Error codes as defined by the TFTP spec.
	tftpErrNotDefined        = tftpError{0, "Not defined, see error message (if any)."}
	tftpErrNotFound          = tftpError{1, "File not found."}
	tftpErrAccessViolation   = tftpError{2, "Access violation."}
	tftpErrDiskFull          = tftpError{3, "Disk full or allocation exceeded."}
	tftpErrIllegalOperation  = tftpError{4, "Illegal TFTP operation."}
	tftpErrUnknownTransferID = tftpError{5, "Unknown transfer ID."}
	tftpErrFileAlreadyExists = tftpError{6, "File already exists."}
	tftpErrNoSuchUser        = tftpError{7, "No such user."}
	tftpErrOptionNegotiation = tftpError{8, "Option negotiation error."}
)

var (
	errOpcode = errors.New("invalid opcode")
	errMode   = errors.New("invalid mode")
)

// Packet is the interface that every TFTP packet implements.
type packet interface {
	Read(b *bytes.Buffer) error
	Write(b *bytes.Buffer) error
}

// readChunk reads until the first NUL byte in the input, returning
// a string containing the data up to but excluding the delimeter.
// Other semantics are identical to bytes.Buffer#ReadBytes.
func readChunk(b *bytes.Buffer) (string, error) {
	tail := b.Bytes()
	i := bytes.IndexByte(tail, 0)
	if i < 0 {
		return "", io.ErrUnexpectedEOF
	}

	// Skip over chunk
	c := b.Next(i + 1)
	return string(c[:i]), nil
}

// writeChunk writes the specified string, followed by a NUL byte.
func writeChunk(b *bytes.Buffer, line string) error {
	_, err := b.WriteString(line)
	if err != nil {
		return err
	}

	return b.WriteByte(0)
}

func readOptions(b *bytes.Buffer) (map[string]string, error) {
	o := make(map[string]string)
	for b.Len() > 0 {
		k, err := readChunk(b)
		if err != nil {
			return nil, err
		}

		v, err := readChunk(b)
		if err != nil {
			return nil, err
		}

		o[strings.ToLower(k)] = strings.ToLower(v)
	}

	return o, nil
}

func writeOptions(b *bytes.Buffer, o map[string]string) error {
	var err error
	for k, v := range o {
		if err = writeChunk(b, k); err != nil {
			return err
		}
		if err = writeChunk(b, v); err != nil {
			return err
		}
	}
	return nil
}

type packetXRQ struct {
	filename string
	mode     mode
	options  map[string]string
}

func (p *packetXRQ) Read(b *bytes.Buffer) error {
	var err error

	p.filename, err = readChunk(b)
	if err != nil {
		return err
	}

	m, err := readChunk(b)
	if err != nil {
		return err
	}

	// Check if mode is valid
	p.mode = mode(strings.ToLower(m))
	if p.mode != modeNETASCII && p.mode != modeOCTET {
		return errMode
	}

	p.options, err = readOptions(b)
	return err
}

func (p *packetXRQ) Write(b *bytes.Buffer) error {
	var err error
	if err = writeChunk(b, string(p.filename)); err != nil {
		return err
	}
	if err = writeChunk(b, string(p.mode)); err != nil {
		return err
	}
	if err = writeOptions(b, p.options); err != nil {
		return err
	}
	return nil
}

type packetRRQ struct {
	packetXRQ
}

type packetWRQ struct {
	packetXRQ
}

type packetDATA struct {
	blockNr uint16
	data    []byte
}

func (p *packetDATA) Read(b *bytes.Buffer) error {
	err := binary.Read(b, binary.BigEndian, &p.blockNr)
	if err != nil {
		return err
	}

	p.data = make([]byte, b.Len())
	_, err = b.Read(p.data)
	return err
}

func (p *packetDATA) Write(b *bytes.Buffer) error {
	err := binary.Write(b, binary.BigEndian, p.blockNr)
	if err != nil {
		return err
	}

	_, err = b.Write(p.data)
	return err
}

type packetACK struct {
	blockNr uint16
}

func (p *packetACK) Read(b *bytes.Buffer) error {
	return binary.Read(b, binary.BigEndian, &p.blockNr)
}

func (p *packetACK) Write(b *bytes.Buffer) error {
	return binary.Write(b, binary.BigEndian, p.blockNr)
}

type packetERROR struct {
	errorCode    uint16
	errorMessage string
}

func (p *packetERROR) Read(b *bytes.Buffer) error {
	err := binary.Read(b, binary.BigEndian, &p.errorCode)
	if err != nil {
		return err
	}

	p.errorMessage, err = readChunk(b)
	return err
}

func (p *packetERROR) Write(b *bytes.Buffer) error {
	err := binary.Write(b, binary.BigEndian, p.errorCode)
	if err != nil {
		return err
	}

	return writeChunk(b, p.errorMessage)
}

type packetOACK struct {
	options map[string]string
}

func (p *packetOACK) Read(b *bytes.Buffer) error {
	o, err := readOptions(b)
	if err != nil {
		return err
	}

	p.options = o
	return nil
}

func (p *packetOACK) Write(b *bytes.Buffer) error {
	return writeOptions(b, p.options)
}

// packetFromWire reads the wire-level representation of a packet from buffer b into a packet.
func packetFromWire(b *bytes.Buffer) (packet, error) {
	var p packet
	var err error
	var opcode opcode

	if err = binary.Read(b, binary.BigEndian, &opcode); err != nil {
		return nil, err
	}

	switch opcode {
	case opcodeRRQ:
		p = &packetRRQ{}
	case opcodeWRQ:
		p = &packetWRQ{}
	case opcodeDATA:
		p = &packetDATA{}
	case opcodeACK:
		p = &packetACK{}
	case opcodeERROR:
		p = &packetERROR{}
	case opcodeOACK:
		p = &packetOACK{}
	default:
		return nil, errOpcode
	}

	if err = p.Read(b); err != nil {
		return nil, err
	}

	return p, nil
}

// packetToWire writes the wire-level representation of packet p to buffer b.
func packetToWire(p packet, b *bytes.Buffer) error {
	var err error
	var opcode opcode

	switch p.(type) {
	case *packetRRQ:
		opcode = opcodeRRQ
	case *packetWRQ:
		opcode = opcodeWRQ
	case *packetDATA:
		opcode = opcodeDATA
	case *packetACK:
		opcode = opcodeACK
	case *packetERROR:
		opcode = opcodeERROR
	case *packetOACK:
		opcode = opcodeOACK
	default:
		panic("unknown packet")
	}

	if err = binary.Write(b, binary.BigEndian, uint16(opcode)); err != nil {
		return err
	}

	if err = p.Write(b); err != nil {
		return err
	}

	return nil
}
