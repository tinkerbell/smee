package syslog

import (
	"flag"
	"net"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/tinkerbell/boots/conf"
)

var syslogAddr = conf.SyslogBind

func init() {
	flag.StringVar(&syslogAddr, "syslog-addr", syslogAddr, "IP and port to listen on for syslog messages.")
}

var syslogMessagePool = sync.Pool{
	New: func() interface{} { return new(message) },
}

type Receiver struct {
	c *net.UDPConn

	parse chan *message

	done chan struct{}
	err  error
}

func StartReceiver(parsers int) (*Receiver, error) {
	if parsers < 1 {
		parsers = 1
	}

	addr, err := net.ResolveUDPAddr("udp4", syslogAddr)
	if err != nil {
		return nil, errors.Wrap(err, "resolve syslog udp listen address")
	}

	c, err := net.ListenUDP("udp4", addr)
	if err != nil {
		return nil, errors.Wrap(err, "listen on syslog udp address")
	}

	s := &Receiver{
		c:     c,
		parse: make(chan *message, parsers),
		done:  make(chan struct{}),
	}

	for i := 0; i < parsers; i++ {
		go s.runParser()
	}
	go s.run()

	return s, nil
}

func (r *Receiver) Done() <-chan struct{} {
	return r.done
}

func (r *Receiver) Err() error {
	return r.err
}

func (r *Receiver) cleanup() {
	r.c.Close()

	close(r.parse)
	close(r.done)
}

func (r *Receiver) run() {
	var msg *message
	defer func() {
		if msg != nil {
			syslogMessagePool.Put(msg)
		}
		r.cleanup()
	}()

	for {
		if msg == nil {
			msg = syslogMessagePool.Get().(*message)
		}
		n, from, err := r.c.ReadFromUDP(msg.buf[:])
		if err != nil {
			err = errors.Wrap(err, "error reading udp message")
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				sysloglog.Error(err)

				continue
			}
			r.err = err

			return
		}
		msg.time = time.Now().UTC()
		msg.host = from.IP
		msg.size = n
		r.parse <- msg
		msg = nil
	}
}

func (r *Receiver) runParser() {
	for m := range r.parse {
		if m.parse() {
			if m.Severity() == DEBUG {
				sysloglog.Debug(m)
			} else {
				sysloglog.Info(m)
			}
		} else {
			sysloglog.Debug(m)
		}
		m.reset()
		syslogMessagePool.Put(m)
	}
}
