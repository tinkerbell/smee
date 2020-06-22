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

import (
	"bytes"
	"net"
	"sync"
	"time"

	"golang.org/x/net/ipv4"
)

type controlMessage struct {
	*ipv4.ControlMessage
}

func (c controlMessage) LocalAddr() net.Addr {
	return &net.IPAddr{IP: c.ControlMessage.Dst}
}

func (c controlMessage) RemoteAddr() net.Addr {
	return &net.IPAddr{IP: c.ControlMessage.Src}
}

type zeroConn struct{}

func (c zeroConn) LocalAddr() net.Addr {
	return &net.IPAddr{IP: net.IPv4zero}
}

func (c zeroConn) RemoteAddr() net.Addr {
	return &net.IPAddr{IP: net.IPv4zero}
}

func newZeroConn() Conn {
	return zeroConn{}
}

// ZeroConn can be used as a placeholder if otherwise not known.
var ZeroConn = newZeroConn()

type packetReaderImpl struct {
	ch <-chan []byte
}

func (p *packetReaderImpl) read(timeout time.Duration) (packet, error) {
	if timeout == 0 {
		select {
		case buf := <-p.ch:
			return packetFromWire(bytes.NewBuffer(buf))
		default:
			return nil, ErrTimeout
		}
	}
	select {
	case buf := <-p.ch:
		return packetFromWire(bytes.NewBuffer(buf))
	case <-time.After(timeout):
		return nil, ErrTimeout
	}
}

type packetWriterImpl struct {
	net.PacketConn

	addr net.Addr
	b    bytes.Buffer
}

func (p *packetWriterImpl) write(x packet) error {
	p.b.Reset()

	err := packetToWire(x, &p.b)
	if err != nil {
		return err
	}

	_, err = p.PacketConn.WriteTo(p.b.Bytes(), p.addr)
	return err
}

type syncPacketConn struct {
	net.PacketConn
	sync.Mutex
}

func (s *syncPacketConn) WriteTo(b []byte, addr net.Addr) (int, error) {
	s.Lock()
	n, err := s.PacketConn.WriteTo(b, addr)
	s.Unlock()
	return n, err
}

func Serve(l net.PacketConn, h Handler) error {
	ipv4pc := ipv4.NewPacketConn(l)
	flags := ipv4.FlagSrc | ipv4.FlagDst | ipv4.FlagInterface
	if err := ipv4pc.SetControlMessage(flags, true); err != nil {
		return err
	}

	var (
		mu    sync.Mutex
		table = make(map[string]chan []byte)
		buf   = make([]byte, 65536)
	)
	for {
		n, cm, addr, err := ipv4pc.ReadFrom(buf)
		if err != nil {
			return err
		}

		// Ignore packet without control message.
		if cm == nil {
			continue
		}

		// Ownership of this buffer is transferred to the goroutine for the peer
		// address, so we need to make a copy before handing it off.
		b := make([]byte, n)
		copy(b, buf[:n])

		mu.Lock()

		ch, ok := table[addr.String()]
		if !ok {
			ch = make(chan []byte, 10)
			ch <- b
			table[addr.String()] = ch

			// Packet reader for client
			r := &packetReaderImpl{
				ch: ch,
			}

			// Packet writer for client
			w := &packetWriterImpl{
				PacketConn: &syncPacketConn{
					PacketConn: l,
				},
				addr: addr,
			}

			// Kick off a serve loop for this peer address.
			go func() {
				// A client MAY reuse its socket for more than one request.
				// Therefore, continue running the serve loop until there are no more
				// inbound packets on the channel for this peer address.
				for stop := false; !stop; {
					serve(controlMessage{cm}, r, w, h)

					mu.Lock()
					if len(ch) == 0 {
						delete(table, addr.String())
						stop = true
					}
					mu.Unlock()
				}
			}()
		} else {
			select {
			case ch <- b:
			default:
				// Drop packet on the floor if we can't handle it
			}
		}

		// Unlock after sending buffer so that other routines can reliably check
		// and use the length of a channel while holding the lock.
		mu.Unlock()
	}
}

func ListenAndServe(addr string, h Handler) error {
	if addr == "" {
		addr = ":69"
	}
	l, err := net.ListenPacket("udp4", addr)
	if err != nil {
		return err
	}
	defer l.Close() // Should I not do this?

	return Serve(l, h)
}
